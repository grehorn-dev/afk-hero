//go:build windows

package config

import (
	"fmt"
	"syscall"
	"unsafe"
)

var procMoveFileExW = syscall.NewLazyDLL("kernel32.dll").NewProc("MoveFileExW")

const (
	moveFileReplaceExisting = 0x1
	moveFileWriteThrough    = 0x8
)

func replaceFile(src, dst string) error {
	srcPtr, err := syscall.UTF16PtrFromString(src)
	if err != nil {
		return fmt.Errorf("encode source path: %w", err)
	}

	dstPtr, err := syscall.UTF16PtrFromString(dst)
	if err != nil {
		return fmt.Errorf("encode destination path: %w", err)
	}

	r1, _, callErr := procMoveFileExW.Call(
		uintptr(unsafe.Pointer(srcPtr)),
		uintptr(unsafe.Pointer(dstPtr)),
		uintptr(moveFileReplaceExisting|moveFileWriteThrough),
	)
	if r1 == 0 {
		return callErr
	}

	return nil
}
