//go:build release

// build/release.go
package build

import "log"

var logger *log.Logger

func SetLogger(l *log.Logger) {
	logger = l
}

func IsProduction() bool {
	return true
}

func LogOut(v ...any) {
}

func LogErr(err error) {
	if err == nil || logger == nil {
		return
	}

	logger.Printf("[ERROR] %s", err.Error())
}

func LogFatal(v ...any) {
	if len(v) == 0 || logger == nil {
		return
	}

	logger.Fatalf("[FATAL] %v", v...)
}

func LogFatalf(format string, args ...interface{}) {
	if format == "" || logger == nil {
		return
	}

	logger.Fatalf("[FATAL] "+format, args...)
}
