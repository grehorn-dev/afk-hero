//go:build windows

package windows

import (
	"afk-hero/internal/domain"
	"fmt"
	"unsafe"
)

// AppInputSignature is the dwExtraInfo value used to tag injected input.
const AppInputSignature uintptr = 0xAF4D0E12

// PointerController implements platform.PointerController using Windows API.
type PointerController struct{}

// NewPointerController creates a new Windows pointer controller.
func NewPointerController() *PointerController {
	return &PointerController{}
}

// GetPosition returns the current cursor position.
func (pc *PointerController) GetPosition() (domain.Point, error) {
	var pt winPoint
	r1, _, err := procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	if r1 == 0 {
		return domain.Point{}, fmt.Errorf("GetCursorPos: %w", err)
	}
	return domain.Point{X: int(pt.X), Y: int(pt.Y)}, nil
}

// MoveBy moves the cursor by a relative delta using SendInput. A zero
// delta is skipped outright so we do not burn a syscall per frame on idle
// ticks (the interpolator can legitimately emit (0,0) near rounded
// sub-pixel positions).
func (pc *PointerController) MoveBy(delta domain.Point, extraInfo uintptr) error {
	if delta.X == 0 && delta.Y == 0 {
		return nil
	}

	input := buildRelativeMouseInput(delta, extraInfo)

	r1, _, err := procSendInput.Call(
		1,
		uintptr(unsafe.Pointer(&input)),
		uintptr(inputSize),
	)
	if r1 == 0 {
		return fmt.Errorf("SendInput: %w", err)
	}
	return nil
}

func buildRelativeMouseInput(delta domain.Point, extraInfo uintptr) mouseInput {
	return mouseInput{
		Type:        inputMouse,
		Dx:          int32(delta.X),
		Dy:          int32(delta.Y),
		DwFlags:     mouseeventfMove,
		DwExtraInfo: extraInfo,
	}
}
