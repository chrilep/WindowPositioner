package main

import (
	"fmt"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

// WindowInfo holds information about a window
// It includes the window handle, title, class name, process ID, executable path or name,
// window styles, extended styles, and rectangles for the client area and window rectangle.
type WindowInfo struct {
	Handle           syscall.Handle
	Title, ClassName string
	ProcessID        uint32
	Executable       string // Process executable path or name
	Style            uint32 // Window styles (GWL_STYLE)
	ExStyle          uint32 // Extended styles (GWL_EXSTYLE)
	ClientRect       RECT   // Client area rectangle (relative to window)
	WindowRect       RECT   // Window rectangle (screen coordinates)
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

// POINT defines the x- and y-coordinates of a point
type POINT struct {
	X, Y int32
}

// WINDOWPLACEMENT contains information about the placement of a window
type WINDOWPLACEMENT struct {
	Length           uint32  // Size of the structure in bytes
	Flags            uint32  // Flags that specify the window's state
	ShowCmd          uintptr // Show command for the window (SW_SHOW, SW_HIDE, etc.)
	PtMinPosition    POINT   // Point for the minimized position of the window
	PtMaxPosition    POINT   // Point for the maximized position of the window
	RcNormalPosition RECT    // Normal position rectangle of the window
}

// IAccessible interface definition
type IAccessible struct {
	vtbl *IAccessibleVtbl
}
type IAccessibleVtbl struct {
	QueryInterface   uintptr // Retrieves a pointer to the IAccessible interface
	AddRef           uintptr // Increments the reference count for the IAccessible interface
	Release          uintptr // Decrements the reference count for the IAccessible interface
	GetTypeInfoCount uintptr // Retrieves the number of type information interfaces that an object provides
	GetTypeInfo      uintptr // Retrieves the type information for an object
	GetIDsOfNames    uintptr // Maps a set of names to a corresponding set of dispatch identifiers
	Invoke           uintptr // Invokes a method or accesses a property of an object
	//get_accParent           uintptr
	//get_accChildCount       uintptr
	//get_accChild            uintptr
	//get_accName             uintptr
	//get_accValue            uintptr
	//get_accDescription      uintptr
	//get_accRole             uintptr
	//get_accState            uintptr
	//get_accHelp             uintptr
	//get_accHelpTopic        uintptr
	//get_accKeyboardShortcut uintptr
	//get_accFocus            uintptr
	//get_accSelection        uintptr
	//get_accDefaultAction    uintptr
	//accSelect               uintptr
	//accLocation             uintptr
	//accNavigate             uintptr
	//accHitTest              uintptr
	//accDoDefaultAction      uintptr
	//put_accName             uintptr
	//put_accValue            uintptr
}

// Windows API functions
var (
	// oleacc.dll functions
	oleacc                         = syscall.NewLazyDLL("oleacc.dll")             // OLE Accessibility functions
	procAccessibleObjectFromWindow = oleacc.NewProc("AccessibleObjectFromWindow") // Retrieves an accessible object from a window handle

	// ole32.dll functions
	ole32              = syscall.NewLazyDLL("ole32.dll") // OLE functions
	procCoInitialize   = ole32.NewProc("CoInitialize")   // Initializes the COM library for use by the calling thread
	procCoUninitialize = ole32.NewProc("CoUninitialize") // Uninitializes the COM library on the calling thread

	// kernel32.dll functions
	kernel32               = syscall.NewLazyDLL("kernel32.dll")
	procCloseHandle        = kernel32.NewProc("CloseHandle")        // Closes a handle to a process or thread
	procGetCurrentThreadId = kernel32.NewProc("GetCurrentThreadId") // Retrieves the thread ID of the calling thread
	procOpenProcess        = kernel32.NewProc("OpenProcess")        // Opens a handle to a process

	// psapi.dll functions
	psapi                    = syscall.NewLazyDLL("psapi.dll")
	procGetModuleFileNameExW = psapi.NewProc("GetModuleFileNameExW") // Retrieves the executable path of a process

	// user32.dll functions
	user32                       = syscall.NewLazyDLL("user32.dll")
	procAllowSetForegroundWindow = user32.NewProc("AllowSetForegroundWindow") // Allows a process to set the foreground window
	procAttachThreadInput        = user32.NewProc("AttachThreadInput")        // Attaches or detaches the input processing mechanism of one thread to another
	procEnumWindows              = user32.NewProc("EnumWindows")              // Enumerates all top-level windows
	procGetClassName             = user32.NewProc("GetClassNameW")            // Retrieves the class name of a window
	procGetClientRect            = user32.NewProc("GetClientRect")            // Retrieves the client area rectangle of a window
	procGetSystemMetrics         = user32.NewProc("GetSystemMetrics")         // Retrieves system metrics or system configuration settings
	procGetWindowLongPtrW        = user32.NewProc("GetWindowLongPtrW")        // Retrieves a value associated with a window (64-bit)
	procGetWindowLongW           = user32.NewProc("GetWindowLongW")           // Retrieves a value associated with a window (32-bit fallback)
	procGetWindowPlacement       = user32.NewProc("GetWindowPlacement")       // Retrieves the placement of a window
	procGetWindowRect            = user32.NewProc("GetWindowRect")            // Retrieves the bounding rectangle of a window
	procGetWindowText            = user32.NewProc("GetWindowTextW")           // Retrieves the title of a window
	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId") // Retrieves the thread and process ID of a window
	procIsWindowVisible          = user32.NewProc("IsWindowVisible")          // Checks if a window is visible
	procPostMessage              = user32.NewProc("PostMessageW")             // Posts a message to a window's message queue
	procSendMessage              = user32.NewProc("SendMessageW")             // Sends a message to a window and waits for the result
	procSetForegroundWindow      = user32.NewProc("SetForegroundWindow")      // Brings a window to the foreground
	procSetWindowPlacement       = user32.NewProc("SetWindowPlacement")       // Sets the placement of a window
	procSetWindowPos             = user32.NewProc("SetWindowPos")             // Sets the position and size of a window
	procShowWindow               = user32.NewProc("ShowWindow")               // Shows or hides a window

)

// Constants for window attributes and styles
const (
	DWMWA_EXTENDED_FRAME_BOUNDS       = 9                // Extended frame bounds for DWM
	GWL_EXSTYLE                       = -20              // Index for extended window styles
	GWL_STYLE                         = -16              // Index for window styles
	HWND_TOP                          = 0                // Place window at top of Z order
	HWND_TOPMOST                      = ^uintptr(0)      // -1 in two's complement (all bits set)
	HWND_NOTOPMOST                    = ^uintptr(0) - 1  // -2 in two's complement (all bits set except least significant)
	CHILDID_SELF                      = 0                // Child ID for the window itself
	OBJID_WINDOW                      = 0x00000000       // Object ID for a window
	PROCESS_QUERY_LIMITED_INFORMATION = 0x1000           // Access rights for OpenProcess
	SC_MOVE                           = 0xF010           // System command to move a window
	SC_RESTORE                        = 0xF120           // System command to restore a window
	SM_CXSCREEN                       = 0                // Width of the primary display
	SM_CXVIRTUALSCREEN                = 78               // Width of the virtual screen
	SM_CYSCREEN                       = 1                // Height of the primary display
	SM_CYVIRTUALSCREEN                = 79               // Height of the virtual screen
	SM_XVIRTUALSCREEN                 = 76               // X-coordinate of the virtual screen
	SM_YVIRTUALSCREEN                 = 77               // Y-coordinate of the virtual screen
	SW_FORCEMINIMIZE                  = 11               // Force minimize window
	SW_MAXIMIZE                       = 3                // Maximize window
	SW_MINIMIZE                       = 6                // Minimize window
	SW_RESTORE                        = 9                // Restore window if minimized
	SW_SHOW                           = 5                // Show window
	SW_SHOWMAXIMIZED                  = 3                // Show window as maximized
	SW_SHOWMINIMIZED                  = 2                // Show window as minimized
	SW_SHOWNORMAL                     = 1                // Show window in normal state
	SWP_ASYNCWINDOWPOS                = 0x4000           // Asynchronous window positioning
	SWP_FRAMECHANGED                  = 0x0020           // The frame changed; send WM_NCCALCSIZE
	SWP_DRAWFRAME                     = SWP_FRAMECHANGED // Draw the frame (if the window has a frame)
	SWP_NOACTIVATE                    = 0x0010           // Do not activate the window
	SWP_NOMOVE                        = 0x0002           // Do not change the position of the window
	SWP_NOSIZE                        = 0x0001           // Do not change the size of the window
	SWP_NOZORDER                      = 0x0004           // Do not change the Z order of the window
	SWP_SHOWWINDOW                    = 0x0040           // Show the window when setting position and size
	SWP_STATECHANGED                  = 0x4000           // The window's state has changed; send WM_WINDOWPOSCHANGED
	WS_EX_TOPMOST                     = 0x00000008       // Extended window style for topmost windows
	WM_SYSCOMMAND                     = 0x0112           // System command message
)

// Global callback for window enumeration to prevent memory leaks
var globalEnumCallback uintptr

// Shared windows slice for callback communication
var enumeratedWindows []WindowInfo
var enumMutex sync.Mutex

// init function to create the callback once
func init() {
	globalEnumCallback = syscall.NewCallback(enumWindowsCallbackFunc)
}

// enumWindowsCallbackFunc is the callback function for EnumWindows
func enumWindowsCallbackFunc(hwnd syscall.Handle, lparam uintptr) uintptr {
	debug := false // Get debug flag from context or use false as default

	// Add error recovery for individual window processing
	defer func() {
		if r := recover(); r != nil {
			log(true, "Panic in window enumeration callback for handle", hwnd, ":", r)
			// Continue enumeration despite the error
		}
	}()

	// Double-check window validity before processing
	if hwnd == 0 || !isValidWindow(hwnd) {
		return 1 // Continue enumeration
	}

	if isWindowVisible(hwnd) {
		info := getWindowInfo(hwnd)
		width := int(info.WindowRect.Right - info.WindowRect.Left)
		height := int(info.WindowRect.Bottom - info.WindowRect.Top)
		if width > 8 && height > 8 {
			log(debug, "Found window via handle:", info.Handle)
			log(debug, "- Title       :", info.Title)
			log(debug, "- ClassName   :", info.ClassName)
			log(debug, "- Executable  :", info.Executable)
			log(debug, "- Style       :", info.Style)
			log(debug, "- ExStyle     :", info.ExStyle)
			log(debug, "- ClientRect  :", info.ClientRect)
			log(debug, "- WindowRect  :", info.WindowRect)

			// Thread-safe append to shared slice
			enumMutex.Lock()
			enumeratedWindows = append(enumeratedWindows, info)
			enumMutex.Unlock()
		}
	}
	return 1 // Continue enumeration
}

// EnumerateWindows retrieves a list of all visible windows on the desktop.
// It returns a slice of WindowInfo structs containing the handle, title, class name, and process ID of each window.
// It uses the EnumWindows function to enumerate all top-level windows.
// The callback function filters out invisible windows and collects the necessary information.
// It returns an error if the enumeration fails.
func EnumerateWindows() ([]WindowInfo, error) {
	debug := false
	log(debug, "Enumerating visible windows.")

	// Reset the shared windows slice
	enumMutex.Lock()
	enumeratedWindows = enumeratedWindows[:0] // Clear slice but keep capacity
	enumMutex.Unlock()

	ret, _, err := procEnumWindows.Call(globalEnumCallback, 0)
	if ret == 0 {
		log(true, "EnumWindows failed:", err)
		return nil, fmt.Errorf("EnumWindows failed: %v", err)
	}

	// Return a copy of the enumerated windows
	enumMutex.Lock()
	result := make([]WindowInfo, len(enumeratedWindows))
	copy(result, enumeratedWindows)
	enumMutex.Unlock()

	return result, nil
}

// isWindowVisible checks if a window is visible.
func isWindowVisible(hwnd syscall.Handle) bool {
	debug := false
	log(debug, "Checking if window is visible for handle:", hwnd)
	ret, _, _ := procIsWindowVisible.Call(uintptr(hwnd))
	visible := ret != 0
	log(debug, "Window visibility:", visible)
	return visible
}

// getWindowInfo retrieves the title, class name, and process ID of a window.
// It uses GetWindowText to get the title, GetClassName to get the class name
func getWindowInfo(hwnd syscall.Handle) WindowInfo {
	debug := false
	log(debug, "Getting window info for handle:", hwnd)

	// Add comprehensive error recovery for this function
	defer func() {
		if r := recover(); r != nil {
			log(true, "Panic in getWindowInfo for handle", hwnd, ":", r)
		}
	}()

	// Validate the handle at the start
	if hwnd == 0 {
		log(debug, "Invalid window handle: 0")
		return WindowInfo{Handle: hwnd}
	}

	const maxWinText = 256

	// Initialize with safe defaults
	var title, className string
	var processID uint32

	// Only proceed with API calls if the window appears to be valid
	if isValidWindow(hwnd) {
		// Get window title
		titleBuf := make([]uint16, maxWinText)
		ret, _, err := procGetWindowText.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&titleBuf[0])), uintptr(len(titleBuf)))
		if ret == 0 {
			log(debug, "GetWindowText failed:", err) // debug since it is common to fail
		} else {
			title = syscall.UTF16ToString(titleBuf)
		}
		log(debug, "Window title:", title)

		// Get class name
		classBuf := make([]uint16, maxWinText)
		ret, _, err = procGetClassName.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&classBuf[0])), uintptr(len(classBuf)))
		if ret == 0 {
			log(debug, "GetClassName failed:", err)
		} else {
			className = syscall.UTF16ToString(classBuf)
		}
		log(debug, "Window class name:", className)

		// Get process ID
		ret, _, err = procGetWindowThreadProcessId.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&processID)))
		if ret == 0 {
			log(debug, "GetWindowThreadProcessId failed:", err)
		}
		log(debug, "Process ID:", processID)
	} else {
		log(debug, "Skipping API calls for invalid window handle:", hwnd)
	}

	// Get process executable path - handle errors gracefully
	var exePath string
	if processID != 0 {
		path, err := getProcessExecutablePath(processID)
		if err != nil {
			log(debug, "Failed to get executable path for PID", processID, ":", err)
			exePath = fmt.Sprintf("PID:%d", processID) // Use PID as fallback
		} else {
			exePath = path
		}
	}
	log(debug, "Process executable path:", exePath)

	// Get window styles and extended styles - only if window is still valid
	var style, exstyle uintptr
	if isValidWindow(hwnd) {
		style, _ = getWindowLong(hwnd, GWL_STYLE)
		exstyle, _ = getWindowLong(hwnd, GWL_EXSTYLE)
	}
	log(debug, "Window styles:", style, "Extended styles:", exstyle)

	// Get client rectangle - only if window is still valid
	var clientRect *RECT
	if isValidWindow(hwnd) {
		clientRect, _ = getClientRect(hwnd)
	}
	if clientRect == nil {
		clientRect = &RECT{} // Use empty rectangle if failed
	}
	log(debug, "Client rectangle:", clientRect)

	// Get window rectangle
	var windowRect RECT
	if isValidWindow(hwnd) {
		ret, _, err := procGetWindowRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&windowRect)))
		if ret == 0 {
			log(true, "GetWindowRect failed:", err)
		}
	}

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

