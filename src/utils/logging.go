package utils

import (
	"fmt"
	"log"
	"os"
)

// For low-risk function calls that would be cumbersome to deal with otherwise, such as connection closing calls in defer statements.
func LogErrors(function func() error, description string) {
	err := function()
	if err != nil {
		DefaultSafeLog(fmt.Sprintf("error while calling function with description: %v: %v", description, err))
	}
}

func SafeLog(l *log.Logger, calldepth int, msg string) {
	if err := l.Output(calldepth, msg); err != nil {
		fmt.Fprintf(os.Stderr, "logging failed: %v\n", err)
	}
}

func DefaultSafeLog(msg string) {
	SafeLog(log.Default(), 100, msg)
}

func FDefaultSafeLog(fmsg string, args ...any) {
	SafeLog(log.Default(), 100, fmt.Sprintf(fmsg, args...))
}
