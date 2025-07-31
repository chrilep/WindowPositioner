package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"
)

/*
=== Lancer's logging module ===
This module provides a simple logging mechanism that writes debug messages to both the console and a log file.
The log file is created in the user's application data directory, and it is purged at the start of the application.
It is designed to be used for debugging purposes, allowing developers to track the flow of the application.

It needs three global variables to work:
- strPublisherName: The name of the publisher, e.g. "Lancer"
- strProductName: The name of the product, e.g. "WindowPositioner"
- strVersion: The version of the product, e.g. "1.2.3.4"

Usage:

log(true, "Some var", "is", var)

*/

var strLogFilePath string // eg. <dataFolder>\Dataport\<Product>\log.txt
var fileLog *os.File
var strAppTempDir string // like %APPDATA%\Dataport\<Product>\

// log writes a message to the log file and console.
// If debug is false, it does nothing. If debug is true, it writes the message to the log file and console.
// It can take multiple arguments, which will be converted to strings.
// The first argument is the debug flag, the rest are the message parts.
// It also includes the name of the function that called it.
func log(debug bool, arrMessageParts ...any) {
	if !debug {
		return
	}
	// check if the logfile is ready
	if fileLog == nil {
		err := activateLogging()
		if err != nil {
			fmt.Println("Error activating logging:", err)
			return
		}
	}
	strParentName := `main.unknown`
	// Get the parent function's name
	ptrCaller, _, _, isSuccess := runtime.Caller(1)
	if isSuccess {
		funcCaller := runtime.FuncForPC(ptrCaller)
		if funcCaller != nil {
			strParentName = funcCaller.Name()
		}
	}
	// Convert inputs to strings
	arrMessages := make([]string, len(arrMessageParts))
	for i, v := range arrMessageParts {
		arrMessages[i] = fmt.Sprint(v)
	}
	fmt.Println(`[`+strParentName+`]`, strings.Join(arrMessages, " "))
	fmt.Fprintln(fileLog, `[`+strParentName+`]`, strings.Join(arrMessages, " "))
}

// Activates the logging module. See function log() for details.
func activateLogging() error {
	// Can have no logging since logger not ready yet!
	debug := true // set to true to enable debug logging
	switch runtime.GOOS {
	case `windows`:
		strTempDir := os.Getenv("LOCALAPPDATA")
		if strTempDir == `` {
			strTempDir = os.Getenv("TMP")
			if strTempDir == `` {
				strTempDir = os.Getenv("TEMP")
			}
		}
		strAppTempDir = strTempDir + `\` + strPublisherName + `\` + strProductName
		strLogFilePath = strAppTempDir + `\log.txt`
	case `linux`:
		strHomeDir, err := os.UserHomeDir()
		if err != nil {
			strAppTempDir = ``
		} else {
			strAppTempDir = strHomeDir + `/.local/` + strProductName
			strLogFilePath = strAppTempDir + `/log.txt`
		}
	default:
		strAppTempDir = ``
	}
	// Check if directory exists.
	if _, err := os.Stat(strAppTempDir); os.IsNotExist(err) {
		// If not, create the directory.
		os.MkdirAll(strAppTempDir, 0755)
	}
	var err error
	fileLog, err = os.OpenFile(strLogFilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		fmt.Println("Unable to open log file at '"+strLogFilePath+"':", err)
		return err
	}
	//defer fileLog.Close()
	log(debug, `Logging to `+strLogFilePath)
	return nil
}
