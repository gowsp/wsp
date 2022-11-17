package logger

import (
	"fmt"
	"log"
	"os"
)

const (
	error = iota
	info
	debug
	trace
)

var level int = info
var logger *log.Logger = log.New(os.Stdout, "", log.LstdFlags)

func Fatalln(v ...any) {
	logger.Output(3, fmt.Sprintln(v...))
	os.Exit(1)
}
func Error(format string, v ...any) {
	logger.Output(3, fmt.Sprintf("[ERROR] "+format+"\n", v...))
}
func Info(format string, v ...any) {
	if level > error {
		logger.Output(3, fmt.Sprintf("[INFO] "+format+"\n", v...))
	}
}
func Debug(format string, v ...any) {
	if level > info {
		logger.Output(3, fmt.Sprintf("[DEBUG] "+format+"\n", v...))
	}
}
func Trace(format string, v ...any) {
	if level > debug {
		logger.Output(3, fmt.Sprintf("[TRACE] "+format+"\n", v...))
	}
}
