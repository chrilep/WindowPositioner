package main

import (
	"fmt"
	"os"
	"runtime/debug"
	"time"

	"fyne.io/fyne/v2/dialog"
)

/*

	Panic handler/logger:
	- It catches any panic that occurs in the application and logs the reason and stack trace to the log file.
	- It can be used by calling panicHandler() at the start of the main function and every goroutine that might panic.

	Usage:

	defer panicHandler()

*/

// panicHandler catches any panic that occurs in the application.
// It logs the panic reason and stack trace to the log file.
func panicHandler() {
	if r := recover(); r != nil {
		// Safely show window and dialog only if wm and mainWindow are available
		if wm != nil && wm.mainWindow != nil {
			wm.mainWindow.Show()
			dialog.ShowError(fmt.Errorf("application crashed: %v", r), wm.mainWindow)
		}
		if fileLog != nil {
			// Write to log file if it is ready
			log(true, "HEARTBEAT: CRITICAL - Application panic detected!")
			log(true, "==== PANIC ====")
			log(true, fmt.Sprintf("Time  : %s", time.Now().Format("2006-01-02 15:04:05")))
			log(true, fmt.Sprintf("Reason: %v", r))
			log(true, string(debug.Stack()))
			log(true, "==== END PANIC ====")
		} else {
			// Append to the log file if it is not ready
			f, err := os.OpenFile(strLogFilePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
			if err != nil {
				return
			}
			defer f.Close()
			fmt.Fprintln(f, "HEARTBEAT: CRITICAL - Application panic detected!")
			fmt.Fprintln(f, "==== PANIC ====")
			fmt.Fprintf(f, "Time  : %s\n", time.Now().Format("2006-01-02 15:04:05"))
			fmt.Fprintln(f, "Reason:", r)
			fmt.Fprintln(f, string(debug.Stack()))
			fmt.Fprintln(f, "==== END PANIC ====")
			fmt.Fprintln(f, "HEARTBEAT: Application may terminate due to panic")
		}
	}
}

// panicHandler catches any panic that occurs in the application.
func safeCallback(function func()) func() {
	return func() {
		defer panicHandler()
		function()
	}
}
