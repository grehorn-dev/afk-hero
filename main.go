package main

import (
	"afk-hero/internal/app"
	"afk-hero/internal/appmeta"
	"afk-hero/internal/bootstrap"
	"afk-hero/internal/config"
	"afk-hero/internal/domain"
	"afk-hero/internal/logging"
	"afk-hero/internal/tray"
	"context"
	"fmt"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

func main() {
	if err := logging.Init(false); err != nil {
		fmt.Fprintf(os.Stderr, "logging init failed: %v\n", err)
	}
	defer logging.RecoverPanic("main")
	log := logging.Get()
	startupSettings, err := config.Load()
	if err != nil {
		startupSettings = domain.DefaultSettings()
	}
	windowHeight := appmeta.WindowHeight(startupSettings)

	services := bootstrap.NewServices()
	application := app.NewApp(services.Activity, services.Pointer, services.Screen, services.WindowManager)

	trayMgr := tray.NewManager(tray.Callbacks{
		OnShowSettings: application.ShowWindow,
		OnEnable: func() {
			application.SetEnabled(true)
		},
		OnDisable: func() {
			application.SetEnabled(false)
		},
		OnExit: application.RequestQuit,
	})

	application.SetTrayCallback(func(s domain.Settings, st domain.State) {
		trayMgr.Update(s, st)
	})
	trayMgr.Register()

	shutdown := func(ctx context.Context) {
		trayMgr.Quit()
		application.Shutdown(ctx)
	}

	appOptions := &options.App{
		Title:             appmeta.WindowTitle,
		Width:             appmeta.WindowWidth,
		Height:            windowHeight,
		MinWidth:          appmeta.WindowWidth,
		MinHeight:         windowHeight,
		MaxWidth:          appmeta.WindowWidth,
		MaxHeight:         windowHeight,
		DisableResize:     true,
		Frameless:         false,
		StartHidden:       true,
		HideWindowOnClose: true,
		BackgroundColour:  &options.RGBA{R: 255, G: 255, B: 255, A: 255},
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup:     application.Startup,
		OnShutdown:    shutdown,
		OnBeforeClose: application.BeforeClose,
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId: appmeta.SingleInstanceID,
			OnSecondInstanceLaunch: func(_ options.SecondInstanceData) {
				application.ShowWindow()
			},
		},
		Bind: []interface{}{
			application,
		},
	}

	bootstrap.ConfigurePlatformOptions(appOptions)

	if err := wails.Run(appOptions); err != nil {
		log.Error("application error", "error", err)
	}
}
