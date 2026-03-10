package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

var (
	// 定义不同级别的日志文件
	InfoLogger    *log.Logger
	WarningLogger *log.Logger
	ErrorLogger   *log.Logger
)

// SetupLogger 初始化日志配置
func SetupLogger() error {
	// 创建日志目录
	logDir := "logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("创建日志目录失败: %v", err)
	}

	// 生成当前日期的日志文件名
	currentTime := time.Now()
	logFileName := filepath.Join(logDir, fmt.Sprintf("%s.log", currentTime.Format("2006-01-02")))

	// 打开日志文件
	logFile, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("打开日志文件失败: %v", err)
	}

	// 设置多重输出：同时输出到控制台和文件
	multiWriter := io.MultiWriter(os.Stdout, logFile)

	// 初始化不同级别的日志记录器
	InfoLogger = log.New(multiWriter, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	WarningLogger = log.New(multiWriter, "WARNING: ", log.Ldate|log.Ltime|log.Lshortfile)
	ErrorLogger = log.New(multiWriter, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)

	return nil
}

// Info 记录信息级别的日志
func Info(format string, v ...interface{}) {
	InfoLogger.Printf(format, v...)
}

// Warning 记录警告级别的日志
func Warning(format string, v ...interface{}) {
	WarningLogger.Printf(format, v...)
}

// Error 记录错误级别的日志
func Error(format string, v ...interface{}) {
	ErrorLogger.Printf(format, v...)
}
