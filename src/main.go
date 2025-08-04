package main

/*
	compile: fyne package --release --os windows --executable "WindowPositioner.exe"
*/

import (
	"context"
	"time"

	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/driver/desktop"
)

var (
	// Global variables for publisher, product, and version names
	strPublisherName = "Lancer"
	strProductName   = "WindowPositioner"
	strVersion       = "1.2.0"
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

	log(true, `Starting`, strAppTitle)

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
	myApp.Run()
}
