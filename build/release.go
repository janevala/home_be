//go:build release

// build/release.go
package build

import "log"

func IsProduction() bool {
	return true
}

func LogOut(v ...any) {
}

func LogErr(err error) {
	if err == nil {
		return
	}

	log.Printf("[ERROR] %s", err.Error())
}

func LogFatal(v ...any) {
	if len(v) == 0 {
		return
	}

	log.Fatalf("[FATAL] %v", v...)
}

func LogFatalf(format string, args ...interface{}) {
	if format == "" {
		return
	}

	log.Fatalf("[FATAL] "+format, args...)
}
