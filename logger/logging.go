package logger

import (
	"fmt"
	"fxaz-random-image/config"
	"log"
	"os"
	"time"
)

var Logger *log.Logger

func InitLogger() error {
	// 定义日志文件路径
	logDir := "./log"
	logFile := fmt.Sprintf("%s/%s-%02d-%02d.log", logDir, config.MainConfig.App.Name, time.Now().Month(), time.Now().Day())

	// 确保日志目录存在，如果不存在则创建
	err := os.MkdirAll(logDir, 0755)
	if err != nil {
		log.Fatalf("Error creating log directory: %v", err)
		return err
	}

	// 打开或创建日志文件
	file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatalf("Error opening log file: %v", err)
		return err
	}

	// 创建 logger
	Logger = log.New(file, "", log.LstdFlags)
	return nil
}
