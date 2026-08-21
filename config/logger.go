package config

import (
	"io"
	"log"
	"os"
	"strings"
)

var (
	TransactionLogger *log.Logger
	InfoLogger        *log.Logger
	WarnLogger        *log.Logger
	ErrorLogger       *log.Logger
	LogFile           *os.File
)

func InitLogger(cfg *EnvConfig) {
	var err error
	LogFile, err = os.OpenFile(cfg.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}

	multi := io.MultiWriter(os.Stdout, LogFile)

	flags := log.Ldate | log.Ltime | log.Lshortfile

	TransactionLogger = log.New(multi, "[TX] ", flags)
	InfoLogger = log.New(multi, "[INFO] ", flags)
	WarnLogger = log.New(multi, "[WARN] ", flags)
	ErrorLogger = log.New(multi, "[ERROR] ", flags)
}

func LogInfo(format string, v ...interface{}) {
	if InfoLogger != nil {
		InfoLogger.Printf(format, v...)
	}
}

func LogWarn(format string, v ...interface{}) {
	if WarnLogger != nil {
		WarnLogger.Printf(format, v...)
	}
}

func LogError(format string, v ...interface{}) {
	if ErrorLogger != nil {
		ErrorLogger.Printf(format, v...)
	}
}

func LogTx(format string, v ...interface{}) {
	if TransactionLogger != nil {
		TransactionLogger.Printf(format, v...)
	}
}

func ShortPath(path string) string {
	return strings.TrimPrefix(path, "/api")
}
