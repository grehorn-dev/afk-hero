//go:build windows

package windows

import (
	"afk-hero/internal/apperrors"
	"afk-hero/internal/domain"
	"afk-hero/internal/logging"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"syscall"
	"unicode/utf16"
	"unsafe"
)

const (
	classNameBufferSize   = 256
	processPathBufferSize = 1024
	autoToleranceRatio    = 0.01
)

var (
	currentProcessID        = uint32(os.Getpid())
	enumWindowsCallback     = syscall.NewCallback(enumWindowsProc)
	enumWindowsSequence     uint64
	enumWindowsCollectorsMu sync.Mutex
	enumWindowsCollectors   = map[uintptr]*windowEnumeration{}
)

type autoCandidate struct {
	window   domain.WindowInfo
	priority int
}

// windowEnumeration keeps the snapshots collected for a single EnumWindows call.
type windowEnumeration struct {
	windows []windowSnapshot
}

type windowSnapshot struct {
	info      domain.WindowInfo
	processID uint32
	rect      domain.Rect
	visible   bool
	iconic    bool
	hasOwner  bool
	cloaked   bool
	style     uintptr
	exStyle   uintptr
}

// WindowManager implements platform.WindowManager using Windows API.
type WindowManager struct{}

// NewWindowManager creates a new Windows window manager.
func NewWindowManager() *WindowManager {
	return &WindowManager{}
}

// ListWindows enumerates visible top-level windows with stable identification fields.
// Auto-target resolution is a best-effort enhancement: when it fails (no
// qualifying fullscreen candidate), the plain user-facing window list is
// still returned so the settings UI stays usable.
func (wm *WindowManager) ListWindows(autoProcessBlacklist []string) ([]domain.WindowInfo, error) {
	snapshots, err := wm.enumerateWindows()
	if err != nil {
		return nil, err
	}

	windows := userFacingWindows(snapshots)
	autoTarget, autoErr := resolveAutoTargetFromSnapshots(snapshots, autoProcessBlacklist)
	if autoErr != nil {
		// Auto-target resolution failure is informational, not fatal: the
		// plain user-facing list is still usable by the settings UI.
		return windows, nil //nolint:nilerr // best-effort enhancement
	}

	return mergeAutoCandidate(windows, autoTarget), nil
}

// ActivateWindow brings the specified window to the foreground.
func (wm *WindowManager) ActivateWindow(id uintptr) error {
	status, err := wm.WindowStatus(id)
	if err != nil {
		logging.Get().Debug("activation preflight failed", "window_id", id, "error", err)
		return err
	}
	if !status.Exists {
		logging.Get().Debug("activation target no longer exists", "window_id", id)
		return apperrors.ErrWindowNotFound
	}

	logging.Get().Debug(
		"activation attempt started",
		"window_id", id,
		"minimized", status.Minimized,
		"foreground", status.Foreground,
	)

	if iconic, _, _ := procIsIconic.Call(id); iconic != 0 {
		logging.Get().Debug("restoring minimized activation target", "window_id", id)
		_, _, _ = procShowWindow.Call(id, uintptr(swRestore))
	}

	fgHwnd, _, _ := procGetForegroundWindow.Call()
	var fgThread uintptr
	if fgHwnd != 0 {
		fgThread, _, _ = procGetWindowThreadProcessID.Call(fgHwnd, 0)
	}

	curThread, _, _ := procGetCurrentThreadID.Call()
	if fgThread != 0 && fgThread != curThread {
		_, _, _ = procAttachThreadInput.Call(curThread, fgThread, 1)
		defer func() {
			_, _, _ = procAttachThreadInput.Call(curThread, fgThread, 0)
		}()
	}

	r1, _, err := procSetForegroundWindow.Call(id)
	if r1 == 0 {
		_, _, _ = procBringWindowToTop.Call(id)
		logging.Get().Debug(
			"SetForegroundWindow failed during activation attempt",
			"window_id", id,
			"foreground_id", fgHwnd,
			"error", err,
		)
		return fmt.Errorf("SetForegroundWindow: %w", err)
	}

	logging.Get().Debug(
		"activation attempt finished",
		"window_id", id,
		"previous_foreground_id", fgHwnd,
	)
	return nil
}

