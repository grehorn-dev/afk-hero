//go:build windows

package windows

import (
	"afk-hero/internal/domain"
	"testing"
)

func TestPackPoint(t *testing.T) {
	packed := packPoint(-10, 25)

	gotX := int32(uint32(packed))
	gotY := int32(uint32(uint64(packed) >> 32))
	if gotX != -10 || gotY != 25 {
		t.Fatalf("expected packed point (-10,25), got (%d,%d)", gotX, gotY)
	}
}

func TestBuildRelativeMouseInput(t *testing.T) {
	input := buildRelativeMouseInput(domain.Point{X: -3, Y: 7}, AppInputSignature)

	if input.Type != inputMouse {
		t.Fatalf("expected input type %d, got %d", inputMouse, input.Type)
	}
	if input.Dx != -3 || input.Dy != 7 {
		t.Fatalf("expected relative delta (-3,7), got (%d,%d)", input.Dx, input.Dy)
	}
	if input.DwFlags != mouseeventfMove {
		t.Fatalf("expected MOVE flag only, got 0x%x", input.DwFlags)
	}
	if input.DwExtraInfo != AppInputSignature {
		t.Fatalf("expected extra info 0x%x, got 0x%x", AppInputSignature, input.DwExtraInfo)
	}
}

func TestRectMatchesMonitor(t *testing.T) {
	monitor := domain.Rect{Left: 0, Top: 0, Right: 1920, Bottom: 1080}

	if !rectMatchesMonitor(monitor, monitor, 0) {
		t.Fatal("expected exact monitor-sized window to match")
	}

	borderless := domain.Rect{Left: 5, Top: 4, Right: 1914, Bottom: 1076}
	if !rectMatchesMonitor(borderless, monitor, autoToleranceRatio) {
		t.Fatal("expected near fullscreen borderless window to match within tolerance")
	}

	windowed := domain.Rect{Left: 120, Top: 80, Right: 1600, Bottom: 900}
	if rectMatchesMonitor(windowed, monitor, autoToleranceRatio) {
		t.Fatal("did not expect normal windowed rect to match monitor tolerance")
	}
}

func TestIsUserFacingWindow(t *testing.T) {
	rect := domain.Rect{Left: 100, Top: 100, Right: 1200, Bottom: 900}

	if !isUserFacingWindow(true, 0, false, false, rect) {
		t.Fatal("expected regular top-level window to be included")
	}
	if isUserFacingWindow(false, 0, false, false, rect) {
		t.Fatal("did not expect invisible window to be included")
	}
	if isUserFacingWindow(true, wsExToolWindow, false, false, rect) {
		t.Fatal("did not expect tool window to be included")
	}
	if isUserFacingWindow(true, 0, true, false, rect) {
		t.Fatal("did not expect owned window without APPWINDOW to be included")
	}
	if !isUserFacingWindow(true, wsExAppWindow, true, false, rect) {
		t.Fatal("expected APPWINDOW-owned window to be included")
	}
	if isUserFacingWindow(true, wsExNoActivate, false, false, rect) {
		t.Fatal("did not expect no-activate window without APPWINDOW to be included")
	}
	if isUserFacingWindow(true, 0, false, true, rect) {
		t.Fatal("did not expect cloaked window to be included")
	}
	if isUserFacingWindow(true, 0, false, false, domain.Rect{}) {
		t.Fatal("did not expect zero-sized window to be included")
	}
}

func TestIsCurrentProcessWindow(t *testing.T) {
	if !isCurrentProcessWindow(currentProcessID) {
		t.Fatal("expected current process window to be filtered out")
	}

	if isCurrentProcessWindow(currentProcessID ^ 1) {
		t.Fatal("did not expect a different process ID to be filtered out")
	}

	if isCurrentProcessWindow(0) {
		t.Fatal("did not expect zero process ID to be treated as current process")
	}
}

func TestScoreLiveAutoCandidate(t *testing.T) {
	monitor := domain.Rect{Left: 0, Top: 0, Right: 1920, Bottom: 1080}
	fullscreen := domain.Rect{Left: 2, Top: 1, Right: 1918, Bottom: 1079}

	if got := scoreLiveAutoCandidate(wsCaption|wsThickframe, fullscreen, monitor); got != 30 {
		t.Fatalf("expected fullscreen-style window to score 30, got %d", got)
	}

	if got := scoreLiveAutoCandidate(wsPopup, fullscreen, monitor); got != 20 {
		t.Fatalf("expected borderless fullscreen window to score 20, got %d", got)
	}

	windowed := domain.Rect{Left: 150, Top: 120, Right: 1700, Bottom: 920}
	if got := scoreLiveAutoCandidate(wsCaption|wsThickframe, windowed, monitor); got != 0 {
		t.Fatalf("expected regular windowed rect to score 0, got %d", got)
	}
}

