package initialize

import (
	"fmt"
	"github.com/FXAZfung/random-image/internal/config"
	"github.com/FXAZfung/random-image/internal/logger"
	"log"
	"os"
	"time"
)

var currentLogFile string

func InitLogger() error {
	return rotateLogFile()
}

func rotateLogFile() error {
	logDir := "./log"
	logFile := fmt.Sprintf("%s/%s-%02d-%02d.log", logDir, config.MainConfig.App.Name, time.Now().Month(), time.Now().Day())

	if logFile == currentLogFile {
		return nil
	}

	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("error creating log directory: %w", err)
	}

	file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return fmt.Errorf("error opening log file: %w", err)
	}

	logger.Logger = log.New(file, "", log.LstdFlags)
	currentLogFile = logFile
	return nil
}