// isValidWindow checks if a window handle is still valid
func isValidWindow(hwnd syscall.Handle) bool {
	if hwnd == 0 {
		return false
	}

	// Use a safer approach: check if window is visible first (faster and safer)
	// If this fails, the window is definitely invalid
	ret1, _, _ := procIsWindowVisible.Call(uintptr(hwnd))
	if ret1 == 0 {
		// Window is not visible, but that doesn't mean it's invalid
		// Let's do a secondary check with GetWindowRect
		var rect RECT
		ret2, _, _ := procGetWindowRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&rect)))
		return ret2 != 0
	}

	// Window is visible, so it's valid
	return true
}

// getWindowPosition retrieves the position and size of a window.
// It uses GetWindowRect to get the window's bounding rectangle.
func getWindowPosition(hwnd syscall.Handle) (*WindowPosition, error) {
	debug := false
	log(debug, "Getting window position for handle:", hwnd)

	// Validate handle first
	if !isValidWindow(hwnd) {
		return nil, fmt.Errorf("invalid or destroyed window handle: %v", hwnd)
	}

	var rect RECT
	ret, _, err := procGetWindowRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&rect)))
	if ret == 0 {
		log(true, "GetWindowRect failed:", err)
		return nil, fmt.Errorf("GetWindowRect failed: %v", err)
	}
	log(debug, "Window rectangle:", rect)
	return &WindowPosition{
		X:      int(rect.Left),
		Y:      int(rect.Top),
		Width:  int(rect.Right - rect.Left),
		Height: int(rect.Bottom - rect.Top),
	}, nil
}

