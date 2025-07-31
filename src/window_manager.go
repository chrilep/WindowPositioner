package main

import (
	"context"
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// WindowManager manages the main application window and the list of windows
// It provides functionality to enumerate, save, and apply window positions.
type WindowManager struct {
	app        fyne.App
	mainWindow fyne.Window
	storage    *PositionStorage
	windowList *widget.List
	windows    []WindowInfo
}

// NewWindowManager initializes the WindowManager with the given application
func NewWindowManager(app fyne.App) *WindowManager {
	wm := &WindowManager{
		app:     app,
		storage: NewPositionStorage(),
	}

	wm.createMainWindow()
	return wm
}

// createMainWindow sets up the main application window
// It includes a close intercept to hide the window instead of closing it.
func (wm *WindowManager) createMainWindow() {
	wm.mainWindow = wm.app.NewWindow(strPublisherName + `'s ` + strProductName + ` ` + strVersion)
	wm.mainWindow.Resize(fyne.NewSize(600, 100))

	// Hide window instead of closing to keep in system tray
	wm.mainWindow.SetCloseIntercept(func() {
		wm.mainWindow.Hide()
	})
	wm.setupMainWindowContent()
}

// setupMainWindowContent sets up the content of the main window
func (wm *WindowManager) setupMainWindowContent() {
	log(true, "Setting up main window content.")

	// Title label
	labTitle := widget.NewLabel("Visible Windows")
	labTitle.TextStyle = fyne.TextStyle{Bold: true}

	// Refresh button
	refreshBtn := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
		wm.refreshWindowList()
	})

	// Window list
	const listItemHeight = 40 // Vertical pixel per scroll item (approx)
	wm.windowList = widget.NewList(
		func() int {
			return len(wm.windows)
		},
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewButtonWithIcon("", theme.DocumentSaveIcon(), nil),
				widget.NewLabel("Window Title"),
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id >= len(wm.windows) {
				return
			}

			window := wm.windows[id]
			hbox := obj.(*fyne.Container)
			button := hbox.Objects[0].(*widget.Button)
			label := hbox.Objects[1].(*widget.Label)

			button.OnTapped = func() {
				wm.saveWindowPosition(window)
			}
			label.SetText(fmt.Sprintf("%s [%s]", window.Title, window.ClassName))
		},
	)
	scrollWindowList := container.NewScroll(wm.windowList)
	scrollWindowList.SetMinSize(fyne.NewSize(0, 5*listItemHeight))

	// Saved positions section
	savedLabel := widget.NewLabel("Saved Positions")
	savedLabel.TextStyle = fyne.TextStyle{Bold: true}

	// Create a list for saved positions
	savedList := wm.createSavedPositionsList()
	scrollSavedList := container.NewScroll(savedList)
	scrollSavedList.SetMinSize(fyne.NewSize(0, 5*listItemHeight))

	// Settings section
	labSettings := widget.NewLabel("Settings")
	labSettings.TextStyle = fyne.TextStyle{Bold: true}
	startupCheck := widget.NewCheck("Start with Windows", func(checked bool) {
		if checked {
			if err := EnableStartup(); err != nil {
				log(true, "Failed to enable startup:", err)
			}
		} else {
			if err := DisableStartup(); err != nil {
				log(true, "Failed to disable startup:", err)
			}
		}
	})

	// Check current startup status
	startupCheck.SetChecked(IsStartupEnabled())

	// Layout
	content := container.NewVBox(
		container.NewHBox(labTitle, widget.NewSeparator(), refreshBtn),
		widget.NewSeparator(),
		scrollWindowList,
		widget.NewSeparator(),
		savedLabel,
		widget.NewSeparator(),
		scrollSavedList,
		widget.NewSeparator(),
		labSettings,
		startupCheck,
	)

	wm.mainWindow.SetContent(content)
	wm.refreshWindowList()
}

