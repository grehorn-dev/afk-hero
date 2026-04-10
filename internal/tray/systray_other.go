//go:build !windows

package tray

import "afk-hero/internal/domain"

// Manager is a stub for non-Windows platforms.
type Manager struct{}

func NewManager(_ Callbacks) *Manager                       { return &Manager{} }
func (m *Manager) Register()                                {}
func (m *Manager) Quit()                                    {}
func (m *Manager) Update(_ domain.Settings, _ domain.State) {}
