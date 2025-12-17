package main

import (
	"fmt"
	"log"
	"os"
	"path"
	"time"
)

var (
	logger  *log.Logger
	logFile *os.File
)

func initLogger(logPath string) {

	err := os.MkdirAll(logPath, os.ModePerm)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create log directory: %v\n", err)
		os.Exit(1)
	}

	timestamp := time.Now().Format("20060102") //20060102150405
	filename := fmt.Sprintf("log-%s.log", timestamp)
	fullpath := path.Join(logPath, filename)

	f, err := os.OpenFile(fullpath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open log file %s: %v\n", fullpath, err)
		// fallback to stderr logger (do not write to stdout, which breaks the TUI)
		logger = log.New(os.Stderr, "", log.LstdFlags)
		return
	}

	logFile = f
	// write only to the file; do not write to stdout to avoid breaking the TUI
	logger = log.New(f, "", log.LstdFlags)

	// confirm initialization
	logger.Printf("initialized logger, writing to %s", fullpath)
}

func closeLogger() {
	if logFile == nil {
		return
	}
	_ = logFile.Sync()
	_ = logFile.Close()
	logFile = nil
	// reset logger to stderr to avoid nil deref if used after close; never use stdout
	logger = log.New(os.Stderr, "", log.LstdFlags)
}