// createSavedPositionsList creates a list of saved window positions
// It allows users to apply or delete saved positions.
func (wm *WindowManager) createSavedPositionsList() *widget.List {
	positions := wm.storage.GetAllPositions()
	positionKeys := make([]string, 0, len(positions))
	for key := range positions {
		positionKeys = append(positionKeys, key)
	}

	return widget.NewList(
		func() int {
			return len(positionKeys)
		},
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewButtonWithIcon("", theme.DeleteIcon(), nil),
				widget.NewLabel("Position"),
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id >= len(positionKeys) {
				return
			}

			key := positionKeys[id]
			hbox := obj.(*fyne.Container)
			deleteBtn := hbox.Objects[0].(*widget.Button)
			label := hbox.Objects[1].(*widget.Label)

			label.SetText(key)
			deleteBtn.OnTapped = func() {
				wm.storage.DeletePosition(key)
				wm.setupMainWindowContent() // Refresh the UI
			}
		},
	)
}

// refreshWindowList fetches the current list of windows and updates the window list widget
func (wm *WindowManager) refreshWindowList() {
	windows, err := EnumerateWindows()
	if err != nil {
		log(true, "Failed to enumerate windows:", err)
		return
	}

	// Filter out system windows and our own window
	var filteredWindows []WindowInfo
	for _, window := range windows {
		if window.Title != "" && window.Title != strAppTitle {
			filteredWindows = append(filteredWindows, window)
		}
	}

	wm.windows = filteredWindows
	wm.windowList.Refresh()
}

// saveWindowPosition saves the current position of a window identified by its class name and title
// It retrieves the window position and stores it in the PositionStorage.
func (wm *WindowManager) saveWindowPosition(window WindowInfo) {
	pos, err := GetWindowPosition(window.Handle)
	if err != nil {
		log(true, "Failed to get window position:", err)
		return
	}

	identifier := fmt.Sprintf("%s|%s|%s|%x|%x", window.Title, window.ClassName, window.Executable, window.Style, window.ExStyle)
	err = wm.storage.SavePosition(identifier, *pos)
	if err != nil {
		log(true, "Failed to save position:", err)
		return
	}

	log(true, "Saved position for:", identifier)
	wm.setupMainWindowContent() // Refresh the UI
}

// repositionSavedWindows repositions all saved windows based on their stored positions
// This is called on startup and periodically by the monitoring service.
func (wm *WindowManager) repositionSavedWindows() {
	debug := false
	positions := wm.storage.GetAllPositions()
	windows, err := EnumerateWindows()
	if err != nil {
		log(true, "Failed to enumerate windows:", err)
		return
	}

	for _, window := range windows {
		identifier := fmt.Sprintf("%s|%s|%s|%x|%x", window.Title, window.ClassName, window.Executable, window.Style, window.ExStyle)
		if pos, exists := positions[identifier]; exists {
			err := MoveWindowAccurate(window.Handle, pos.X, pos.Y, pos.Width, pos.Height)
			if err != nil {
				log(true, "Failed to auto-position window:", identifier, err)
			} else {
				log(debug, "Auto-positioned:", identifier)
			}
		}
	}
}

// startMonitoringService runs a background service that periodically checks for window positions
// and repositions them if necessary. This is useful for keeping windows in their saved positions.
func (wm *WindowManager) startMonitoringService(ctx context.Context) {
	log(true, "Starting background window monitoring service.")
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			wm.repositionSavedWindows()
		}
	}
}

// setupSystemTray sets up the system tray menu for the application
func (wm *WindowManager) setupSystemTray(desk desktop.App) {
	log(true, "Setting up system tray menu.")
	menu := fyne.NewMenu(strProductName,
		fyne.NewMenuItem("Show Manager", func() {
			wm.mainWindow.Show()
			wm.mainWindow.RequestFocus()
		}),
		//fyne.NewMenuItemSeparator(),
		//fyne.NewMenuItem("Auto-position Now", func() {
		//	wm.repositionSavedWindows()
		//}),
		//fyne.NewMenuItemSeparator(),
		//fyne.NewMenuItem("Quit", func() {
		//	wm.app.Quit()
		//}),
	)
	desk.SetSystemTrayMenu(menu)
}
