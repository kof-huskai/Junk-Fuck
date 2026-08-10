package main

import (
	"context"
	"embed"
	"log"

	"github.com/kof-huskai/Junk-Fuck/services"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/updater"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/github"
)

//go:embed all:embed
var assets embed.FS

// Version is injected at release-build time via:
//
//	go build -ldflags "-X main.Version=${VERSION}"
//
// It is the single source of truth for the running version and is also
// passed to the updater so version comparison happens against the same
// value the GitHub release was tagged with.
var Version = "dev"

func main() {
	core := services.NewCore()
	defer core.Close()

	settingsService := core.SettingsService()
	settingsService.SetVersion(Version)
	updateService := core.UpdateService()

	app := application.New(application.Options{
		Name:        "JunkFuck",
		Description: "Deep Windows junk scanner & cleaner",
		Services: []application.Service{
			application.NewService(core.ScannerService()),
			application.NewService(core.CleanupService()),
			application.NewService(settingsService),
			application.NewService(updateService),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Windows: application.WindowsOptions{
			// Windows 10/11 with WebView2 runtime (preinstalled on
			// current updates; otherwise installed by Microsoft).
		},
	})

	// Register the official updater against GitHub Releases. Release tags
	// are the source of truth; Telegram is only a distribution layer.
	if gh, err := github.New(github.Config{
		Repository:    "kof-huskai/Junk-Fuck",
		ChecksumAsset: "SHA256SUMS",
	}); err == nil {
		if err := app.Updater.Init(updater.Config{
			CurrentVersion: Version,
			Providers:      []updater.Provider{gh},
			// PublicKey intentionally unset: releases are unsigned for now.
			// When code signing is added, set PublicKey here to verify
			// release signatures (the framework fails closed on any release
			// that ships a signature while no key is pinned).
		}); err != nil {
			log.Printf("updater init skipped: %v", err)
		}
	} else {
		log.Printf("updater provider skipped: %v", err)
	}
	updateService.SetUpdater(app.Updater)

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "JunkFuck",
		Width:            1180,
		Height:           780,
		MinWidth:         900,
		MinHeight:        620,
		BackgroundColour: application.NewRGB(11, 13, 18),
		URL:              "/",
	})

	app.OnShutdown(func() {
		core.Close()
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_ = ctx // available for future background work

	if err := app.Run(); err != nil {
		log.Fatalf("failed to run JunkFuck: %v", err)
	}
}
