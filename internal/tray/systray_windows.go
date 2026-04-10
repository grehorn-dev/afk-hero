//go:build windows

package tray

import (
	"afk-hero/internal/appmeta"
	"afk-hero/internal/domain"
	"afk-hero/internal/logging"
	"sync"

	"github.com/getlantern/systray"
)

// Manager handles the system tray icon and menu on Windows.
type Manager struct {
	mu        sync.Mutex
	callbacks Callbacks
	state     MenuState
	ready     bool
	stopCh    chan struct{}
	stopOnce  sync.Once

	menuSettings *systray.MenuItem
	menuToggle   *systray.MenuItem
	menuExit     *systray.MenuItem
}

// NewManager creates a new tray manager.
func NewManager(cb Callbacks) *Manager {
	return &Manager{
		callbacks: cb,
		stopCh:    make(chan struct{}),
	}
}

// Register initializes the tray for coexistence with another GUI loop.
func (m *Manager) Register() {
	systray.Register(m.onReady, m.onExit)
}

// Quit stops the system tray.
func (m *Manager) Quit() {
	m.signalStop()
	systray.Quit()
}

func (m *Manager) onReady() {
	systray.SetIcon(DisabledIcon)
	systray.SetTitle(appmeta.DisplayName)
	systray.SetTooltip(appmeta.DisplayName)

	m.menuSettings = systray.AddMenuItem("Settings", "Open settings window")
	systray.AddSeparator()
	m.menuToggle = systray.AddMenuItem("Enable", "Toggle enable/disable")
	systray.AddSeparator()
	m.menuExit = systray.AddMenuItem("Exit", "Quit the application")

	m.mu.Lock()
	m.ready = true
	current := m.state
	m.mu.Unlock()

	m.render(current)

	// Handle menu clicks
	go m.handleClicks()
}

func (m *Manager) onExit() {
	m.signalStop()
}

func (m *Manager) handleClicks() {
	defer logging.RecoverPanic("tray.Manager.handleClicks")

	for {
		select {
		case <-m.stopCh:
			return
		case <-m.menuSettings.ClickedCh:
			if m.callbacks.OnShowSettings != nil {
				m.callbacks.OnShowSettings()
			}
		case <-m.menuToggle.ClickedCh:
			m.mu.Lock()
			enabled := m.state.Enabled
			m.mu.Unlock()

			if enabled {
				if m.callbacks.OnDisable != nil {
					m.callbacks.OnDisable()
				}
			} else {
				if m.callbacks.OnEnable != nil {
					m.callbacks.OnEnable()
				}
			}
		case <-m.menuExit.ClickedCh:
			if m.callbacks.OnExit != nil {
				m.callbacks.OnExit()
			}
		}
	}
}

func (m *Manager) signalStop() {
	m.stopOnce.Do(func() {
		close(m.stopCh)
	})
}

// Update refreshes the tray menu to match current state.
func (m *Manager) Update(settings domain.Settings, state domain.State) {
	m.mu.Lock()
	m.state = StateToMenuState(settings, state)
	ready := m.ready
	ms := m.state
	m.mu.Unlock()

	if !ready {
		return
	}

	m.render(ms)
}

func (m *Manager) render(ms MenuState) {
	if ms.Enabled {
		systray.SetIcon(EnabledIcon)
	} else {
		systray.SetIcon(DisabledIcon)
	}

	labels := GetMenuLabels(ms.Language)
	if m.menuSettings != nil {
		m.menuSettings.SetTitle(labels.Settings)
	}
	if m.menuToggle != nil {
		if ms.Enabled {
			m.menuToggle.SetTitle(labels.Disable)
		} else {
			m.menuToggle.SetTitle(labels.Enable)
		}
	}
	if m.menuExit != nil {
		m.menuExit.SetTitle(labels.Exit)
	}
}
