package main

/*
	compile: fyne package --release --os windows --executable "WindowPositioner.exe"
*/

import (
	"context"
	"fmt"
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
	strVersion       = "1.3.1"
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
	log(true, "HEARTBEAT: Application startup initiated at", time.Now().Format("2006-01-02 15:04:05"))

	// Create context for coordinated shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Install a handler for SIGINT/SIGTERM signals to log when the application receives these signals
	chanSignal := make(chan os.Signal, 1)
	signal.Notify(chanSignal, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		defer panicHandler()
		for sig := range chanSignal {
			log(true, "Signal received:", sig)
			if sig == syscall.SIGINT || sig == syscall.SIGTERM {
				log(true, "HEARTBEAT: Graceful shutdown requested via signal", sig)
				cancel() // Cancel the context to stop other goroutines
				if wm != nil && wm.app != nil {
					wm.app.Quit()
				}
			}
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

	// Heartbeat logging to track application lifetime
	go func() {
		defer panicHandler()
		heartbeatTicker := time.NewTicker(5 * time.Minute) // Log every 5 minutes
		defer heartbeatTicker.Stop()

		startTime := time.Now()
		heartbeatCounter := 0

		for {
			select {
			case <-ctx.Done():
				log(true, "HEARTBEAT: Application shutdown requested after", time.Since(startTime).Round(time.Second))
				return
			case <-heartbeatTicker.C:
				heartbeatCounter++
				uptime := time.Since(startTime).Round(time.Second)

				var m runtime.MemStats
				runtime.ReadMemStats(&m)

				// Get current goroutine count
				goroutines := runtime.NumGoroutine()

				// Get current window count if available
				windowCount := 0
				if wm != nil {
					windows := wm.getWindows()
					windowCount = len(windows)
				}

				log(true, fmt.Sprintf("HEARTBEAT #%d: Uptime=%v, Memory=%dKB, Goroutines=%d, Windows=%d",
					heartbeatCounter, uptime, m.Alloc/1024, goroutines, windowCount))

				// Log additional info every 30 minutes (every 6th heartbeat)
				if heartbeatCounter%6 == 0 {
					log(true, fmt.Sprintf("HEARTBEAT EXTENDED: TotalAlloc=%dKB, Sys=%dKB, NumGC=%d, HeapObjects=%d",
						m.TotalAlloc/1024, m.Sys/1024, m.NumGC, m.HeapObjects))
				}
			}
		}
	}()

	myApp.Run()
	log(debug, "Exiting event loop. App closes now.")
	log(true, "HEARTBEAT: Application shutdown completed at", time.Now().Format("2006-01-02 15:04:05"))
}
