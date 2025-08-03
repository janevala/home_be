//go:build release

// llog/release.go
package llog

import "log"

func Out(v ...any) {
}

func Err(err error) {
	if err == nil {
		return
	}

	log.Printf("[ERROR] %s", err.Error())
}

func Fatal(v ...any) {
	if len(v) == 0 {
		return
	}

	log.Fatalf("[FATAL] %v", v...)
}

func Fatalf(format string, args ...interface{}) {
	if format == "" {
		return
	}

	log.Fatalf("[FATAL] "+format, args...)
}
