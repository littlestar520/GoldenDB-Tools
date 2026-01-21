# 告警过滤配置说明

## 配置文件格式

配置文件位于 `config/alarm_filter.json`，采用JSON格式配置过滤规则。

## 基本结构

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

## 配置参数说明

### 1. 顶层配置
- `enabled`: 是否启用整个过滤功能 (true/false)
- `rules`: 过滤规则数组

### 2. 规则配置
- `name`: 规则名称，便于识别
- `enabled`: 是否启用该规则 (true/false)
- `filters`: 过滤条件数组
- `logic`: 逻辑关系 ("AND" 或 "OR")

### 3. 过滤条件
- `field`: 告警字段名
  - `code`: 告警代码
  - `content`: 告警内容
  - `dstinfo`: 目标信息
- `operator`: 操作符
  - `equals`: 等于（主要用于code字段）
  - `contains`: 包含（主要用于字符串字段）
- `value`: 匹配的值
  - 数字：用于code字段
  - 字符串：用于content和dstinfo字段

## 使用示例

### 示例1：过滤测试告警
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
**效果**：告警内容包含"测试"或"test"的告警都会被过滤

### 示例2：过滤特定代码告警
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
**效果**：告警代码为1001或1002的告警都会被过滤

### 示例3：组合条件过滤
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
**效果**：只有目标信息包含"backup"**且**内容包含"维护"的告警才会被过滤

### 示例4：多规则组合
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
**效果**：告警内容包含"测试"**或**告警代码为1001的告警都会被过滤

## 逻辑关系说明

### OR 逻辑
- 多个条件中只要有一个匹配，整个规则就匹配
- 适用于"过滤其中任意一种情况"

### AND 逻辑
- 所有条件都必须匹配，整个规则才匹配
- 适用于"只有同时满足多个条件才过滤"

## 字段说明

### 1. code字段
- 类型：整数
- 用途：匹配具体的告警代码
- 示例：`{"field": "code", "operator": "equals", "value": 1001}`

### 2. content字段
- 类型：字符串
- 用途：匹配告警内容中的关键词
- 示例：`{"field": "content", "operator": "contains", "value": "连接失败"}`

### 3. dstinfo字段
- 类型：字符串
- 用途：匹配目标主机或设备信息
- 示例：`{"field": "dstinfo", "operator": "contains", "value": "db-server"}`

## 高级用法

### 动态修改配置
程序启动时会自动加载配置文件，修改配置文件后需要重启服务生效。

### 临时禁用
可以设置 `"enabled": false` 来临时禁用所有过滤规则。

### 规则级别的启用/禁用
可以单独设置每个规则的 `"enabled": false` 来禁用特定规则。

## 常见问题

### 1. 配置文件格式错误
确保JSON格式正确，可以使用在线JSON验证工具检查。

### 2. 字段名拼写错误
确保使用正确的字段名：`code`, `content`, `dstinfo`

### 3. 操作符使用错误
- `equals` 只用于数字字段（code）
- `contains` 只用于字符串字段（content, dstinfo）

### 4. 逻辑关系理解错误
- OR：任意条件匹配就过滤
- AND：所有条件都匹配才过滤

## 验证配置

启动程序时会显示配置加载状态：
```
加载过滤配置成功，启用状态: true, 规则数量: 3
```

处理告警时会显示过滤效果：
```
过滤了 2 条告警，剩余 5 条告警
```