package utils

import (
	"fmt"
	"log"
	"os"
)

const logDepth int = 100

// For low-risk function calls that would be cumbersome to deal with otherwise, such as connection closing calls in defer statements.
func LogErrors(function func() error, description string) {
	if function == nil {
		FDefaultSafeLog("error while calling log on nonexistant function with description %v", description)
	}
	err := function()
	if err != nil {
		DefaultSafeLog(fmt.Sprintf("error while calling function with description: %v: %v", description, err))
	}
}

func SafeLog(l *log.Logger, calldepth int, msg string) {
	err := l.Output(calldepth, msg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "logging failed: %v\n", err)
	}
}

func DefaultSafeLog(msg string) {
	SafeLog(log.Default(), logDepth, msg)
}

func FDefaultSafeLog(fmsg string, args ...any) {
	SafeLog(log.Default(), logDepth, fmt.Sprintf(fmsg, args...))
}