// MoveWindowAccurate moves a window to a specified position and size.
// It uses multiple techniques to work around elevation restrictions.
func MoveWindowAccurate(hwnd syscall.Handle, x, y, width, height int) error {
	debug := false
	log(debug, "Moving window:", hwnd, "to position:", x, y, "with size:", width, height)

	// Validate handle first
	if !isValidWindow(hwnd) {
		return fmt.Errorf("invalid or destroyed window handle: %v", hwnd)
	}

	// Get current position and size
	pos, err := getWindowPosition(hwnd)
	if err != nil {
		log(true, "-> Failed to get current window position:", err)
		return fmt.Errorf("failed to get current window position: %v", err)
	}
	if pos.X == x && pos.Y == y && pos.Width == width && pos.Height == height {
		log(debug, "-> Window already at desired position and size.")
		return nil // Already at desired position and size
	}

	// Flags for SetWindowPos
	flags := SWP_SHOWWINDOW

	// Try the standard method
	if trySetWindowPos(hwnd, x, y, width, height, uint32(flags)) {
		log(debug, "Window moved successfully using standard SetWindowPos.")
		return nil
	}
	log(true, "Standard SetWindowPos failed, trying AttachThreadInput method.")

	// Try AttachThreadInput method
	if tryAttachThreadInputForSetPos(hwnd, x, y, width, height, uint32(flags)) {
		log(debug, "Window moved successfully using AttachThreadInput.")
		return nil
	}
	log(true, "AttachThreadInput method failed, trying minimize/restore trick.")

	// Try minimize/restore method
	if tryMinimizeRestoreForSetPos(hwnd, x, y, width, height, uint32(flags)) {
		log(debug, "Window moved successfully using minimize/restore trick.")
		return nil
	}
	log(true, "Minimize/restore method failed, trying SetWindowPlacement method.")

	// Try SetWindowPlacement method
	if trySetWindowPlacementForSetPos(hwnd, x, y, width, height) {
		log(debug, "Window moved successfully using SetWindowPlacement.")
		return nil
	}
	log(true, "SetWindowPlacement method failed, trying async window positioning.")

	// Try async window positioning
	if tryAsyncWindowPos(hwnd, x, y, width, height) {
		log(debug, "Window moved successfully using async window positioning.")
		return nil
	}
	log(true, "Async window positioning failed, trying PostMessage approach.")

	// Try PostMessage approach
	if tryPostMessageApproach(hwnd, x, y, width, height) {
		log(debug, "Window moved successfully using PostMessage approach.")
		return nil
	}
	log(true, "PostMessage approach failed, trying SendMessage approach.")

	// Try SendMessage approach
	if trySendMessageApproach(hwnd, x, y, width, height) {
		log(debug, "Window moved successfully using SendMessage approach.")
		return nil
	}
	log(true, "SendMessage approach failed, trying indirect approach.")

	// Try indirect approach
	if tryIndirectApproach(hwnd, x, y, width, height) {
		log(debug, "Window moved successfully using indirect approach.")
		return nil
	}
	log(true, "Indirect approach failed, trying combined approach.")

	// Try combined approach
	if tryCombinedApproach(hwnd, x, y, width, height) {
		log(debug, "Window moved successfully using combined approach.")
		return nil
	}
	log(true, "Combined approach failed, trying Accessibility approach.")

	// Try Accessibility approach
	if tryAccessibilityApproach(hwnd, x, y, width, height) {
		log(debug, "Window moved successfully using Accessibility approach.")
		return nil
	}
	log(true, "Accessibility approach failed, trying Windows UI Automation approach.")

	// Try Windows UI Automation approach
	if tryWindowsAutomationApproach(hwnd, x, y, width, height) {
		log(debug, "Window moved successfully using Windows UI Automation approach.")
		return nil
	}

	return fmt.Errorf("failed to move window after multiple attempts")
}

