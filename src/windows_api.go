package main

import (
	"fmt"
	"syscall"
	"unsafe"
)

// WindowInfo holds information about a window
// It includes the window handle, title, class name, process ID, executable path or name,
// window styles, extended styles, and rectangles for the client area and window rectangle.
type WindowInfo struct {
	Handle     syscall.Handle
	Title      string
	ClassName  string
	ProcessID  uint32
	Executable string // Process executable path or name
	Style      uint32 // Window styles (GWL_STYLE)
	ExStyle    uint32 // Extended styles (GWL_EXSTYLE)
	ClientRect RECT   // Client area rectangle (relative to window)
	WindowRect RECT   // Window rectangle (screen coordinates)
}

// WindowPosition holds the position and size of a window
// It includes the x and y coordinates, width, and height.
type WindowPosition struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

// RECT represents a rectangle in screen coordinates
// It is used to define the position and size of a window.
type RECT struct {
	Left, Top, Right, Bottom int32
}

// Windows API functions
var (
	user32 = syscall.NewLazyDLL("user32.dll")
	//dwmapi                       = syscall.NewLazyDLL("dwmapi.dll")
	procEnumWindows              = user32.NewProc("EnumWindows")
	procGetWindowText            = user32.NewProc("GetWindowTextW")
	procGetClassName             = user32.NewProc("GetClassNameW")
	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
	procSetWindowPos             = user32.NewProc("SetWindowPos")
	procGetWindowRect            = user32.NewProc("GetWindowRect")
	procIsWindowVisible          = user32.NewProc("IsWindowVisible")
	//procDwmGetWindowAttribute    = dwmapi.NewProc("DwmGetWindowAttribute")
	kernel32                 = syscall.NewLazyDLL("kernel32.dll")
	psapi                    = syscall.NewLazyDLL("psapi.dll")
	procOpenProcess          = kernel32.NewProc("OpenProcess")
	procCloseHandle          = kernel32.NewProc("CloseHandle")
	procGetWindowLongPtrW    = user32.NewProc("GetWindowLongPtrW")
	procGetWindowLongW       = user32.NewProc("GetWindowLongW") // fallback if 32-bit
	procGetClientRect        = user32.NewProc("GetClientRect")
	procGetModuleFileNameExW = psapi.NewProc("GetModuleFileNameExW")
)

const (
	DWMWA_EXTENDED_FRAME_BOUNDS       = 9
	PROCESS_QUERY_LIMITED_INFORMATION = 0x1000
	GWL_STYLE                         = -16
	GWL_EXSTYLE                       = -20
)

// EnumerateWindows retrieves a list of all visible windows on the desktop.
// It returns a slice of WindowInfo structs containing the handle, title, class name, and process ID of each window.
// It uses the EnumWindows function to enumerate all top-level windows.
// The callback function filters out invisible windows and collects the necessary information.
// It returns an error if the enumeration fails.
func EnumerateWindows() ([]WindowInfo, error) {
	debug := false
	log(debug, "Enumerating visible windows.")
	var windows []WindowInfo

	callback := syscall.NewCallback(func(hwnd syscall.Handle, lparam uintptr) uintptr {
		if isWindowVisible(hwnd) {
			info := getWindowInfo(hwnd)
			width := int(info.WindowRect.Right - info.WindowRect.Left)
			height := int(info.WindowRect.Bottom - info.WindowRect.Top)
			if info.Title != "" &&
				len(info.Title) > 0 &&
				width > 8 &&
				height > 8 {
				log(debug, "Found window via handle:", info.Handle)
				log(debug, "- Title       :", info.Title)
				log(debug, "- ClassName   :", info.ClassName)
				log(debug, "- Executable  :", info.Executable)
				log(debug, "- Style       :", info.Style)
				log(debug, "- ExStyle     :", info.ExStyle)
				log(debug, "- ClientRect  :", info.ClientRect)
				log(debug, "- WindowRect  :", info.WindowRect)

				windows = append(windows, info)
			}
		}
		return 1 // Continue enumeration
	})

	ret, _, _ := procEnumWindows.Call(uintptr(callback), 0)
	if ret == 0 {
		return nil, fmt.Errorf("EnumWindows failed")
	}

	return windows, nil
}

// isWindowVisible checks if a window is visible.
func isWindowVisible(hwnd syscall.Handle) bool {
	ret, _, _ := procIsWindowVisible.Call(uintptr(hwnd))
	return ret != 0
}

// getWindowInfo retrieves the title, class name, and process ID of a window.
// It uses GetWindowText to get the title, GetClassName to get the class name
func getWindowInfo(hwnd syscall.Handle) WindowInfo {
	debug := false
	log(debug, "Getting window info for handle:", hwnd)

	// Get window title
	titleBuf := make([]uint16, 256)
	procGetWindowText.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&titleBuf[0])), uintptr(len(titleBuf)))
	title := syscall.UTF16ToString(titleBuf)
	log(debug, "Window title:", title)

	// Get class name
	classBuf := make([]uint16, 256)
	procGetClassName.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&classBuf[0])), uintptr(len(classBuf)))
	className := syscall.UTF16ToString(classBuf)
	log(debug, "Window class name:", className)

	// Get process ID
	var processID uint32
	procGetWindowThreadProcessId.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&processID)))
	log(debug, "Process ID:", processID)

	// Get process executable path
	exePath, _ := getProcessExecutablePath(processID)
	log(debug, "Process executable path:", exePath)

	// Get window styles and extended styles
	style, _ := getWindowLong(hwnd, GWL_STYLE)
	exstyle, _ := getWindowLong(hwnd, GWL_EXSTYLE)
	log(debug, "Window styles:", style, "Extended styles:", exstyle)

	// Get client rectangle
	clientRect, _ := getClientRect(hwnd)
	log(debug, "Client rectangle:", clientRect)

	// Get window rectangle
	var windowRect RECT
	procGetWindowRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&windowRect)))
	log(debug, "Window rectangle:", windowRect)

	return WindowInfo{
		Handle:     hwnd,
		Title:      title,
		ClassName:  className,
		ProcessID:  processID,
		Executable: exePath,
		Style:      uint32(style),
		ExStyle:    uint32(exstyle),
		ClientRect: *clientRect,
		WindowRect: windowRect,
	}
}