// FindWindowByFingerprint resolves a stable process/class matcher to a live window.
func (wm *WindowManager) FindWindowByFingerprint(fingerprint domain.WindowFingerprint) (domain.WindowInfo, error) {
	if !fingerprint.Valid() {
		return domain.WindowInfo{}, fmt.Errorf("window fingerprint is incomplete")
	}

	snapshots, err := wm.enumerateWindows()
	if err != nil {
		return domain.WindowInfo{}, err
	}

	for _, window := range searchableWindows(snapshots) {
		if fingerprint.Matches(window.info) {
			return window.info, nil
		}
	}

	return domain.WindowInfo{}, apperrors.ErrWindowNotFound
}

// ResolveAutoTarget finds the current best-effort target for Auto activation mode.
func (wm *WindowManager) ResolveAutoTarget(autoProcessBlacklist []string) (domain.WindowInfo, error) {
	snapshots, err := wm.enumerateWindows()
	if err != nil {
		return domain.WindowInfo{}, err
	}

	candidate, err := resolveAutoTargetFromSnapshots(snapshots, autoProcessBlacklist)
	if err != nil {
		return domain.WindowInfo{}, err
	}

	return candidate.window, nil
}

// AutoTargetInfo reports whether the specified window handle still qualifies as an Auto target.
func (wm *WindowManager) AutoTargetInfo(id uintptr, autoProcessBlacklist []string) (domain.WindowInfo, error) {
	if id == 0 {
		return domain.WindowInfo{}, fmt.Errorf("window id must be non-zero")
	}

	snapshot, ok := buildWindowSnapshot(id)
	if !ok {
		return domain.WindowInfo{}, apperrors.ErrWindowNotFound
	}
	if isBlockedAutoProcess(snapshot.info.ProcessName, makeAutoProcessBlocklist(autoProcessBlacklist)) {
		return domain.WindowInfo{}, apperrors.ErrWindowNotFound
	}

	candidate, ok := describeAutoCandidate(snapshot)
	if !ok {
		return domain.WindowInfo{}, apperrors.ErrWindowNotFound
	}

	candidate.window.AutoCandidate = true
	return candidate.window, nil
}

// WindowInfo returns the current identity fields for the specified window handle.
func (wm *WindowManager) WindowInfo(id uintptr) (domain.WindowInfo, error) {
	if id == 0 {
		return domain.WindowInfo{}, fmt.Errorf("window id must be non-zero")
	}

	snapshot, ok := buildWindowSnapshot(id)
	if !ok {
		return domain.WindowInfo{}, apperrors.ErrWindowNotFound
	}

	return snapshot.info, nil
}

// WindowStatus reports whether the window exists and whether it is minimized/foreground.
func (wm *WindowManager) WindowStatus(id uintptr) (domain.WindowActivationStatus, error) {
	if id == 0 {
		return domain.WindowActivationStatus{}, fmt.Errorf("window id must be non-zero")
	}

	exists, _, _ := procIsWindow.Call(id)
	if exists == 0 {
		return domain.WindowActivationStatus{Exists: false}, nil
	}

	iconic, _, _ := procIsIconic.Call(id)
	foreground, _, _ := procGetForegroundWindow.Call()

	return domain.WindowActivationStatus{
		Exists:     true,
		Minimized:  iconic != 0,
		Foreground: foreground == id,
	}, nil
}

func (wm *WindowManager) enumerateWindows() ([]windowSnapshot, error) {
	collector := &windowEnumeration{}
	token := registerWindowEnumeration(collector)
	defer unregisterWindowEnumeration(token)

	r1, _, err := procEnumWindows.Call(enumWindowsCallback, token)
	if r1 == 0 {
		return nil, fmt.Errorf("EnumWindows: %w", err)
	}

	return collector.windows, nil
}

func registerWindowEnumeration(collector *windowEnumeration) uintptr {
	token := uintptr(atomic.AddUint64(&enumWindowsSequence, 1))

	enumWindowsCollectorsMu.Lock()
	enumWindowsCollectors[token] = collector
	enumWindowsCollectorsMu.Unlock()

	return token
}

func unregisterWindowEnumeration(token uintptr) {
	enumWindowsCollectorsMu.Lock()
	delete(enumWindowsCollectors, token)
	enumWindowsCollectorsMu.Unlock()
}

func lookupWindowEnumeration(token uintptr) *windowEnumeration {
	enumWindowsCollectorsMu.Lock()
	collector := enumWindowsCollectors[token]
	enumWindowsCollectorsMu.Unlock()

	return collector
}