// trySetWindowPlacementForSetPos uses SetWindowPlacement to set window position
func trySetWindowPlacementForSetPos(hwnd syscall.Handle, x, y, width, height int) bool {
	debug := true
	log(debug, "Trying SetWindowPlacement for handle:", hwnd)

	// Get current window placement
	var placement WINDOWPLACEMENT
	placement.Length = uint32(unsafe.Sizeof(placement))
	ret, _, err := procGetWindowPlacement.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&placement)))
	if ret == 0 {
		log(true, "GetWindowPlacement failed:", err)
		return false
	}

	// Set the normal position to the desired position and size
	placement.RcNormalPosition.Left = int32(x)
	placement.RcNormalPosition.Top = int32(y)
	placement.RcNormalPosition.Right = int32(x + width)
	placement.RcNormalPosition.Bottom = int32(y + height)

	// If window is minimized, we'll restore it temporarily
	wasMinimized := (placement.ShowCmd == SW_SHOWMINIMIZED)
	if wasMinimized {
		placement.ShowCmd = SW_RESTORE
	}

	// Set the window placement
	ret, _, err = procSetWindowPlacement.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&placement)))
	if ret == 0 {
		if errno, ok := err.(syscall.Errno); ok && errno != 0 {
			log(true, "SetWindowPlacement failed:", errno)
		}
		return false
	}

	// If window was originally minimized, minimize it again
	if wasMinimized {
		ret, _, err = procShowWindow.Call(uintptr(hwnd), SW_MINIMIZE)
		if ret == 0 {
			if errno, ok := err.(syscall.Errno); ok && errno != 0 {
				log(true, "ShowWindow (minimize) failed:", errno)
			}
			return false
		}
	}

	return true
}

// tryCombinedApproach combines multiple techniques to set window position
func tryCombinedApproach(hwnd syscall.Handle, x, y, width, height int) bool {
	debug := true
	log(debug, "Trying combined approach for handle:", hwnd)

	// Step 1: Minimize the window
	ret, _, err := procShowWindow.Call(uintptr(hwnd), SW_MINIMIZE)
	if ret == 0 {
		if errno, ok := err.(syscall.Errno); ok && errno != 0 {
			log(true, "ShowWindow (minimize) failed:", errno)
		}
		return false
	}

	// Step 2: Wait for minimize to complete
	time.Sleep(200 * time.Millisecond)

	// Step 3: Try to set position while minimized (this might work for some windows)
	ret, _, err = procSetWindowPos.Call(
		uintptr(hwnd),
		0, // HWND_TOP
		uintptr(x),
		uintptr(y),
		uintptr(width),
		uintptr(height),
		SWP_SHOWWINDOW,
	)
	if ret != 0 {
		if errno, ok := err.(syscall.Errno); ok && errno != 0 {
			log(true, "SetWindowPos while minimized failed:", errno)
			return false
		}
		log(debug, "SetWindowPos while minimized succeeded.")
		// If successful, restore the window
		procShowWindow.Call(uintptr(hwnd), SW_RESTORE)
		return true
	}

	// Step 4: Get thread IDs for AttachThreadInput
	var targetThreadID uint32
	ret, _, err = procGetWindowThreadProcessId.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&targetThreadID)))
	if ret == 0 {
		if errno, ok := err.(syscall.Errno); ok && errno != 0 {
			log(true, "GetWindowThreadProcessId failed:", errno)
		}
		return false
	}

	currentThreadID, _, err := procGetCurrentThreadId.Call()
	if currentThreadID == 0 {
		if errno, ok := err.(syscall.Errno); ok && errno != 0 {
			log(true, "GetCurrentThreadId failed:", errno)
		}
		return false
	}

	// Step 5: Attach threads
	attachRet, _, err := procAttachThreadInput.Call(uintptr(currentThreadID), uintptr(targetThreadID), 1)
	if attachRet == 0 {
		if errno, ok := err.(syscall.Errno); ok && errno != 0 {
			log(true, "AttachThreadInput failed:", errno)
		}
		return false
	}

	// Step 6: Set position with attached threads
	ret, _, err = procSetWindowPos.Call(
		uintptr(hwnd),
		0, // HWND_TOP
		uintptr(x),
		uintptr(y),
		uintptr(width),
		uintptr(height),
		SWP_SHOWWINDOW,
	)

	// Step 7: Detach threads
	procAttachThreadInput.Call(uintptr(currentThreadID), uintptr(targetThreadID), 0)

	if ret == 0 {
		if errno, ok := err.(syscall.Errno); ok && errno != 0 {
			log(true, "SetWindowPos with AttachThreadInput failed:", errno)
		}
		return false
	}

	// Step 8: Restore the window
	ret, _, err = procShowWindow.Call(uintptr(hwnd), SW_RESTORE)
	if ret == 0 {
		if errno, ok := err.(syscall.Errno); ok && errno != 0 {
			log(true, "ShowWindow (restore) failed:", errno)
		}
		return false
	}

	return true
}

// openProcess opens a handle to a process by its PID.
// It uses OpenProcess with PROCESS_QUERY_LIMITED_INFORMATION access.
func openProcess(pid uint32) (syscall.Handle, error) {
	// Add validation for PID
	if pid == 0 {
		return 0, fmt.Errorf("invalid PID: 0")
	}

	h, _, err := procOpenProcess.Call(uintptr(PROCESS_QUERY_LIMITED_INFORMATION), uintptr(0), uintptr(pid))
	if h == 0 {
		if errno, ok := err.(syscall.Errno); ok && errno != 0 {
			// Don't log "Access is denied" errors as critical - they're common for system/protected processes
			if errno == 5 { // ERROR_ACCESS_DENIED
				log(false, "OpenProcess access denied for PID", pid, "- this is normal for protected processes")
				return 0, fmt.Errorf("OpenProcess access denied for PID %d", pid)
			}
			log(true, "OpenProcess failed for PID", pid, ":", errno)
			return 0, fmt.Errorf("OpenProcess failed for PID %d: %v", pid, errno)
		}
		log(true, "OpenProcess failed for PID", pid)
		return 0, fmt.Errorf("OpenProcess failed for PID %d", pid)
	}
	return syscall.Handle(h), nil
}

// closeHandle closes a handle to a process.
// It uses CloseHandle to release the handle.
func closeHandle(handle syscall.Handle) {
	debug := false
	r1, _, err := procCloseHandle.Call(uintptr(handle))
	if r1 == 0 {
		if errno, ok := err.(syscall.Errno); ok && errno != 0 {
			log(true, "CloseHandle failed:", errno)
			return
		}
		log(true, "CloseHandle failed")
		return
	}
	log(debug, "Closed handle:", handle)
}

