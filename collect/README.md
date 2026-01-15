b# GoldenDB Alarm Monitor

GoldenDB告警监控服务，用于监控GoldenDB数据库集群的告警信息，并将告警推送到告警平台。

## 功能特性

- **多MDS监控**：支持并发监控多个MDS节点
- **告警推送**：自动将告警信息推送到指定的告警接收API
- **变更检测**：智能检测告警的新增和消失
- **日志功能**：支持日志记录和自动清理
- **告警过滤**：支持通过配置文件过滤不需要的告警
- **后台运行**：支持后台守护进程模式运行

## 目录结构

```
.
├── main.go              # 主程序入口
├── manager.sh           # 服务管理脚本
├── config/
│   ├── amp_api.yaml     # 主配置文件
│   ├── alarm_filter.json # 告警过滤配置
│   └── alarm_filter_examples.json # 告警过滤配置示例
├── alarm/
│   ├── collect.go       # 告警采集和处理
│   └── filter.go        # 告警过滤功能
├── connect/
│   └── connect.go       # 数据库连接
├── log/
│   └── logger.go        # 日志功能
├── GdbAlarm-darwin-arm64  # macOS ARM64 编译版本
├── GdbAlarm-linux-amd64   # Linux AMD64 编译版本
└── GdbAlarm-linux-arm     # Linux ARM 编译版本
```

## 快速开始

### 1. 安装依赖

```bash
# 确保已安装Go语言环境 (Go 1.24+)
go version
```

### 2. 编译程序

```bash
# 编译所有平台的版本
./build.sh



### 3. 配置说明

编辑 `config/amp_api.yaml` 文件：

```yaml
alarm:
  # 告警接收API地址
  api_address: "http://your-api-server:8090/js/api/ocpAlarm"
  # 告警推送间隔时间（秒）
  time: 20

# 日志配置
log:
  # 日志文件路径
  path: "log/app.log"
  # 日志等级: DEBUG, INFO, WARN, ERROR
  level: "INFO"
  # 日志清理：保留天数
  keep_days: 7
  # 日志清理检查间隔（秒）
  clean_interval: 86400
```

### 4. 运行服务

#### 前台运行
```bash
./GdbAlarm-darwin-arm64 -s
```

#### 后台运行
```bash
./manager.sh start
```

#### 查看状态
```bash
./manager.sh status
```

#### 停止服务
```bash
./manager.sh stop
```

#### 重启服务
```bash
./manager.sh restart
```

## 管理脚本使用

```bash
# 查看帮助
./manager.sh help

# 启动服务
./manager.sh start

# 停止服务
./manager.sh stop

# 重启服务
./manager.sh restart

# 查看状态
./manager.sh status
```

## 告警过滤配置

### 配置文件位置

编辑 `config/alarm_filter.json` 文件来配置告警过滤规则。

### 基础配置格式

```json
{
  "enabled": true,
  "rules": [
    {
      "name": "规则名称",
      "enabled": true,
      "filters": [
        {
          "field": "字段名",
          "operator": "操作符",
          "value": "值"
        }
      ],
      "logic": "AND"
    }
  ]
}
```

### 支持的字段

| 字段 | 说明 | 操作符 |
|------|------|--------|
| `code` | 告警代码 | `equals` |
| `content` | 告警内容 | `contains` |
| `dstinfo` | 目标信息 | `contains` |

### 配置示例

#### 示例1：过滤测试告警
```json
{
  "enabled": true,
  "rules": [
    {
      "name": "过滤测试告警",
      "enabled": true,
      "filters": [
        {
          "field": "content",
          "operator": "contains",
          "value": "测试"
        },
        {
          "field": "content",
          "operator": "contains",
          "value": "test"
        }
      ],
      "logic": "OR"
    }
  ]
}
```

#### 示例2：过滤特定告警代码
```json
{
  "enabled": true,
  "rules": [
    {
      "name": "过滤维护类告警",
      "enabled": true,
      "filters": [
        {
          "field": "code",
          "operator": "equals",
          "value": 1001
        },
        {
          "field": "code",
          "operator": "equals",
          "value": 1002
        }
      ],
      "logic": "OR"
    }
  ]
}
```

#### 示例3：组合条件过滤
```json
{
  "enabled": true,
  "rules": [
    {
      "name": "过滤特定主机的维护告警",
      "enabled": true,
      "filters": [
        {
          "field": "dstinfo",
          "operator": "contains",
          "value": "backup"
        },
        {
          "field": "content",
          "operator": "contains",
          "value": "维护"
        }
      ],
      "logic": "AND"
    }
  ]
}
```

### 逻辑关系说明

- **OR**：任意条件匹配就过滤
- **AND**：所有条件都必须匹配才过滤

### 使用多个规则

配置文件中的 `rules` 数组可以包含多个规则，规则之间是 OR 关系：

```json
{
  "enabled": true,
  "rules": [
    {
      "name": "过滤测试告警",
      "enabled": true,
      "filters": [
        {
          "field": "content",
          "operator": "contains",
          "value": "测试"
        }
      ],
      "logic": "OR"
    },
    {
      "name": "过滤特定代码",
      "enabled": true,
      "filters": [
        {
          "field": "code",
          "operator": "equals",
          "value": 1001
        }
      ],
      "logic": "OR"
    }
  ]
}
```

修改过滤配置后需要重启服务生效。

## 日志管理

### 日志文件位置

默认日志文件：`log/app.log`

### 日志配置

在 `config/amp_api.yaml` 中配置：

```yaml
log:
  path: "log/app.log"           # 日志文件路径
  level: "INFO"                 # 日志等级
  keep_days: 7                  # 日志保留天数
  clean_interval: 86400         # 清理间隔（秒）
```

### 日志等级

- `DEBUG`：调试信息
- `INFO`：一般信息
- `WARN`：警告信息
- `ERROR`：错误信息

### 自动清理

程序会按照 `clean_interval` 配置的间隔自动清理超过 `keep_days` 天的日志文件。

## 程序参数

### 命令行参数

```bash
# 前台运行
./GdbAlarm -s

# 加密密码
./GdbAlarm -p "明文密码"

# 显示帮助
./GdbAlarm -h
```

## 系统要求

- Go 1.24+
- 支持的平台：
  - macOS ARM64
  - Linux AMD64
  - Linux ARM

## 常见问题

### 1. 程序无法启动
- 检查配置文件格式是否正确
- 检查API地址是否可达
- 查看日志文件获取详细错误信息

### 2. 告警没有推送
- 检查API地址配置是否正确
- 检查网络连接
- 查看日志中的错误信息

### 3. 过滤规则不生效
- 确认 `enabled` 字段设置为 `true`
- 检查JSON格式是否正确
- 重启服务使配置生效

### 4. 日志文件过大
- 调整 `keep_days` 参数减少保留天数
- 调整 `clean_interval` 参数增加清理频率
- 使用日志轮转工具

## 贡献指南

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交改动 (`git commit -m 'Add some AmazingFeature'`)
4. 推送分支 (`git push origin feature/AmazingFeature`)
5. 创建一个 Pull Request

## 许可证

本项目采用 MIT 许可证。