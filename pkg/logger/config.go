package logger

import (
	"io"
	"log"
	"os"
	"strings"
)

type Config struct {
	Level  string `json:"level,omitempty"`
	Output string `json:"output,omitempty"`
}

func (c *Config) Init() {
	flag := log.LstdFlags
	switch strings.ToUpper(c.Level) {
	case "ERROR":
		level = error
	case "DEBUG":
		flag |= log.Lshortfile
		level = debug
	case "TRACE":
		flag |= log.Lshortfile
		level = trace
	}
	out := output(c.Output)
	logger = log.New(out, "", flag)
}

func output(path string) io.Writer {
	if path == "" {
		return os.Stdout
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		logger.Fatalln(err)
	}
	return file
}