// getProcessExecutablePath retrieves the executable path of a process by its PID.
// It uses GetModuleFileNameExW to get the executable path of the main module of the process.
// It returns the path as a string or an error if it fails.
func getProcessExecutablePath(pid uint32) (string, error) {
	debug := false
	log(debug, "Getting executable path for PID:", pid)

	// Validate PID
	if pid == 0 {
		return "", fmt.Errorf("invalid PID: 0")
	}

	// Add panic recovery for this function
	defer func() {
		if r := recover(); r != nil {
			log(true, "Panic in getProcessExecutablePath for PID", pid, ":", r)
		}
	}()

	handle, err := openProcess(pid)
	if err != nil || handle == 0 {
		// Return a more descriptive error for the caller to handle gracefully
		return "", fmt.Errorf("OpenProcess failed for PID %d: %v", pid, err)
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
		return "", fmt.Errorf("GetModuleFileNameExW failed: %w", err)
	}
	path := syscall.UTF16ToString(buf[:ret])
	log(debug, "Executable path:", path)
	return path, nil
}

// getWindowLong retrieves a specified value associated with a window.
func getWindowLong(hwnd syscall.Handle, index int32) (uintptr, error) {
	debug := true

	// Validate handle first to prevent crashes
	if hwnd == 0 {
		return 0, fmt.Errorf("invalid window handle: 0")
	}

	ret, _, err := procGetWindowLongPtrW.Call(uintptr(hwnd), uintptr(index))
	if ret != 0 {
		return ret, nil
	}
	// Try 32-bit fallback for older Windows
	ret32, _, err32 := procGetWindowLongW.Call(uintptr(hwnd), uintptr(index))
	if ret32 != 0 {
		return ret32, nil
	}
	// Both calls failed, return error
	if errno, ok := err.(syscall.Errno); ok && errno != 0 {
		log(debug, "GetWindowLongPtrW failed:", errno)
	}
	if errno32, ok := err32.(syscall.Errno); ok && errno32 != 0 {
		log(debug, "GetWindowLongW failed (fallback):", errno32)
		return 0, fmt.Errorf("GetWindowLong failed (fallback): %v", errno32)
	}
	log(debug, "GetWindowLongW failed (fallback)")
	return 0, fmt.Errorf("GetWindowLong failed")
}

// getClientRect retrieves the client area rectangle of a window.
// It uses GetClientRect to get the rectangle in client coordinates.
func getClientRect(hwnd syscall.Handle) (*RECT, error) {
	// Validate handle first
	if hwnd == 0 {
		return nil, fmt.Errorf("invalid window handle: 0")
	}

	var rect RECT
	ret, _, err := procGetClientRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&rect)))
	if ret == 0 {
		if errno, ok := err.(syscall.Errno); ok && errno != 0 {
			log(true, "GetClientRect failed:", errno)
			return nil, fmt.Errorf("GetClientRect failed: %v", errno)
		}
		log(true, "GetClientRect failed")
		return nil, fmt.Errorf("GetClientRect failed")
	}
	return &rect, nil
}

// focusWindow brings the specified window to the front using multiple techniques
// It attempts several methods to work around elevation restrictions
func focusWindow(hwnd syscall.Handle) error {
	debug := true
	log(debug, "Attempting to focus window with handle:", hwnd)

	// Validate handle first
	if !isValidWindow(hwnd) {
		return fmt.Errorf("invalid or destroyed window handle: %v", hwnd)
	}

	// Get virtual screen bounds
	virtualScreen := getVirtualScreenRect()
	log(debug, "Virtual screen bounds:", virtualScreen)

	// Get primary display bounds
	primaryDisplay := getPrimaryDisplayRect()
	log(debug, "Primary display bounds:", primaryDisplay)

	// Get current window position
	var rect RECT
	ret, _, err := procGetWindowRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&rect)))
	if ret == 0 {
		if errno, ok := err.(syscall.Errno); ok && errno != 0 {
			log(true, "GetWindowRect failed:", errno)
			return fmt.Errorf("GetWindowRect failed: %v", errno)
		}
		return fmt.Errorf("GetWindowRect failed: %v", err)
	}
	log(debug, "Current window position:", rect)

	// Check if window is outside all displays
	if !isRectOnScreen(rect, virtualScreen) {
		log(debug, "Window is outside all displays, moving to primary display")

		// Calculate new position (centered on primary display)
		width := int(rect.Right - rect.Left)
		height := int(rect.Bottom - rect.Top)
		x := int(primaryDisplay.Left) + (int(primaryDisplay.Right-primaryDisplay.Left)-width)/2
		y := int(primaryDisplay.Top) + (int(primaryDisplay.Bottom-primaryDisplay.Top)-height)/2

		// Move window to primary display
		ret, _, err = procSetWindowPos.Call(
			uintptr(hwnd),
			0, // HWND_TOP
			uintptr(x),
			uintptr(y),
			uintptr(width),
			uintptr(height),
			SWP_SHOWWINDOW,
		)
		if ret == 0 {
			log(debug, "SetWindowPos failed: %v", err)
		}
	}

	// Try multiple techniques to bring window to front
	if trySetForegroundWindow(hwnd) {
		log(debug, "Successfully brought window to foreground")
		return nil
	}

	// If SetForegroundWindow failed, try other methods
	log(debug, "SetForegroundWindow failed, trying alternative methods")

	// Method 1: AttachThreadInput technique
	if tryAttachThreadInput(hwnd) {
		log(debug, "Successfully brought window to foreground using AttachThreadInput")
		return nil
	}

	// Method 2: Minimize/Restore trick
	if tryMinimizeRestore(hwnd) {
		log(debug, "Successfully brought window to foreground using minimize/restore trick")
		return nil
	}

	// Method 3: AllowSetForegroundWindow
	if tryAllowSetForegroundWindow(hwnd) {
		log(debug, "Successfully brought window to foreground using AllowSetForegroundWindow")
		return nil
	}

	return fmt.Errorf("failed to bring window to front after multiple attempts")
}

// trySetForegroundWindow attempts the standard method
func trySetForegroundWindow(hwnd syscall.Handle) bool {
	debug := true
	log(debug, "Trying SetForegroundWindow for handle:", hwnd)
	ret, _, err := procSetForegroundWindow.Call(uintptr(hwnd))
	if ret == 0 {
		if errno, ok := err.(syscall.Errno); ok && errno != 0 {
			log(true, "SetForegroundWindow failed:", errno)
			return false
		}
		log(true, "SetForegroundWindow failed")
		return false
	}
	log(debug, "SetForegroundWindow succeeded for handle:", hwnd)
	return ret != 0
}

