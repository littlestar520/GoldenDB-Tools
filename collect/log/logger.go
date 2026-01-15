package log

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Logger 日志结构体
type Logger struct {
	file     *os.File
	filePath string
	module   string
}

// NewLogger 创建日志实例
func NewLogger(module string, filePath string) (*Logger, error) {
	// 确保目录存在
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("创建日志目录失败: %w", err)
	}

	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("打开日志文件失败: %w", err)
	}

	return &Logger{
		file:     file,
		filePath: filePath,
		module:   module,
	}, nil
}

// log 写入日志
func (l *Logger) log(level string, format string, args ...interface{}) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	message := fmt.Sprintf(format, args...)
	logLine := fmt.Sprintf("[%s] [%s] [%s] %s\n", timestamp, level, l.module, message)
	l.file.WriteString(logLine)
	l.file.Sync()
}

// InfoInfo
func (l *Logger) Info(format string, args ...interface{}) {
	l.log("INFO", format, args...)
}

// Error 错误日志
func (l *Logger) Error(format string, args ...interface{}) {
	l.log("ERROR", format, args...)
}

// Warn 警告日志
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log("WARN", format, args...)
}

// Debug 调试日志
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log("DEBUG", format, args...)
}

// Close 关闭日志文件
func (l *Logger) Close() error {
	return l.file.Close()
}

// CleanOldLogs 清理指定天数之前的日志文件
func CleanOldLogs(logDir string, days int) error {
	dir, err := os.Open(logDir)
	if err != nil {
		return fmt.Errorf("打开日志目录失败: %w", err)
	}
	defer dir.Close()

	files, err := dir.Readdirnames(-1)
	if err != nil {
		return fmt.Errorf("读取日志目录失败: %w", err)
	}

	cutoff := time.Now().AddDate(0, 0, -days)
	deletedCount := 0

	for _, filename := range files {
		// 只处理 .log 文件
		if !strings.HasSuffix(filename, ".log") {
			continue
		}

		filePath := filepath.Join(logDir, filename)
		info, err := os.Stat(filePath)
		if err != nil {
			continue
		}

		// 检查文件修改时间
		if info.ModTime().Before(cutoff) {
			if err := os.Remove(filePath); err != nil {
				continue
			}
			deletedCount++
		}
	}

	if deletedCount > 0 {
		fmt.Printf("已清理 %d 个过期日志文件\n", deletedCount)
	}

	return nil
}

// GetLogFileSize 获取日志文件大小（字节）
func GetLogFileSize(filePath string) (int64, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}
