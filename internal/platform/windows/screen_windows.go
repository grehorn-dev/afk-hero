//go:build windows

package windows

import (
	"afk-hero/internal/domain"
	"fmt"
	"unsafe"
)

// ScreenProvider implements platform.ScreenProvider using Windows API.
type ScreenProvider struct{}

// NewScreenProvider creates a new Windows screen provider.
func NewScreenProvider() *ScreenProvider {
	return &ScreenProvider{}
}

// WorkArea returns the work area of the monitor containing the given point.
func (sp *ScreenProvider) WorkArea(at domain.Point) (domain.Rect, error) {
	hMonitor, _, _ := procMonitorFromPoint.Call(
		packPoint(at.X, at.Y),
		uintptr(monitorDefaultToNearest),
	)
	if hMonitor == 0 {
		return sp.PrimaryWorkArea()
	}

	var mi monitorInfo
	mi.CbSize = uint32(unsafe.Sizeof(mi))
	r1, _, err := procGetMonitorInfoW.Call(hMonitor, uintptr(unsafe.Pointer(&mi)))
	if r1 == 0 {
		return domain.Rect{}, fmt.Errorf("GetMonitorInfoW: %w", err)
	}

	return domain.Rect{
		Left:   int(mi.RcWork.Left),
		Top:    int(mi.RcWork.Top),
		Right:  int(mi.RcWork.Right),
		Bottom: int(mi.RcWork.Bottom),
	}, nil
}

// PrimaryWorkArea returns the primary monitor's work area.
func (sp *ScreenProvider) PrimaryWorkArea() (domain.Rect, error) {
	cx, _, _ := procGetSystemMetrics.Call(uintptr(smCxScreen))
	cy, _, _ := procGetSystemMetrics.Call(uintptr(smCyScreen))

	w := int(int32(cx))
	h := int(int32(cy))

	if w == 0 || h == 0 {
		return domain.Rect{}, fmt.Errorf("GetSystemMetrics returned zero dimensions")
	}

	// Try to get the actual work area from the primary monitor
	var mi monitorInfo
	mi.CbSize = uint32(unsafe.Sizeof(mi))
	hMonitor, _, _ := procMonitorFromPoint.Call(packPoint(0, 0), uintptr(monitorDefaultToNearest))
	if hMonitor != 0 {
		r1, _, _ := procGetMonitorInfoW.Call(hMonitor, uintptr(unsafe.Pointer(&mi)))
		if r1 != 0 {
			return domain.Rect{
				Left:   int(mi.RcWork.Left),
				Top:    int(mi.RcWork.Top),
				Right:  int(mi.RcWork.Right),
				Bottom: int(mi.RcWork.Bottom),
			}, nil
		}
	}

	// Fallback to screen dimensions
	return domain.Rect{Left: 0, Top: 0, Right: w, Bottom: h}, nil
}

func packPoint(x, y int) uintptr {
	return uintptr(uint32(int32(x))) | uintptr(uint64(uint32(int32(y)))<<32)
}
