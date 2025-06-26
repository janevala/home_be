//go:build release

package llog

func Out(v ...any) {
}

func Err(err error) {
}

func Fatal(v ...any) {
}

func Fatalf(format string, args ...interface{}) {
}
