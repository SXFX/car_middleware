package log

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

var logger log.Logger

func InitLogger(logFilePrefix string) {
	now := time.Now()
	nowStr := fmt.Sprintf("%4d%02d%02d%02d%02d%02d",
		now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second())

	logFileName := logFilePrefix + "_" + nowStr + ".log"
	logFileAbs, err := filepath.Abs(logFileName)
	if err != nil {
		log.Fatalln(err)
	}

	logFile, err := os.Create(logFileAbs)
	if err != nil {
		log.Fatalln(err)
	}

	logger.SetOutput(logFile)
	logger.SetFlags(log.LstdFlags | log.Llongfile)
}

func Debug(v ...interface{}) {
	logger.SetPrefix("[debug]")
	logger.Println(v)
}

func Info(v ...interface{}) {
	logger.SetPrefix("[info]")
	logger.Println(v)
}

func Error(v ...interface{}) {
	logger.SetPrefix("[error]")
	logger.Println(v)
}

func Fatal(v ...interface{}) {
	logger.SetPrefix("[fatal]")
	logger.Fatalln(v)
}