// tryAttachThreadInput uses thread attachment to bypass elevation restrictions
func tryAttachThreadInput(hwnd syscall.Handle) bool {
	debug := true
	log(debug, "Trying AttachThreadInput for handle:", hwnd)
	// Get thread IDs
	var targetThreadID uint32
	ret, _, err := procGetWindowThreadProcessId.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&targetThreadID)))
	if ret == 0 {
		if errno, ok := err.(syscall.Errno); ok && errno != 0 {
			log(true, "GetWindowThreadProcessId failed:", errno)
			return false
		}
		log(true, "GetWindowThreadProcessId failed")
		return false
	}

	currentThreadID, _, err := procGetCurrentThreadId.Call()
	if currentThreadID == 0 {
		if errno, ok := err.(syscall.Errno); ok && errno != 0 {
			log(true, "GetCurrentThreadId failed:", errno)
			return false
		}
		log(true, "GetCurrentThreadId failed")
		return false
	}

	// Attach threads
	attachRet, _, err := procAttachThreadInput.Call(uintptr(currentThreadID), uintptr(targetThreadID), 1)
	if attachRet == 0 {
		if errno, ok := err.(syscall.Errno); ok && errno != 0 {
			log(true, "AttachThreadInput failed:", errno)
			return false
		}
		log(true, "AttachThreadInput failed")
		return false
	}

	// Bring window to front
	defer func() {
		ret, _, err := procAttachThreadInput.Call(uintptr(currentThreadID), uintptr(targetThreadID), 0)
		if ret == 0 {
			if errno, ok := err.(syscall.Errno); ok && errno != 0 {
				log(true, "AttachThreadInput failed:", errno)
			} else {
				log(true, "AttachThreadInput failed")
			}
		} else {
			log(debug, "Attached thread input successfully")
		}
	}()

	// Try to set foreground window
	ret, _, err = procSetForegroundWindow.Call(uintptr(hwnd))
	if ret != 0 {
		if errno, ok := err.(syscall.Errno); ok && errno != 0 {
			log(true, "SetForegroundWindow failed:", errno)
			return true
		}
		log(debug, "SetForegroundWindow succeeded for handle:", hwnd)
		return true
	}

	// Alternative: Bring to top without activating
	ret, _, err = procSetWindowPos.Call(
		uintptr(hwnd),
		HWND_TOP,
		0, 0, 0, 0,
		SWP_NOSIZE|SWP_NOMOVE|SWP_NOACTIVATE,
	)
	if ret == 0 {
		if errno, ok := err.(syscall.Errno); ok && errno != 0 {
			log(true, "SetWindowPos failed:", errno)
			return false
		}
		log(true, "SetWindowPos failed")
		return false
	}
	log(debug, "SetWindowPos succeeded for handle:", hwnd)
	return ret != 0
}

// tryMinimizeRestore uses the minimize/restore trick
func tryMinimizeRestore(hwnd syscall.Handle) bool {
	debug := true
	log(debug, "Trying minimize/restore trick for handle:", hwnd)

	// Get current window placement
	var placement WINDOWPLACEMENT
	placement.Length = uint32(unsafe.Sizeof(placement))
	ret, _, err := procGetWindowPlacement.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&placement)))
	if ret == 0 {
		if errno, ok := err.(syscall.Errno); ok && errno != 0 {
			log(true, "GetWindowPlacement failed:", errno)
			return false
		}
		log(true, "GetWindowPlacement failed")
		return false
	}

	// If window is minimized, restore it
	if placement.ShowCmd == SW_SHOWMINIMIZED {
		ret, _, err = procShowWindow.Call(uintptr(hwnd), SW_RESTORE)
		if ret == 0 {
			if errno, ok := err.(syscall.Errno); ok && errno != 0 {
				log(true, "ShowWindow (restore) failed:", errno)
				return false
			}
			log(true, "ShowWindow (restore) failed")
			return false
		}
	} else {
		// Minimize the window
		ret, _, err = procShowWindow.Call(uintptr(hwnd), SW_MINIMIZE)
		if ret == 0 {
			if errno, ok := err.(syscall.Errno); ok && errno != 0 {
				log(true, "ShowWindow (minimize) failed:", errno)
				return false
			}
			log(true, "ShowWindow (minimize) failed")
			return false
		}

		// Small delay to ensure minimize completes
		time.Sleep(250 * time.Millisecond)

		// Restore the window
		ret, _, err = procShowWindow.Call(uintptr(hwnd), SW_RESTORE)
		if ret == 0 {
			if errno, ok := err.(syscall.Errno); ok && errno != 0 {
				log(true, "ShowWindow (restore) failed:", errno)
				return false
			}
			log(true, "ShowWindow (restore) failed")
			return false
		}
	}

	// Try to set foreground window
	ret, _, err = procSetForegroundWindow.Call(uintptr(hwnd))
	if ret == 0 {
		if errno, ok := err.(syscall.Errno); ok && errno != 0 {
			log(true, "SetForegroundWindow failed:", errno)
			return false
		}
		log(true, "SetForegroundWindow failed")
		return false
	}
	return ret != 0
}

// tryAllowSetForegroundWindow attempts to grant foreground permission
func tryAllowSetForegroundWindow(hwnd syscall.Handle) bool {
	debug := true
	log(debug, "Trying AllowSetForegroundWindow for handle:", hwnd)
	// Get process ID of target window
	var processID uint32
	ret, _, err := procGetWindowThreadProcessId.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&processID)))
	if ret == 0 {
		if errno, ok := err.(syscall.Errno); ok && errno != 0 {
			log(true, "GetWindowThreadProcessId failed:", errno)
			return false
		}
		log(true, "GetWindowThreadProcessId failed")
		return false
	}
	log(debug, "Process ID of target window:", processID)

	// Allow the process to set foreground window
	ret, _, err = procAllowSetForegroundWindow.Call(uintptr(processID))
	if ret == 0 {
		if errno, ok := err.(syscall.Errno); ok && errno != 0 {
			log(true, "AllowSetForegroundWindow failed:", errno)
			return false
		}
		log(true, "AllowSetForegroundWindow failed")
		return false
	}
	log(debug, "AllowSetForegroundWindow is:", ret != 0)

	// Try to set foreground window
	ret, _, err = procSetForegroundWindow.Call(uintptr(hwnd))
	if ret == 0 {
		if errno, ok := err.(syscall.Errno); ok && errno != 0 {
			log(true, "SetForegroundWindow failed:", errno)
			return false
		}
		log(true, "SetForegroundWindow failed")
		return false
	}
	log(debug, "SetForegroundWindow succeeded for handle:", hwnd)
	return ret != 0
}

// getVirtualScreenRect returns the bounding rectangle of the virtual screen
func getVirtualScreenRect() RECT {
	debug := true
	log(debug, "Getting virtual screen bounds")
	left := getSystemMetrics(SM_XVIRTUALSCREEN)
	top := getSystemMetrics(SM_YVIRTUALSCREEN)
	width := getSystemMetrics(SM_CXVIRTUALSCREEN)
	height := getSystemMetrics(SM_CYVIRTUALSCREEN)
	var retStruct RECT
	retStruct.Left = left
	retStruct.Top = top
	retStruct.Right = left + width
	retStruct.Bottom = top + height
	log(debug, "Virtual screen bounds (left, top, right, bottom):", retStruct)
	return retStruct
}

// getPrimaryDisplayRect returns the bounding rectangle of the primary display
func getPrimaryDisplayRect() RECT {
	debug := true
	log(debug, "Getting primary display bounds")
	width := getSystemMetrics(SM_CXSCREEN)
	height := getSystemMetrics(SM_CYSCREEN)
	log(debug, "Primary display bounds:", fmt.Sprintf("0, 0 %dx%d", width, height))
	return RECT{
		Left:   0,
		Top:    0,
		Right:  width,
		Bottom: height,
	}
}