func TestScoreSuspendedAutoCandidate(t *testing.T) {
	monitor := domain.Rect{Left: 0, Top: 0, Right: 1920, Bottom: 1080}
	fullscreen := domain.Rect{Left: 0, Top: 0, Right: 1920, Bottom: 1080}

	if got := scoreSuspendedAutoCandidate(wsVisible, fullscreen, monitor); got != 10 {
		t.Fatalf("expected suspended captionless fullscreen window to score 10, got %d", got)
	}

	if got := scoreSuspendedAutoCandidate(wsPopup|wsVisible, fullscreen, monitor); got != 10 {
		t.Fatalf("expected suspended exclusive fullscreen window to score 10, got %d", got)
	}

	if got := scoreSuspendedAutoCandidate(wsCaption|wsThickframe|wsVisible, fullscreen, monitor); got != 0 {
		t.Fatalf("expected captioned minimized window to be rejected, got %d", got)
	}

	windowed := domain.Rect{Left: 100, Top: 100, Right: 1600, Bottom: 900}
	if got := scoreSuspendedAutoCandidate(wsPopup|wsVisible, windowed, monitor); got != 0 {
		t.Fatalf("expected non-fullscreen restored rect to be rejected, got %d", got)
	}
}

func TestMergeAutoCandidate(t *testing.T) {
	windows := []domain.WindowInfo{
		{
			ID:          1,
			Title:       "Settings",
			ProcessName: "app.exe",
			ClassName:   "MainWindow",
		},
	}

	candidate := autoCandidate{
		window: domain.WindowInfo{
			ID:          2,
			Title:       "Game",
			ProcessName: "game.exe",
			ClassName:   "HordeMode",
		},
		priority: 20,
	}

	merged := mergeAutoCandidate(windows, candidate)
	if len(merged) != 2 {
		t.Fatalf("expected auto candidate to be appended, got %d windows", len(merged))
	}
	if !merged[1].AutoCandidate {
		t.Fatal("expected appended auto candidate to be marked")
	}

	updated := mergeAutoCandidate(merged, autoCandidate{
		window: domain.WindowInfo{
			ID:          3,
			Title:       "Game Updated",
			ProcessName: "game.exe",
			ClassName:   "HordeMode",
		},
		priority: 30,
	})
	if len(updated) != 2 {
		t.Fatalf("expected matching auto candidate to be merged in place, got %d windows", len(updated))
	}
	if !updated[1].AutoCandidate {
		t.Fatal("expected matching auto candidate to remain marked")
	}
}

func TestIsSearchableWindow(t *testing.T) {
	rect := domain.Rect{Left: 0, Top: 0, Right: 1920, Bottom: 1080}

	if !isSearchableWindow(true, false, 0, false, false, rect) {
		t.Fatal("expected fullscreen-sized window to be considered searchable")
	}
	if !isSearchableWindow(false, true, 0, false, false, rect) {
		t.Fatal("expected iconic fullscreen-sized window to be considered searchable")
	}
	if isSearchableWindow(true, false, wsExToolWindow, false, false, rect) {
		t.Fatal("did not expect tool window to be searchable")
	}
	if isSearchableWindow(true, false, wsExNoActivate, false, false, rect) {
		t.Fatal("did not expect no-activate window without APPWINDOW to be searchable")
	}
	if isSearchableWindow(false, false, 0, false, false, rect) {
		t.Fatal("did not expect non-visible, non-iconic window to be searchable")
	}
	if isSearchableWindow(true, false, 0, false, true, rect) {
		t.Fatal("did not expect cloaked window to be searchable")
	}
	if isSearchableWindow(true, false, 0, false, false, domain.Rect{}) {
		t.Fatal("did not expect zero-sized window to be searchable")
	}
}

func TestAutoProcessBlocklist(t *testing.T) {
	blocklist := makeAutoProcessBlocklist([]string{
		` "Chrome.exe" `,
		`C:\Program Files\Mozilla Firefox\firefox.exe`,
		"chrome.exe",
	})

	if len(blocklist) != 2 {
		t.Fatalf("expected 2 distinct blocked processes, got %d", len(blocklist))
	}
	if !isBlockedAutoProcess("chrome.exe", blocklist) {
		t.Fatal("expected chrome.exe to be blocked")
	}
	if !isBlockedAutoProcess("FIREFOX.EXE", blocklist) {
		t.Fatal("expected firefox.exe to be blocked case-insensitively")
	}
	if isBlockedAutoProcess("game.exe", blocklist) {
		t.Fatal("did not expect game.exe to be blocked")
	}
}

func TestWindowEnumerationRegistry(t *testing.T) {
	first := &windowEnumeration{}
	firstToken := registerWindowEnumeration(first)
	if firstToken == 0 {
		t.Fatal("expected non-zero token")
	}
	if got := lookupWindowEnumeration(firstToken); got != first {
		t.Fatal("expected registered collector to be returned")
	}

	second := &windowEnumeration{}
	secondToken := registerWindowEnumeration(second)
	if secondToken == 0 {
		t.Fatal("expected non-zero second token")
	}
	if secondToken == firstToken {
		t.Fatal("expected unique tokens for concurrent enumerations")
	}
	if got := lookupWindowEnumeration(secondToken); got != second {
		t.Fatal("expected second collector to be returned")
	}

	unregisterWindowEnumeration(firstToken)
	if got := lookupWindowEnumeration(firstToken); got != nil {
		t.Fatal("expected unregistered collector to be removed")
	}

	unregisterWindowEnumeration(secondToken)
	if got := lookupWindowEnumeration(secondToken); got != nil {
		t.Fatal("expected second collector to be removed")
	}
}
