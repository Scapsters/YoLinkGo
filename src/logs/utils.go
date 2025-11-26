package logs

import (
	"context"
	"fmt"
	"log"
	"os"
	"runtime"
)

// For low-risk function calls that would be cumbersome to deal with otherwise, such as connection closing calls in defer statements.
func LogErrors(function func() error, description string) {
	if function == nil {
		FDefaultLog("error while calling log on nonexistant function with description %v", description)
	}
	err := function()
	if err != nil {
		FDefaultLog("error while calling function with description: %v: %v", description, err)
	}
}

// For low-risk function calls that would be cumbersome to deal with otherwise, such as connection closing calls in defer statements.
func LogErrorsWithContext(ctx context.Context, function func() error, description string) {
	if function == nil {
		ErrorWithContext(ctx, "error while calling log on nonexistant function with description %v", description)
	}
	err := function()
	if err != nil {
		ErrorWithContext(ctx, "error while calling function with description: %v: %v", description, err)
	}
}

// Format and log message to standard out.
func FDefaultLog(fmsg string, args ...any) {
	err := log.Default().Output(logDepth, fmt.Sprintf(fmsg, args...))
	if err != nil {
		fmt.Fprintf(os.Stderr, "logging failed: %v\n", err)
	}
}

func LogWithContext(ctx context.Context, level int, fstring string, args ...any) {
	logger := ctx.Value(loggerKey{})
	if logger == nil {
		FDefaultLog("[NO CONTEXT] %v", fmt.Sprintf(fstring, args...))
	} else {
		logger, ok := logger.(*JobLogger)
		if !ok {
			FDefaultLog("Unable to log, cannot cast %v from context %v into logger. intended message %v:", logger, ctx, fmt.Sprintf(fstring, args...))
		}
		logger.Debug(ctx, fstring, args...)
	}
}

func DebugWithContext(ctx context.Context, fstring string, args ...any) {
	LogWithContext(ctx, 4, fstring, args...)
}

func InfoWithContext(ctx context.Context, fstring string, args ...any) {
	LogWithContext(ctx, 3, fstring, args...)
}

func WarnWithContext(ctx context.Context, fstring string, args ...any) {
	LogWithContext(ctx, 2, fstring, args...)
}

func ErrorWithContext(ctx context.Context, fstring string, args ...any) {
	LogWithContext(ctx, 1, fstring, args...)
}

// End the current job in the context.
func EndJobWithContext(ctx context.Context) {
	logger := ctx.Value(loggerKey{})
	if logger == nil {
		stackBuffer := make([]byte, 64*1024)
		numBytes := runtime.Stack(stackBuffer, false)
		stackTrace := string(stackBuffer[:numBytes])
		FDefaultLog("Attempted to end job where no job existed. Context: %v, stack trace: %v", ctx, stackTrace)
	} else {
		logger, ok := logger.(*JobLogger)
		if !ok {
			FDefaultLog("Unable to end job, cannot cast %v from context %v into logger.", logger, ctx)
		}
		logger.End(ctx)
	}
}
