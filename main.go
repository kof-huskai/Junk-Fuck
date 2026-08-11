package main

import (
	"context"
	"embed"
	"log"
	"math"
	"sync"
	"time"

	"github.com/kof-huskai/Junk-Fuck/services"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
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

// The single shared app background. It must match --color-bg in
// frontend/src/index.css so the native title bar, window background and the
// rendered page appear as one continuous surface (no horizontal seam).
const (
	appBackgroundRed   = 0x0b // #0b0d12
	appBackgroundGreen = 0x0d
	appBackgroundBlue  = 0x12

	appTextRed   = 0xe7 // #e7eaf3 (body text colour)
	appTextGreen = 0xea
	appTextBlue  = 0xf3

	appTextMutedRed   = 0x8a // #8a93a8 (--color-muted)
	appTextMutedGreen = 0x93
	appTextMutedBlue  = 0xa8

	// Preferred logical window size. The final size is capped to fit the
	// current monitor's logical WorkArea (DPI-aware; Wails reports logical
	// coordinates), falling back to ~92% of the WorkArea on small screens
	// (1366x768 laptops, high-DPI setups) without forcing the full height.
	preferredWinWidth   = 1120
	preferredWinHeight  = 760
	workAreaFitFraction = 0.92
)

// fitWindowSize returns the fixed logical size for the given logical
// WorkArea: the preferred size when it fits, otherwise ~92% of the WorkArea.
func fitWindowSize(workAreaWidth, workAreaHeight int) (int, int) {
	w, h := preferredWinWidth, preferredWinHeight
	if w > workAreaWidth {
		w = int(math.Round(float64(workAreaWidth) * workAreaFitFraction))
	}
	if h > workAreaHeight {
		h = int(math.Round(float64(workAreaHeight) * workAreaFitFraction))
	}
	if w > workAreaWidth {
		w = workAreaWidth
	}
	if h > workAreaHeight {
		h = workAreaHeight
	}
	return w, h
}

// fitWindowToScreen sizes the window for the screen it is on and centers it
// within that screen's logical WorkArea. The window itself stays fixed-size
// for the user; only the application may change it programmatically.
func fitWindowToScreen(window *application.WebviewWindow) {
	screen, err := window.GetScreen()
	if err != nil || screen == nil {
		return
	}
	wa := screen.WorkArea
	w, h := fitWindowSize(wa.Width, wa.Height)
	window.SetSize(w, h)
	window.SetPosition(wa.X+(wa.Width-w)/2, wa.Y+(wa.Height-h)/2)
}

// monitorWindowPlacement keeps the fixed-size window usable across monitors:
// if the user drags it onto a screen whose WorkArea is too small, the window
// is resized programmatically (once, debounced) and kept visible. The
// maximize control is disabled; if the OS still maximizes via another
// gesture, the window is restored to its fixed windowed layout.
func monitorWindowPlacement(window *application.WebviewWindow) {
	var mu sync.Mutex
	var lastMove time.Time

	window.OnWindowEvent(events.Windows.WindowDidMove, func(*application.WindowEvent) {
		mu.Lock()
		lastMove = time.Now()
		mu.Unlock()
		go func() {
			// Debounce: re-evaluate only once the window has stopped moving.
			time.Sleep(320 * time.Millisecond)
			mu.Lock()
			settled := time.Since(lastMove) >= 280*time.Millisecond
			mu.Unlock()
			if !settled {
				return
			}
			fitIfNeeded(window)
		}()
	})

	window.OnWindowEvent(events.Windows.WindowMaximise, func(*application.WindowEvent) {
		if window.IsMaximised() {
			window.UnMaximise()
			fitIfNeeded(window)
		}
	})

	// Re-fit on the first show as a fallback: if GetScreen was not ready at
	// creation time, this guarantees the DPI-aware fit still runs at startup.
	var fitOnce sync.Once
	window.OnWindowEvent(events.Windows.WindowShow, func(*application.WindowEvent) {
		fitOnce.Do(func() { fitWindowToScreen(window) })
	})
}

// fitIfNeeded shrinks the window only when it no longer fits the current
// screen's WorkArea. It never grows the window while the user is dragging.
func fitIfNeeded(window *application.WebviewWindow) {
	screen, err := window.GetScreen()
	if err != nil || screen == nil {
		return
	}
	wa := screen.WorkArea
	curW, curH := window.Size()
	fitW, fitH := fitWindowSize(wa.Width, wa.Height)
	if curW > fitW || curH > fitH {
		window.SetSize(fitW, fitH)
		window.SetPosition(wa.X+(wa.Width-fitW)/2, wa.Y+(wa.Height-fitH)/2)
	}
}

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
		}, Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
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

	// Fixed-size, DPI-aware window: the user cannot resize or maximize it,
	// but the application sizes it to fit the current monitor's logical
	// WorkArea and adapts when the window is moved to a smaller screen.
	//
	// Theme/CustomTheme paint the native title bar, its text and the window
	// border with the same palette as the page background, so the title bar
	// and the app shell read as one continuous surface (Windows 11;
	// Windows 10 falls back to the standard dark title bar). Dark mode is
	// forced so the custom theme applies regardless of the user's theme.
	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:               "JunkFuck",
		Width:               preferredWinWidth,
		Height:              preferredWinHeight,
		DisableResize:       true,
		MaximiseButtonState: application.ButtonDisabled,
		BackgroundColour:    application.NewRGB(appBackgroundRed, appBackgroundGreen, appBackgroundBlue),
		InitialPosition:     application.WindowCentered,
		Windows: application.WindowsWindow{
			Theme: application.Dark,
			CustomTheme: application.ThemeSettings{
				DarkModeActive: &application.WindowTheme{
					TitleBarColour:  application.NewRGBPtr(appBackgroundRed, appBackgroundGreen, appBackgroundBlue),
					TitleTextColour: application.NewRGBPtr(appTextRed, appTextGreen, appTextBlue),
					BorderColour:    application.NewRGBPtr(appBackgroundRed, appBackgroundGreen, appBackgroundBlue),
				},
				DarkModeInactive: &application.WindowTheme{
					TitleBarColour:  application.NewRGBPtr(appBackgroundRed, appBackgroundGreen, appBackgroundBlue),
					TitleTextColour: application.NewRGBPtr(appTextMutedRed, appTextMutedGreen, appTextMutedBlue),
					BorderColour:    application.NewRGBPtr(appBackgroundRed, appBackgroundGreen, appBackgroundBlue),
				},
			},
		},
		URL: "/",
	})
	fitWindowToScreen(window)
	monitorWindowPlacement(window)

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