func enumWindowsProc(hwnd uintptr, lParam uintptr) uintptr {
	collector := lookupWindowEnumeration(lParam)
	if collector == nil {
		return 0
	}

	info, ok := buildWindowSnapshot(hwnd)
	if !ok {
		return 1
	}

	collector.windows = append(collector.windows, info)
	return 1
}

func buildWindowSnapshot(hwnd uintptr) (windowSnapshot, bool) {
	rect, ok := getWindowRect(hwnd)
	if !ok {
		return windowSnapshot{}, false
	}

	className := getWindowClassName(hwnd)
	processID, processName := getWindowProcessIdentity(hwnd)
	if className == "" || processName == "" || isCurrentProcessWindow(processID) {
		return windowSnapshot{}, false
	}

	info := domain.WindowInfo{
		ID:          hwnd,
		Title:       getWindowText(hwnd),
		ClassName:   className,
		ProcessName: processName,
	}

	visible, _, _ := procIsWindowVisible.Call(hwnd)
	iconic, _, _ := procIsIconic.Call(hwnd)
	exStyle := getWindowExStyle(hwnd)
	style, _, _ := procGetWindowLongW.Call(hwnd, uintptr(gwlStyle&0xFFFFFFFF))

	return windowSnapshot{
		info:      info,
		processID: processID,
		rect:      rect,
		visible:   visible != 0,
		iconic:    iconic != 0,
		hasOwner:  getWindowOwner(hwnd) != 0,
		cloaked:   isWindowCloaked(hwnd),
		style:     style,
		exStyle:   exStyle,
	}, true
}

func isCurrentProcessWindow(processID uint32) bool {
	return processID != 0 && processID == currentProcessID
}

func resolveAutoTargetFromSnapshots(windows []windowSnapshot, autoProcessBlacklist []string) (autoCandidate, error) {
	searchable := searchableWindows(windows)
	blockedProcesses := makeAutoProcessBlocklist(autoProcessBlacklist)

	var best autoCandidate
	found := false

	for _, window := range searchable {
		if isBlockedAutoProcess(window.info.ProcessName, blockedProcesses) {
			continue
		}

		candidate, ok := describeAutoCandidate(window)
		if !ok {
			continue
		}

		if !found || candidate.priority > best.priority {
			best = candidate
			found = true
		}
	}

	if !found {
		return autoCandidate{}, apperrors.ErrWindowNotFound
	}

	best.window.AutoCandidate = true
	return best, nil
}

func makeAutoProcessBlocklist(processes []string) map[string]struct{} {
	if len(processes) == 0 {
		return nil
	}

	blocklist := make(map[string]struct{}, len(processes))
	for _, process := range processes {
		name := domain.NormalizeProcessName(process)
		if name == "" {
			continue
		}

		blocklist[name] = struct{}{}
	}

	return blocklist
}

func isBlockedAutoProcess(processName string, blocklist map[string]struct{}) bool {
	if len(blocklist) == 0 {
		return false
	}

	_, blocked := blocklist[domain.NormalizeProcessName(processName)]
	return blocked
}

func userFacingWindows(windows []windowSnapshot) []domain.WindowInfo {
	result := make([]domain.WindowInfo, 0, len(windows))
	for _, window := range windows {
		if isUserFacingWindow(window.visible, window.exStyle, window.hasOwner, window.cloaked, window.rect) {
			result = append(result, window.info)
		}
	}

	return result
}

func searchableWindows(windows []windowSnapshot) []windowSnapshot {
	result := make([]windowSnapshot, 0, len(windows))
	for _, window := range windows {
		if isSearchableWindow(window.visible, window.iconic, window.exStyle, window.hasOwner, window.cloaked, window.rect) {
			result = append(result, window)
		}
	}

	return result
}

func describeAutoCandidate(window windowSnapshot) (autoCandidate, bool) {
	if candidate, ok := describeLiveAutoCandidate(window); ok {
		return candidate, true
	}

	return describeSuspendedAutoCandidate(window)
}

func describeLiveAutoCandidate(window windowSnapshot) (autoCandidate, bool) {
	monitorRect, ok := monitorRectForRect(window.rect)
	if !ok {
		return autoCandidate{}, false
	}

	priority := scoreLiveAutoCandidate(window.style, window.rect, monitorRect)
	if priority == 0 {
		return autoCandidate{}, false
	}

	return autoCandidate{
		window:   window.info,
		priority: priority,
	}, true
}

