// Package apperrors holds application-specific sentinel errors shared
// across packages. Only errors that are actually used by the rest of the
// codebase live here — dead sentinels have been removed to avoid agents
// or humans picking them up by mistake.
//
// The package is deliberately named `apperrors` (not `errors`) so it does
// not shadow the standard library `errors` package at every import site.
package apperrors

import "errors"

var (
	// ErrNotImplemented indicates a platform feature is not implemented
	// on the current build target. Returned by the non-Windows platform
	// stubs so callers can distinguish "unsupported" from "runtime error".
	ErrNotImplemented = errors.New("not implemented on this platform")

	// ErrWindowNotFound indicates that the target window could not be
	// resolved through the Windows manager. Returned by manual fingerprint
	// lookups and auto-target resolution when no candidate qualifies.
	ErrWindowNotFound = errors.New("target window not found")
)
