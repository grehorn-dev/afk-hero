//go:build windows

package windows

import "syscall"

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	dwmapi   = syscall.NewLazyDLL("dwmapi.dll")

	procGetLastInputInfo           = user32.NewProc("GetLastInputInfo")
	procGetTickCount64             = kernel32.NewProc("GetTickCount64")
	procSendInput                  = user32.NewProc("SendInput")
	procGetCursorPos               = user32.NewProc("GetCursorPos")
	procGetSystemMetrics           = user32.NewProc("GetSystemMetrics")
	procMonitorFromPoint           = user32.NewProc("MonitorFromPoint")
	procMonitorFromWindow          = user32.NewProc("MonitorFromWindow")
	procGetMonitorInfoW            = user32.NewProc("GetMonitorInfoW")
	procEnumWindows                = user32.NewProc("EnumWindows")
	procGetWindowTextW             = user32.NewProc("GetWindowTextW")
	procGetWindowTextLengthW       = user32.NewProc("GetWindowTextLengthW")
	procGetClassNameW              = user32.NewProc("GetClassNameW")
	procGetWindow                  = user32.NewProc("GetWindow")
	procGetWindowPlacement         = user32.NewProc("GetWindowPlacement")
	procIsWindowVisible            = user32.NewProc("IsWindowVisible")
	procIsWindow                   = user32.NewProc("IsWindow")
	procIsIconic                   = user32.NewProc("IsIconic")
	procShowWindow                 = user32.NewProc("ShowWindow")
	procSetForegroundWindow        = user32.NewProc("SetForegroundWindow")
	procBringWindowToTop           = user32.NewProc("BringWindowToTop")
	procGetForegroundWindow        = user32.NewProc("GetForegroundWindow")
	procGetWindowThreadProcessID   = user32.NewProc("GetWindowThreadProcessId")
	procAttachThreadInput          = user32.NewProc("AttachThreadInput")
	procGetWindowLongW             = user32.NewProc("GetWindowLongW")
	procGetWindowRect              = user32.NewProc("GetWindowRect")
	procGetCurrentThreadID         = kernel32.NewProc("GetCurrentThreadId")
	procOpenProcess                = kernel32.NewProc("OpenProcess")
	procQueryFullProcessImageNameW = kernel32.NewProc("QueryFullProcessImageNameW")
	procCloseHandle                = kernel32.NewProc("CloseHandle")
	procDwmGetWindowAttribute      = dwmapi.NewProc("DwmGetWindowAttribute")
)

const (
	inputMouse      = 0
	mouseeventfMove = 0x0001

	smCxScreen = 0
	smCyScreen = 1

	monitorDefaultToNearest = 2

	swRestore = 9

	gwlStyle       = -16
	gwlExStyle     = -20
	wsVisible      = 0x10000000
	wsPopup        = 0x80000000
	wsCaption      = 0x00C00000
	wsThickframe   = 0x00040000
	wsExAppWindow  = 0x00040000
	wsExToolWindow = 0x00000080
	wsExNoActivate = 0x08000000

	gwOwner = 4

	dwmwaCloaked = 14

	processQueryLimitedInformation = 0x1000
)

// LASTINPUTINFO structure.
type lastInputInfo struct {
	CbSize uint32
	DwTime uint32
}

// POINT structure.
type winPoint struct {
	X, Y int32
}

// RECT structure.
type winRect struct {
	Left, Top, Right, Bottom int32
}

// MONITORINFO structure.
type monitorInfo struct {
	CbSize    uint32
	RcMonitor winRect
	RcWork    winRect
	DwFlags   uint32
}

// WINDOWPLACEMENT structure.
type windowPlacement struct {
	Length           uint32
	Flags            uint32
	ShowCmd          uint32
	PtMinPosition    winPoint
	PtMaxPosition    winPoint
	RcNormalPosition winRect
}

// INPUT structure for SendInput.
// On 64-bit Windows, the union inside INPUT is 8-byte aligned (due to ULONG_PTR),
// and dwExtraInfo within MOUSEINPUT is also 8-byte aligned.
// Explicit padding fields match the C struct layout (total 40 bytes).
type mouseInput struct {
	Type        uint32
	_           uint32 // padding: align union to 8-byte boundary
	Dx          int32
	Dy          int32
	MouseData   uint32
	DwFlags     uint32
	Time        uint32
	_           uint32 // padding: align dwExtraInfo to 8-byte boundary
	DwExtraInfo uintptr
}

const inputSize = 40 // sizeof(INPUT) on 64-bit Windows