// GetWindowPosition retrieves the position and size of a window.
// It uses GetWindowRect to get the window's bounding rectangle.
func GetWindowPosition(hwnd syscall.Handle) (*WindowPosition, error) {
	var rect RECT
	ret, _, _ := procGetWindowRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&rect)))
	if ret == 0 {
		return nil, fmt.Errorf("GetWindowRect failed")
	}

	return &WindowPosition{
		X:      int(rect.Left),
		Y:      int(rect.Top),
		Width:  int(rect.Right - rect.Left),
		Height: int(rect.Bottom - rect.Top),
	}, nil
}

// MoveWindowAccurate moves a window to a specified position and size.
// It uses SetWindowPos to set the window's position and size accurately.
func MoveWindowAccurate(hwnd syscall.Handle, x, y, width, height int) error {
	debug := false
	log(debug, "Moving window:", hwnd, "to position:", x, y, "with size:", width, height)
	// Get current position and size
	pos, err := GetWindowPosition(hwnd)
	if err != nil {
		log(true, "Failed to get current window position:", err)
		return fmt.Errorf("failed to get current window position: %v", err)
	}
	if pos.X == x && pos.Y == y && pos.Width == width && pos.Height == height {
		log(debug, "Window already at desired position and size.")
		return nil // Already at desired position and size
	}

	// Use SetWindowPos for precise positioning
	ret, _, _ := procSetWindowPos.Call(
		uintptr(hwnd),
		0, // HWND_TOP
		uintptr(x),
		uintptr(y),
		uintptr(width),
		uintptr(height),
		0x0040, // SWP_SHOWWINDOW
	)
	if ret == 0 {
		return fmt.Errorf("SetWindowPos failed")
	}
	log(debug, "Window moved successfully.")
	return nil
}

// openProcess opens a handle to a process by its PID.
// It uses OpenProcess with PROCESS_QUERY_LIMITED_INFORMATION access.
func openProcess(pid uint32) (syscall.Handle, error) {
	h, _, err := procOpenProcess.Call(uintptr(PROCESS_QUERY_LIMITED_INFORMATION), uintptr(0), uintptr(pid))
	if h == 0 {
		return 0, err
	}
	return syscall.Handle(h), nil
}

// closeHandle closes a handle to a process.
// It uses CloseHandle to release the handle.
func closeHandle(handle syscall.Handle) {
	procCloseHandle.Call(uintptr(handle))
}

// getProcessExecutablePath retrieves the executable path of a process by its PID.
// It uses GetModuleFileNameExW to get the executable path of the main module of the process.
// It returns the path as a string or an error if it fails.
func getProcessExecutablePath(pid uint32) (string, error) {
	debug := false
	log(debug, "Getting executable path for PID:", pid)

	handle, err := openProcess(pid)
	if err != nil || handle == 0 {
		return "", fmt.Errorf("OpenProcess failed: %v", err)
	}
	defer closeHandle(handle)
	log(debug, "Opened process handle:", handle)

	buf := make([]uint16, syscall.MAX_PATH)
	ret, _, err := procGetModuleFileNameExW.Call(
		uintptr(handle),
		uintptr(0), // Null-HANDLE für Hauptmodul
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)),
	)
	if ret == 0 {
		errno, ok := err.(syscall.Errno)
		if ok {
			return "", fmt.Errorf("GetModuleFileNameExW failed: %v", errno.Error())
		}
		return "", fmt.Errorf("GetModuleFileNameExW failed: %v", err)
	}
	path := syscall.UTF16ToString(buf[:ret])
	log(debug, "Executable path:", path)
	return path, nil
}

// getWindowLong retrieves the window styles or extended styles for a window.
// It uses GetWindowLongPtrW for 64-bit Windows and falls back to GetWindowLongW for 32-bit Windows.
// It returns the styles as a uintptr or an error if it fails.
func getWindowLong(hwnd syscall.Handle, index int32) (uintptr, error) {
	ret, _, _ := procGetWindowLongPtrW.Call(uintptr(hwnd), uintptr(index))
	if ret == 0 {
		// Try 32-bit fallback for older Windows
		ret32, _, err32 := procGetWindowLongW.Call(uintptr(hwnd), uintptr(index))
		if ret32 == 0 {
			return 0, fmt.Errorf("GetWindowLong failed: %v", err32)
		}
		return ret32, nil
	}
	return ret, nil
}

// getClientRect retrieves the client area rectangle of a window.
// It uses GetClientRect to get the rectangle in client coordinates.
func getClientRect(hwnd syscall.Handle) (*RECT, error) {
	var rect RECT
	ret, _, err := procGetClientRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&rect)))
	if ret == 0 {
		return nil, fmt.Errorf("GetClientRect failed: %v", err)
	}
	return &rect, nil
}
