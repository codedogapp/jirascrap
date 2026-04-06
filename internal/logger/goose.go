package logger

import (
	"fmt"
)

type GooseLoggerAdapter struct{}

func (l GooseLoggerAdapter) Fatalf(format string, v ...any) {
	log := fmt.Sprintf(format, v...)
	Log.Error(log)
}

func (l GooseLoggerAdapter) Printf(format string, v ...any) {
	log := fmt.Sprintf(format, v...)
	Log.Info(log)
}
