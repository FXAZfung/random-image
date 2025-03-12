package initialize

import (
	"fmt"
	"github.com/FXAZfung/random-image/internal/config"
	"github.com/FXAZfung/random-image/internal/logger"
	"log"
	"os"
)

func InitLogger() error {
	logDir := "./log"
	logFile := fmt.Sprintf("%s/%s.log", logDir, config.MainConfig.App.Name)

	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("error creating log directory: %w", err)
	}

	file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return fmt.Errorf("error opening log file: %w", err)
	}

	logger.Logger = log.New(file, "", log.LstdFlags)
	return nil
}
