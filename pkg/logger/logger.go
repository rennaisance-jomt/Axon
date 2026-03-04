package logger

import (
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"
	"github.com/mattn/go-colorable"
)

// Debug logs a debug message
func Debug(format string, v ...interface{}) {
	defaultLogger.logFormat(LevelDebug, Gray, format, v...)
}

// Log levels
const (
	LevelInfo    = "INFO"
	LevelSuccess = "SUCCESS"
	LevelWarn    = "WARN"
	LevelError   = "ERROR"
	LevelAction  = "ACTION"
	LevelSystem  = "SYSTEM"
	LevelDebug   = "DEBUG"
)

// Colors
const (
	Reset  = "\033[0m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Purple = "\033[35m"
	Cyan   = "\033[36m"
	Gray   = "\033[37m"
	Bold   = "\033[1m"
)

// Logger provides human-readable formatted logging
type Logger struct {
	*log.Logger
}

var stderr = colorable.NewColorableStderr()

var defaultLogger = &Logger{
	log.New(stderr, "", 0),
}

// Info logs an informational message
func Info(format string, v ...interface{}) {
	defaultLogger.logFormat(LevelInfo, Cyan, format, v...)
}

// Success logs a success message
func Success(format string, v ...interface{}) {
	defaultLogger.logFormat(LevelSuccess, Green, format, v...)
}

// Warn logs a warning message
func Warn(format string, v ...interface{}) {
	defaultLogger.logFormat(LevelWarn, Yellow, format, v...)
}

// Error logs an error message
func Error(format string, v ...interface{}) {
	defaultLogger.logFormat(LevelError, Red, format, v...)
}

// Action logs a user or agent action
func Action(format string, v ...interface{}) {
	defaultLogger.logFormat(LevelAction, Purple, format, v...)
}

// System logs a system event
func System(format string, v ...interface{}) {
	defaultLogger.logFormat(LevelSystem, Blue, format, v...)
}

func (l *Logger) logFormat(level string, color string, format string, v ...interface{}) {
	timestamp := time.Now().Format("15:04:05")
	msg := fmt.Sprintf(format, v...)
	
	// Pad the level to be consistent
	paddedLevel := fmt.Sprintf("%-7s", strings.ToUpper(level))
	
	prefix := fmt.Sprintf("%s %s[%s]%s ", Gray+timestamp+Reset, color+Bold, paddedLevel, Reset)
	l.Println(prefix + msg)
}

// Section prints a bold section header
func Section(name string) {
	fmt.Fprintf(stderr, "\n%s%s  --- %s ---%s\n\n", Bold, Cyan, strings.ToUpper(name), Reset)
}

// Banner prints an animated Axon branding banner with a glitch settle effect
func Banner() {
	// Stylized ASCII Axon for "WOWNESS"
	bannerLines := []string{
		`     ___    _  __ ____   _   __`,
		`    /   |  | |/ // __ \ / | / /`,
		`   / /| |  |   // / / //  |/ / `,
		`  / ___ | /   |/ /_/ // /|  /  `,
		` /_/  |_|/_/|_|\____//_/ |_/   `,
	}

	glitchChars := "!@#$%^&*()_+-=[]{}|;:,.<>?"
	rand.Seed(time.Now().UnixNano())

	// Animation frames
	numFrames := 15
	for f := 0; f < numFrames; f++ {
		// Move cursor back to top of banner area (5 lines + initial newline)
		if f > 0 {
			fmt.Fprintf(stderr, "\033[6A") // Move up 6 lines
		} else {
			fmt.Fprintf(stderr, "\n")
		}

		// Progress from 0.0 to 1.0
		progress := float64(f) / float64(numFrames-1)

		for _, line := range bannerLines {
			glitchedLine := ""
			for _, char := range line {
				if char == ' ' {
					glitchedLine += " "
				} else {
					// Probability of settling increases with progress
					if rand.Float64() < progress {
						glitchedLine += string(char)
					} else {
						// Show a random glitch character
						glitchedLine += string(glitchChars[rand.Intn(len(glitchChars))])
					}
				}
			}
			fmt.Fprintf(stderr, "\033[K%s%s%s%s\n", Bold, Cyan, glitchedLine, Reset)
		}
		
		time.Sleep(50 * time.Millisecond)
	}

	// Print final branding lines
	fmt.Fprintf(stderr, "\n%s  %s%sAI AGENT NATIVE BROWSER%s\n", Bold+Cyan, Gray, Bold, Reset)
	fmt.Fprintf(stderr, "%s  ----------------------------------------------------------------%s\n", Gray, Reset)
	fmt.Fprintf(stderr, "%s  Not a browser for humans that AI can use.%s\n", Gray, Reset)
	fmt.Fprintf(stderr, "%s  A browser built for AI that humans can watch.%s\n\n", Gray, Reset)
}


