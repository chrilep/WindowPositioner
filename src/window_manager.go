package main

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
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
	//wm.mainWindow.Resize(fyne.NewSize(600, 100))

	// Hide window instead of closing to keep in system tray
	wm.mainWindow.SetCloseIntercept(func() {
		wm.mainWindow.Hide()
	})
	wm.setupMainWindowContent()
}

// setupMainWindowContent sets up the content of the main window
func (wm *WindowManager) setupMainWindowContent() {
	log(true, "Setting up main window content.")

	// Separators
	separator := widget.NewSeparator()

	// Title label
	labTitle := widget.NewLabel("Visible Windows")
	labTitle.TextStyle = fyne.TextStyle{Bold: true}

	// Refresh button
	refreshBtn := widget.NewButtonWithIcon("Refresh", theme.ViewRefreshIcon(), func() {
		wm.refreshWindowList()
	})

	// Exit button
	exitBtn := widget.NewButtonWithIcon("Exit", theme.LogoutIcon(), func() {
		wm.app.Quit()
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
				widget.NewButtonWithIcon("", theme.InfoIcon(), nil), // Info-Button
				widget.NewLabel("Window Title"),
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id >= len(wm.windows) {
				return
			}

			window := wm.windows[id]
			hbox := obj.(*fyne.Container)
			saveBtn := hbox.Objects[0].(*widget.Button)
			infoBtn := hbox.Objects[1].(*widget.Button)
			label := hbox.Objects[2].(*widget.Label)

			saveBtn.OnTapped = func() {
				wm.saveWindowPosition(window)
			}
			infoBtn.OnTapped = func() {
				x := int(window.WindowRect.Left)
				y := int(window.WindowRect.Top)
				width := int(window.WindowRect.Right - window.WindowRect.Left)
				height := int(window.WindowRect.Bottom - window.WindowRect.Top)

				infoText := fmt.Sprintf(
					"Window    :\n'%s'\n\n"+
						"Position  : %d,%d\n"+
						"Size      : %dx%d\n"+
						"Process ID: %d\n"+
						"Class Name: %s\n"+
						"HWND      : 0x%08X\n"+
						"Style     : 0x%08X\n"+
						"ExStyle   : 0x%08X\n"+
						"Executable:\n'%s'",
					window.Title,
					x, y, width, height,
					window.ProcessID,
					window.ClassName,
					window.Handle,
					window.Style,
					window.ExStyle,
					window.Executable,
				)

				entry := widget.NewMultiLineEntry()
				entry.SetText(infoText)
				entry.TextStyle = fyne.TextStyle{Monospace: true}
				entry.Wrapping = fyne.TextWrapBreak

				scroll := container.NewScroll(entry)
				scroll.SetMinSize(fyne.NewSize(400, 300))

				dialog.ShowCustom("Details for this window", "Close", scroll, wm.mainWindow)
			}
			label.SetText(fmt.Sprintf("%s [%s]", window.Title, window.ClassName))
		},
	)
	scrollWindowList := container.NewScroll(wm.windowList)
	scrollWindowList.SetMinSize(fyne.NewSize(0, 5*listItemHeight))

	// Saved positions section
	savedLabel := widget.NewLabel("Saved Positions")
	savedLabel.TextStyle = fyne.TextStyle{Bold: true}

	configBtn := widget.NewButtonWithIcon("Edit", theme.FileTextIcon(), func() {
		// Open the configuration file ps.storageFile in the default text editor
		cmd := exec.Command("cmd", "/C", "start", "", wm.storage.storageFile)
		err := cmd.Run()
		if err != nil {
			log(true, "Failed to open config file:", err)
			dialog.ShowError(err, wm.mainWindow)
		}
	})

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
		container.New(layout.NewGridLayout(4), labTitle, separator, refreshBtn, exitBtn),
		separator,
		//container.NewHBox(labTitle, separator, refreshBtn, separator, exitBtn),
		separator,
		scrollWindowList,
		widget.NewSeparator(),
		container.New(layout.NewGridLayout(4), savedLabel, separator, separator, configBtn),
		//container.NewHBox(savedLabel, separator, configBtn),
		separator,
		scrollSavedList,
		separator,
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

	identifier := fmt.Sprintf("%s|%s|%s|0x%08X|0x%08X", window.Title, window.ClassName, window.Executable, window.Style, window.ExStyle)
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
		identifier := fmt.Sprintf("%s|%s|%s|0x%08X|0x%08X", window.Title, window.ClassName, window.Executable, window.Style, window.ExStyle)
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
	log(true, "Setting up system tray menu for", strProductName+`.`)
	menu := fyne.NewMenu(strProductName,
		fyne.NewMenuItem("Show Manager", func() {
			wm.mainWindow.Show()
			wm.mainWindow.RequestFocus()
			wm.mainWindow.CenterOnScreen()
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
