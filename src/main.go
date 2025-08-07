package main

/*
	compile: fyne package --release --os windows --executable "WindowPositioner.exe"
*/

import (
	"context"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/driver/desktop"
)

// Global variables for the application
var (
	// Global variables for publisher, product, and version names
	strPublisherName = "Lancer"
	strProductName   = "WindowPositioner"
	strVersion       = "1.2.1"
	strAppId         = "com.lancer.windowpositioner"
	// cd src
	// fyne package -os windows

	// Global variable for the main application window manager
	strAppTitle = strPublisherName + `'s ` + strProductName + ` ` + strVersion
	wm          *WindowManager
)

// Main entry point for the application
func main() {

	defer panicHandler()

	debug := true
	log(true, `Starting`, strAppTitle)

	// Install a handler for SIGINT/SIGTERM signals to log when the application receives these signals
	chanSignal := make(chan os.Signal, 1)
	signal.Notify(chanSignal, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		for sig := range chanSignal {
			log(true, "Signal received:", sig)
		}
	}()

	// Create the application
	myApp := app.NewWithID(strAppId)

	// Initialize the window manager
	wm = NewWindowManager(myApp)

	// Set up system tray if supported
	if desk, ok := myApp.(desktop.App); ok {
		wm.setupSystemTray(desk)
	}

	// Start the background window monitoring service
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go wm.startMonitoringService(ctx)

	// Auto-position any saved windows on startup
	go func() {
		defer panicHandler()
		time.Sleep(2 * time.Second) // Give time for other apps to load
		wm.repositionSavedWindows()
	}()

	// Run the application (this blocks until app.Quit() is called)
	log(debug, "Entering event loop.")

	// Keep the app active and handle memory usage
	go func() {
		defer panicHandler()
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				var m runtime.MemStats
				runtime.ReadMemStats(&m)

				// Force garbage collection periodically if memory usage is higher than 100 MB
				if m.Alloc > 100*1024*1024 {
					log(true, "Memory usage:", m.Alloc/1024, "KB, Goroutines:", runtime.NumGoroutine(), "-> High memory usage, forcing garbage collecting.")
					runtime.GC()
					runtime.ReadMemStats(&m)
					log(true, "Memory usage:", m.Alloc/1024, "KB (after GC)")
				}
			}
		}
	}()

	myApp.Run()
	log(debug, "Exiting event loop. App closes now.")
}
