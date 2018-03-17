package log

import (
	"os"
	"log"
	"fmt"
)

const (
	CALLDEPTH = 1

	BLUE  = "0;34"
	RED   = "0;31"
	GREEN = "0;32"
)

var logger *log.Logger

func InitLog() {
	logger = log.New(os.Stdout, "", log.Ldate|log.Lmicroseconds)
}

func Info(msg ...interface{}) {
	logger.Output(CALLDEPTH, color(GREEN, fmt.Sprint(msg)))
}

func Trace(msg ...interface{}) {
	logger.Output(CALLDEPTH, color(BLUE, fmt.Sprint(msg)))
}

func Error(msg ...interface{}) {
	logger.Output(CALLDEPTH, color(RED, fmt.Sprint(msg)))
}

func color(code, msg string) string {
	return fmt.Sprintf("\033[%sm%s\033[m", code, msg)
}
