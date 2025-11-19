package utils

import (
	"context"
	"time"
)

func TimeoutContextWithCancel(timeoutSeconds int) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), time.Duration(timeoutSeconds) * time.Second)
}

func TimeoutContext(timeoutSeconds int) (context.Context, context.CancelFunc){
	return context.WithTimeout(context.Background(), time.Duration(timeoutSeconds) * time.Second)
}
