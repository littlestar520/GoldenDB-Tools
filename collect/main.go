package main

import (
	"GoldenDB/alarm"
	"GoldenDB/config"
	"GoldenDB/connect"
	"GoldenDB/log"
	"fmt"
	"os"
	"sync"
	"time"
)

func main() {
	args := os.Args
	if len(args) < 2 {
		fmt.Println("用法: -s | -p <明文密码>")
		return
	}
	if args[1] == "-s" {
		Start()
		return
	}
	if args[1] == "-p" {
		if len(args) < 3 {
			fmt.Println("请输入明文密码: -p <text>")
			return
		}
		encrypt, err := connect.Encrypt(args[2])
		if err != nil {
			fmt.Println("加密失败")
			return
		}
		fmt.Println(encrypt)
		return
	}
	fmt.Println("参数错误: -s | -p <明文密码>")
	return
}

// 全局日志实例
var logger *log.Logger

func initLogger() {
	// 读取完整配置
	cfg, err := config.ReadFullConfig("config/amp_api.yaml")
	if err != nil {
		fmt.Printf("读取日志配置失败: %v，使用默认配置\n", err)
		cfg = &config.Config{}
		cfg.Log.Path = "log/app.log"
		cfg.Log.KeepDays = 7
		cfg.Log.CleanInterval = 86400
	}

	var errCreate error
	logger, errCreate = log.NewLogger("main", cfg.Log.Path)
	if errCreate != nil {
		fmt.Printf("初始化日志失败: %v\n", errCreate)
		logger = nil
		return
	}

	logger.Info("========== 程序启动 ==========")
	logger.Info("日志文件: %s", cfg.Log.Path)
	logger.Info("日志保留天数: %d", cfg.Log.KeepDays)

	// 启动日志清理定时器
	go func() {
		cleanTicker := time.NewTicker(time.Duration(cfg.Log.CleanInterval) * time.Second)
		defer cleanTicker.Stop()

		for range cleanTicker.C {
			logDir := "log"
			if err := log.CleanOldLogs(logDir, cfg.Log.KeepDays); err != nil {
				if logger != nil {
					logger.Error("清理日志失败: %v", err)
				}
			}
		}
	}()

	// 立即执行一次日志清理
	if err := log.CleanOldLogs("log", cfg.Log.KeepDays); err != nil {
		logger.Error("首次清理日志失败: %v", err)
	}
}

func Start() {
	// 初始化日志
	initLogger()
	if logger != nil {
		logger.Info("开始启动监控服务")
	}

	// 设置alarm包的logger
	alarm.SetLogger(logger)

	// 获取MDS列表
	mdsList := connect.GetMDS()
	if logger != nil {
		logger.Info("获取到 %d 个MDS节点", len(mdsList))
	}

	var wg sync.WaitGroup
	// 读取配置文件
	api_address, timePeriod, _ := config.ReadAlarmConfig("config/amp_api.yaml")
	if logger != nil {
		logger.Info("API地址: %s, 监控周期: %d秒", api_address, timePeriod)
	}

	// 连接所有的MDS
	for _, mds := range mdsList {
		currentMDS := mds
		wg.Add(1)
		// 并发处理
		go func() {
			defer wg.Done()
			if logger != nil {
				logger.Info("正在连接 MDS: %s", mds.Name)
			}

			dbConnect := connect.GetDBConnect(currentMDS.DSN)
			defer dbConnect.Close()
			if err := dbConnect.Ping(); err != nil {
				if logger != nil {
					logger.Error("MDS连接失败: %s, 错误: %v", mds.Name, err)
				}
				return
			}

			if logger != nil {
				logger.Info("MDS连接成功: %s", mds.Name)
			}

			insight := mds.Name //insight平台名称
			var cache sync.Map  // 定义缓存

			ticker := time.NewTicker(time.Duration(timePeriod) * time.Second) //定时器
			defer ticker.Stop()

			// 定时查询并告警
			for {
				select {
				case <-ticker.C:
					// 查询当前所有告警并封装成切片
					Alarms := alarm.GetAlarm(dbConnect)
					currentAlarms := alarm.GenAlarmList(Alarms, insight, "trigger")

					// 处理告警
					if err := alarm.ProcessAlarmChanges(currentAlarms, &cache, api_address); err != nil {
						if logger != nil {
							logger.Error("处理告警失败: %v", err)
							return
						}
					} else {
						if logger != nil {
							logger.Info("处理告警完成, 当前告警数: %d", len(currentAlarms))
							for _, v := range currentAlarms {
								logger.Info("当前告警: %+v", v)
							}
						}
					}
				}
			}

		}()
	}
	wg.Wait()
}