func describeSuspendedAutoCandidate(window windowSnapshot) (autoCandidate, bool) {
	if !window.iconic {
		return autoCandidate{}, false
	}

	placement, ok := getWindowPlacement(window.info.ID)
	if !ok {
		return autoCandidate{}, false
	}

	restoredRect := rectFromWinRect(placement.RcNormalPosition)
	if restoredRect.Width() <= 0 || restoredRect.Height() <= 0 {
		return autoCandidate{}, false
	}

	monitorRect, ok := monitorRectForWindowHandle(window.info.ID)
	if !ok {
		return autoCandidate{}, false
	}

	priority := scoreSuspendedAutoCandidate(window.style, restoredRect, monitorRect)
	if priority == 0 {
		return autoCandidate{}, false
	}

	return autoCandidate{
		window:   window.info,
		priority: priority,
	}, true
}

func isUserFacingWindow(visible bool, exStyle uintptr, hasOwner, cloaked bool, rect domain.Rect) bool {
	if rect.Width() <= 0 || rect.Height() <= 0 {
		return false
	}
	if !visible {
		return false
	}
	if cloaked {
		return false
	}
	if exStyle&wsExToolWindow != 0 {
		return false
	}

	appWindow := exStyle&wsExAppWindow != 0
	if hasOwner && !appWindow {
		return false
	}
	if exStyle&wsExNoActivate != 0 && !appWindow {
		return false
	}

	return true
}

func isSearchableWindow(visible, iconic bool, exStyle uintptr, hasOwner, cloaked bool, rect domain.Rect) bool {
	if rect.Width() <= 0 || rect.Height() <= 0 {
		return false
	}
	if !visible && !iconic {
		return false
	}
	if cloaked {
		return false
	}
	if exStyle&wsExToolWindow != 0 {
		return false
	}

	appWindow := exStyle&wsExAppWindow != 0
	if hasOwner && !appWindow {
		return false
	}
	if exStyle&wsExNoActivate != 0 && !appWindow {
		return false
	}

	return true
}

func mergeAutoCandidate(windows []domain.WindowInfo, candidate autoCandidate) []domain.WindowInfo {
	for i := range windows {
		if windows[i].Fingerprint() == candidate.window.Fingerprint() {
			windows[i].AutoCandidate = true
			if windows[i].Title == "" {
				windows[i].Title = candidate.window.Title
			}
			if windows[i].ID == 0 {
				windows[i].ID = candidate.window.ID
			}
			return windows
		}
	}

	candidate.window.AutoCandidate = true
	return append(windows, candidate.window)
}

func scoreLiveAutoCandidate(style uintptr, windowRect, monitorRect domain.Rect) int {
	if !rectMatchesMonitor(windowRect, monitorRect, autoToleranceRatio) {
		return 0
	}

	if isBorderlessStyle(style) {
		return 20
	}

	return 30
}

func scoreSuspendedAutoCandidate(style uintptr, restoredRect, monitorRect domain.Rect) int {
	if !isSuspendedFullscreenStyle(style) {
		return 0
	}
	if !rectMatchesMonitor(restoredRect, monitorRect, autoToleranceRatio) {
		return 0
	}

	return 10
}

func isBorderlessStyle(style uintptr) bool {
	return style&wsCaption == 0 && style&wsThickframe == 0
}

func isSuspendedFullscreenStyle(style uintptr) bool {
	return isBorderlessStyle(style)
}

func getWindowRect(hwnd uintptr) (domain.Rect, bool) {
	var rect winRect
	r1, _, _ := procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&rect)))
	if r1 == 0 {
		return domain.Rect{}, false
	}

	return rectFromWinRect(rect), true
}

func getWindowPlacement(hwnd uintptr) (windowPlacement, bool) {
	placement := windowPlacement{Length: uint32(unsafe.Sizeof(windowPlacement{}))}
	r1, _, _ := procGetWindowPlacement.Call(hwnd, uintptr(unsafe.Pointer(&placement)))
	if r1 == 0 {
		return windowPlacement{}, false
	}

	return placement, true
}

func rectFromWinRect(rect winRect) domain.Rect {
	return domain.Rect{
		Left:   int(rect.Left),
		Top:    int(rect.Top),
		Right:  int(rect.Right),
		Bottom: int(rect.Bottom),
	}
}