// getSystemMetrics retrieves the specified system metric
func getSystemMetrics(index int32) int32 {
	debug := true
	log(debug, "Getting system metrics for index:", index)
	ret, _, err := procGetSystemMetrics.Call(uintptr(index))
	if ret == 0 {
		if errno, ok := err.(syscall.Errno); ok && errno != 0 {
			log(true, "GetSystemMetrics failed:", errno)
			return 0 // Return 0 on error
		}
		log(true, "GetSystemMetrics failed")
		return 0 // Return 0 on error
	}
	log(debug, "GetSystemMetrics succeeded for index:", index, " value:", ret)
	return int32(ret)
}

// isRectOnScreen checks if a rectangle is within the virtual screen bounds
func isRectOnScreen(rect RECT, virtualScreen RECT) bool {
	// Check if the window is completely to the left, right, above, or below the virtual screen
	if rect.Right < virtualScreen.Left || rect.Left > virtualScreen.Right ||
		rect.Bottom < virtualScreen.Top || rect.Top > virtualScreen.Bottom {
		return false
	}
	return true
}

// trySetWindowPos attempts the standard method to set window position
func trySetWindowPos(hwnd syscall.Handle, x, y, width, height int, flags uint32) bool {
	ret, _, _ := procSetWindowPos.Call(
		uintptr(hwnd),
		0, // HWND_TOP
		uintptr(x),
		uintptr(y),
		uintptr(width),
		uintptr(height),
		uintptr(flags),
	)
	return ret != 0
}

// tryAttachThreadInputForSetPos uses thread attachment to set window position
func tryAttachThreadInputForSetPos(hwnd syscall.Handle, x, y, width, height int, flags uint32) bool {
	// Get thread IDs
	var targetThreadID uint32
	ret, _, _ := procGetWindowThreadProcessId.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&targetThreadID)))
	if ret == 0 {
		return false
	}

	currentThreadID, _, _ := procGetCurrentThreadId.Call()
	if currentThreadID == 0 {
		return false
	}

	// Attach threads
	attachRet, _, _ := procAttachThreadInput.Call(uintptr(currentThreadID), uintptr(targetThreadID), 1)
	if attachRet == 0 {
		return false
	}

	// Detach when done
	defer func() {
		procAttachThreadInput.Call(uintptr(currentThreadID), uintptr(targetThreadID), 0)
	}()

	// Try to set window position
	ret, _, _ = procSetWindowPos.Call(
		uintptr(hwnd),
		0, // HWND_TOP
		uintptr(x),
		uintptr(y),
		uintptr(width),
		uintptr(height),
		uintptr(flags),
	)
	return ret != 0
}

// tryMinimizeRestoreForSetPos uses the minimize/restore trick to set window position
func tryMinimizeRestoreForSetPos(hwnd syscall.Handle, x, y, width, height int, flags uint32) bool {
	// Get current window placement
	var placement WINDOWPLACEMENT
	placement.Length = uint32(unsafe.Sizeof(placement))
	ret, _, _ := procGetWindowPlacement.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&placement)))
	if ret == 0 {
		return false
	}

	// If window is minimized, we need to restore it first to set position
	if placement.ShowCmd == SW_SHOWMINIMIZED {
		// Restore the window
		ret, _, _ = procShowWindow.Call(uintptr(hwnd), SW_RESTORE)
		if ret == 0 {
			return false
		}

		// Set the position
		ret, _, _ = procSetWindowPos.Call(
			uintptr(hwnd),
			0, // HWND_TOP
			uintptr(x),
			uintptr(y),
			uintptr(width),
			uintptr(height),
			uintptr(flags),
		)
		if ret == 0 {
			return false
		}

		// Minimize it again
		ret, _, _ = procShowWindow.Call(uintptr(hwnd), SW_MINIMIZE)
		return ret != 0
	} else {
		// If not minimized, minimize and then restore to the same state
		ret, _, _ = procShowWindow.Call(uintptr(hwnd), SW_MINIMIZE)
		if ret == 0 {
			return false
		}

		// Small delay to ensure minimize completes
		time.Sleep(100 * time.Millisecond)

		// Restore the window to its previous state
		ret, _, _ = procShowWindow.Call(uintptr(hwnd), placement.ShowCmd)
		if ret == 0 {
			return false
		}

		// Set the position
		ret, _, _ = procSetWindowPos.Call(
			uintptr(hwnd),
			0, // HWND_TOP
			uintptr(x),
			uintptr(y),
			uintptr(width),
			uintptr(height),
			uintptr(flags),
		)
		return ret != 0
	}
}

// tryPostMessageApproach uses window messages to manipulate the window
func tryPostMessageApproach(hwnd syscall.Handle, x, y, width, height int) bool {
	debug := true
	log(debug, "Trying PostMessage approach for handle:", hwnd)

	// Step 1: Restore the window if minimized
	ret, _, err := procPostMessage.Call(uintptr(hwnd), WM_SYSCOMMAND, SC_RESTORE, 0)
	if ret == 0 {
		if errno, ok := err.(syscall.Errno); ok && errno != 0 {
			log(true, "PostMessage (SC_RESTORE) failed:", errno)
		}
		log(true, "PostMessage (SC_RESTORE) failed")
	}

	// Step 2: Small delay to allow restore to complete
	time.Sleep(100 * time.Millisecond)

	// Step 3: Try to set position with async flag
	ret, _, err = procSetWindowPos.Call(
		uintptr(hwnd),
		0, // HWND_TOP
		uintptr(x),
		uintptr(y),
		uintptr(width),
		uintptr(height),
		SWP_SHOWWINDOW|SWP_ASYNCWINDOWPOS,
	)
	if ret != 0 {
		return true
	} else if errno, ok := err.(syscall.Errno); ok && errno != 0 {
		log(true, "SetWindowPos (PostMessage) failed:", errno)
	}
	log(true, "SetWindowPos (PostMessage) failed")

	// Step 4: Try without topmost flag
	ret, _, _ = procSetWindowPos.Call(
		uintptr(hwnd),
		HWND_NOTOPMOST,
		uintptr(x),
		uintptr(y),
		uintptr(width),
		uintptr(height),
		SWP_SHOWWINDOW|SWP_ASYNCWINDOWPOS,
	)

	return ret != 0
}

