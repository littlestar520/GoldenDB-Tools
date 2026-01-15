package config

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
)

type Config struct {
	Alarm struct {
		ApiAddress string `yaml:"api_address"`
		Time       int    `yaml:"time"`
	} `yaml:"alarm"`
	Log struct {
		Path          string `yaml:"path"`
		Level         string `yaml:"level"`
		KeepDays      int    `yaml:"keep_days"`
		CleanInterval int    `yaml:"clean_interval"`
	} `yaml:"log"`
}

// ReadAlarmConfig 函数读取YAML文件，返回api_address和time值
func ReadAlarmConfig(filename string) (string, int, error) {
	// 读取文件内容
	data, err := os.ReadFile(filename)
	if err != nil {
		return "", 0, fmt.Errorf("读取文件失败: %w", err)
	}

	// 解析YAML
	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return "", 0, fmt.Errorf("解析YAML失败: %w", err)
	}

	// 返回提取的值
	return config.Alarm.ApiAddress, config.Alarm.Time, nil
}

// ReadFullConfig 读取完整配置
func ReadFullConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("读取文件失败: %w", err)
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("解析YAML失败: %w", err)
	}

	// 设置默认值
	if config.Log.Path == "" {
		config.Log.Path = "log/app.log"
	}
	if config.Log.Level == "" {
		config.Log.Level = "INFO"
	}
	if config.Log.KeepDays == 0 {
		config.Log.KeepDays = 7
	}
	if config.Log.CleanInterval == 0 {
		config.Log.CleanInterval = 86400
	}

	return &config, nil
}