func monitorRectForRect(windowRect domain.Rect) (domain.Rect, bool) {
	centerX := windowRect.Left + windowRect.Width()/2
	centerY := windowRect.Top + windowRect.Height()/2

	hMonitor, _, _ := procMonitorFromPoint.Call(packPoint(centerX, centerY), uintptr(monitorDefaultToNearest))
	if hMonitor == 0 {
		return domain.Rect{}, false
	}

	return monitorRectFromHandle(hMonitor)
}

func monitorRectForWindowHandle(hwnd uintptr) (domain.Rect, bool) {
	hMonitor, _, _ := procMonitorFromWindow.Call(hwnd, uintptr(monitorDefaultToNearest))
	if hMonitor == 0 {
		return domain.Rect{}, false
	}

	return monitorRectFromHandle(hMonitor)
}

func monitorRectFromHandle(hMonitor uintptr) (domain.Rect, bool) {
	var mi monitorInfo
	mi.CbSize = uint32(unsafe.Sizeof(mi))
	r1, _, _ := procGetMonitorInfoW.Call(hMonitor, uintptr(unsafe.Pointer(&mi)))
	if r1 == 0 {
		return domain.Rect{}, false
	}

	return rectFromWinRect(mi.RcMonitor), true
}

func rectMatchesMonitor(windowRect, monitorRect domain.Rect, toleranceRatio float64) bool {
	widthTolerance := toleranceForDimension(monitorRect.Width(), toleranceRatio)
	heightTolerance := toleranceForDimension(monitorRect.Height(), toleranceRatio)

	return abs(windowRect.Left-monitorRect.Left) <= widthTolerance &&
		abs(windowRect.Right-monitorRect.Right) <= widthTolerance &&
		abs(windowRect.Top-monitorRect.Top) <= heightTolerance &&
		abs(windowRect.Bottom-monitorRect.Bottom) <= heightTolerance
}

func toleranceForDimension(size int, ratio float64) int {
	if ratio <= 0 {
		return 0
	}

	tolerance := int(float64(size) * ratio)
	if tolerance < 1 {
		return 1
	}
	return tolerance
}

func getWindowText(hwnd uintptr) string {
	textLen, _, _ := procGetWindowTextLengthW.Call(hwnd)
	if textLen == 0 {
		return ""
	}

	buf := make([]uint16, textLen+1)
	_, _, _ = procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), textLen+1)
	return string(utf16.Decode(buf[:textLen]))
}

func getWindowClassName(hwnd uintptr) string {
	buf := make([]uint16, classNameBufferSize)
	r1, _, _ := procGetClassNameW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if r1 == 0 {
		return ""
	}

	return syscall.UTF16ToString(buf[:r1])
}

func getWindowOwner(hwnd uintptr) uintptr {
	owner, _, _ := procGetWindow.Call(hwnd, gwOwner)
	return owner
}

func getWindowExStyle(hwnd uintptr) uintptr {
	exStyle, _, _ := procGetWindowLongW.Call(hwnd, uintptr(gwlExStyle&0xFFFFFFFF))
	return exStyle
}

func isWindowCloaked(hwnd uintptr) bool {
	var cloaked uint32
	hr, _, _ := procDwmGetWindowAttribute.Call(
		hwnd,
		uintptr(dwmwaCloaked),
		uintptr(unsafe.Pointer(&cloaked)),
		unsafe.Sizeof(cloaked),
	)
	if hr != 0 {
		return false
	}

	return cloaked != 0
}

func getWindowProcessIdentity(hwnd uintptr) (uint32, string) {
	var pid uint32
	_, _, _ = procGetWindowThreadProcessID.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	if pid == 0 {
		return 0, ""
	}

	handle, _, _ := procOpenProcess.Call(uintptr(processQueryLimitedInformation), 0, uintptr(pid))
	if handle == 0 {
		return 0, ""
	}
	defer func() {
		_, _, _ = procCloseHandle.Call(handle)
	}()

	buf := make([]uint16, processPathBufferSize)
	size := uint32(len(buf))
	r1, _, _ := procQueryFullProcessImageNameW.Call(
		handle,
		0,
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&size)),
	)
	if r1 == 0 || size == 0 {
		return 0, ""
	}

	fullPath := syscall.UTF16ToString(buf[:size])
	if fullPath == "" {
		return 0, ""
	}

	return pid, filepath.Base(fullPath)
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