// trySendMessageApproach uses SendMessage to directly manipulate the window
func trySendMessageApproach(hwnd syscall.Handle, x, y, width, height int) bool {
	debug := true
	log(debug, "Trying SendMessage approach for handle:", hwnd)

	// Step 1: Restore the window if minimized
	ret, _, err := procSendMessage.Call(uintptr(hwnd), WM_SYSCOMMAND, SC_RESTORE, 0)
	if ret == 0 {
		if errno, ok := err.(syscall.Errno); ok && errno != 0 {
			log(true, "SendMessage (SC_RESTORE) failed:", errno)
		}
	}

	// Step 2: Small delay to allow restore to complete
	time.Sleep(100 * time.Millisecond)

	// Step 3: Try to set position
	ret, _, _ = procSetWindowPos.Call(
		uintptr(hwnd),
		0, // HWND_TOP
		uintptr(x),
		uintptr(y),
		uintptr(width),
		uintptr(height),
		SWP_SHOWWINDOW,
	)

	return ret != 0
}

// tryAsyncWindowPos uses async window positioning
func tryAsyncWindowPos(hwnd syscall.Handle, x, y, width, height int) bool {
	debug := true
	log(debug, "Trying async window positioning for handle:", hwnd)

	// Try with async flag
	ret, _, _ := procSetWindowPos.Call(
		uintptr(hwnd),
		0, // HWND_TOP
		uintptr(x),
		uintptr(y),
		uintptr(width),
		uintptr(height),
		SWP_SHOWWINDOW|SWP_ASYNCWINDOWPOS,
	)
	if ret != 0 {
		return true
	}

	// Try with async flag and no z-order change
	ret, _, _ = procSetWindowPos.Call(
		uintptr(hwnd),
		0, // HWND_TOP
		uintptr(x),
		uintptr(y),
		uintptr(width),
		uintptr(height),
		SWP_SHOWWINDOW|SWP_ASYNCWINDOWPOS|SWP_NOZORDER,
	)

	return ret != 0
}

// tryIndirectApproach uses indirect methods that might work with elevated windows
func tryIndirectApproach(hwnd syscall.Handle, x, y, width, height int) bool {
	debug := true
	log(debug, "Trying indirect approach for handle:", hwnd)

	// Step 1: Get the window's current style (unused but kept for consistency)
	_, err := getWindowLong(hwnd, GWL_STYLE)
	if err != nil {
		log(true, "Failed to get window style:", err)
		return false
	}

	// Step 2: Temporarily remove WS_EX_TOPMOST style if present
	exStyle, err := getWindowLong(hwnd, GWL_EXSTYLE)
	if err != nil {
		log(true, "Failed to get extended window style:", err)
		return false
	}
	if exStyle&WS_EX_TOPMOST != 0 { // WS_EX_TOPMOST
		ret, _, err := procSetWindowPos.Call(
			uintptr(hwnd),
			HWND_NOTOPMOST,
			0, 0, 0, 0,
			SWP_NOMOVE|SWP_NOSIZE,
		)
		if ret == 0 {
			log(true, "Failed to remove topmost style:", err)
			return false
		}
		// Small delay to let Windows process the change
		time.Sleep(50 * time.Millisecond)
	}

	// Step 3: Try to set position with minimal flags
	ret, _, _ := procSetWindowPos.Call(
		uintptr(hwnd),
		HWND_TOP,
		uintptr(x), uintptr(y),
		uintptr(width), uintptr(height),
		SWP_NOZORDER|SWP_NOACTIVATE|SWP_ASYNCWINDOWPOS,
	)
	if ret != 0 {
		return true
	}

	// Step 4: Try with show window flag
	ret, _, err = procSetWindowPos.Call(
		uintptr(hwnd),
		HWND_TOP,
		uintptr(x), uintptr(y),
		uintptr(width), uintptr(height),
		SWP_SHOWWINDOW|SWP_NOZORDER|SWP_ASYNCWINDOWPOS,
	)
	if ret == 0 {
		log(true, "SetWindowPos (indirect) failed:", err)
		return false
	}
	log(debug, "SetWindowPos (indirect) succeeded for handle:", hwnd)

	return true
}

// tryAccessibilityApproach uses Windows Accessibility API to move the window
func tryAccessibilityApproach(hwnd syscall.Handle, x, y, width, height int) bool {
	debug := true
	log(debug, "Trying Accessibility approach for handle:", hwnd)

	// Initialize COM
	procCoInitialize.Call(uintptr(0))
	defer procCoUninitialize.Call()

	// Get the IAccessible interface for the window
	var pAccessible *IAccessible
	var varChild uintptr

	// Create the IID for IAccessible
	iid := &syscall.GUID{
		Data1: 0x618736E0,
		Data2: 0x3C3D,
		Data3: 0x11CF,
		Data4: [8]byte{0x81, 0x0C, 0x00, 0xAA, 0x00, 0x38, 0x9B, 0x71},
	}

	ret, _, _ := procAccessibleObjectFromWindow.Call(
		uintptr(hwnd),
		uintptr(OBJID_WINDOW),
		uintptr(unsafe.Pointer(iid)),
		uintptr(unsafe.Pointer(&pAccessible)),
		uintptr(unsafe.Pointer(&varChild)),
	)

	if ret != 0 {
		log(true, "AccessibleObjectFromWindow failed with error:", ret)
		return false
	}

	// Make sure to release the interface when done
	if pAccessible != nil && pAccessible.vtbl != nil {
		syscall.Syscall(pAccessible.vtbl.Release, 1, uintptr(unsafe.Pointer(pAccessible)), 0, 0)
	}

	// Try to set position with async flag
	ret, _, _ = procSetWindowPos.Call(
		uintptr(hwnd),
		0, // HWND_TOP
		uintptr(x),
		uintptr(y),
		uintptr(width),
		uintptr(height),
		SWP_ASYNCWINDOWPOS|SWP_NOZORDER|SWP_NOACTIVATE,
	)

	if ret != 0 {
		return true
	}

	// Try with show window flag
	ret, _, _ = procSetWindowPos.Call(
		uintptr(hwnd),
		0, // HWND_TOP
		uintptr(x),
		uintptr(y),
		uintptr(width),
		uintptr(height),
		SWP_ASYNCWINDOWPOS|SWP_SHOWWINDOW,
	)

	return ret != 0
}

// tryWindowsAutomationApproach uses Windows UI Automation
func tryWindowsAutomationApproach(hwnd syscall.Handle, x, y, width, height int) bool {
	debug := true
	log(debug, "Trying Windows UI Automation approach for handle:", hwnd)

	// Initialize COM
	procCoInitialize.Call(uintptr(0))
	defer procCoUninitialize.Call()

	// Try to set position with async flag and no z-order change
	ret, _, _ := procSetWindowPos.Call(
		uintptr(hwnd),
		0, // HWND_TOP
		uintptr(x),
		uintptr(y),
		uintptr(width),
		uintptr(height),
		SWP_ASYNCWINDOWPOS|SWP_NOZORDER|SWP_NOACTIVATE,
	)

	if ret != 0 {
		return true
	}

	// Try with show window flag
	ret, _, _ = procSetWindowPos.Call(
		uintptr(hwnd),
		0, // HWND_TOP
		uintptr(x),
		uintptr(y),
		uintptr(width),
		uintptr(height),
		SWP_ASYNCWINDOWPOS|SWP_SHOWWINDOW,
	)

	return ret != 0
}
