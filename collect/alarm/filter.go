package alarm

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// FilterConfig 过滤配置
type FilterConfig struct {
	Enabled bool     `json:"enabled"`
	Rules   []Filter `json:"rules"`
}

// Filter 单个过滤规则
type Filter struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
	Filters []Rule `json:"filters"`
	Logic   string `json:"logic"` // "AND" 或 "OR"
}

// Rule 单个过滤条件
type Rule struct {
	Field    string      `json:"field"`    // "code", "content", "dstinfo"
	Operator string      `json:"operator"` // "equals", "contains"
	Value    interface{} `json:"value"`    // 字符串或数字
}

// LoadFilterConfig 加载过滤配置
func LoadFilterConfig(filename string) (*FilterConfig, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("读取过滤配置文件失败: %w", err)
	}

	var config FilterConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析过滤配置失败: %w", err)
	}

	return &config, nil
}

// FilterAlarms 过滤告警列表
func FilterAlarms(alarms []Alarm, config *FilterConfig) []Alarm {
	if config == nil || !config.Enabled {
		return alarms
	}

	var result []Alarm
	for _, alarm := range alarms {
		if !shouldFilter(alarm, config) {
			result = append(result, alarm)
		}
	}
	return result
}

// shouldFilter 检查告警是否应该被过滤
func shouldFilter(alarm Alarm, config *FilterConfig) bool {
	for _, rule := range config.Rules {
		if !rule.Enabled {
			continue
		}

		if matchesRule(alarm, rule) {
			return true
		}
	}
	return false
}

// matchesRule 检查告警是否匹配规则
func matchesRule(alarm Alarm, rule Filter) bool {
	var matches []bool

	for _, condition := range rule.Filters {
		matches = append(matches, matchesCondition(alarm, condition))
	}

	if rule.Logic == "AND" {
		for _, match := range matches {
			if !match {
				return false
			}
		}
		return len(matches) > 0
	} else { // OR
		for _, match := range matches {
			if match {
				return true
			}
		}
		return false
	}
}

// matchesCondition 检查单个条件
func matchesCondition(alarm Alarm, condition Rule) bool {
	switch condition.Field {
	case "code":
		return compareCode(alarm.Code, condition)
	case "content":
		return compareString(alarm.Content, condition)
	case "dstinfo":
		return compareString(alarm.Reserve4.DstInfo, condition)
	default:
		return false
	}
}

// compareCode 比较告警代码
func compareCode(code int, condition Rule) bool {
	if condition.Operator != "equals" {
		return false
	}

	var condInt int
	switch v := condition.Value.(type) {
	case float64:
		condInt = int(v)
	case string:
		parsed, err := strconv.Atoi(v)
		if err != nil {
			return false
		}
		condInt = parsed
	default:
		return false
	}

	return code == condInt
}

// compareString 比较字符串
func compareString(str string, condition Rule) bool {
	if condition.Operator != "contains" {
		return false
	}

	var condStr string
	switch v := condition.Value.(type) {
	case string:
		condStr = v
	case float64:
		condStr = strconv.Itoa(int(v))
	default:
		return false
	}

	return strings.Contains(strings.ToLower(str), strings.ToLower(condStr))
}

// FilterAlarmsCustom 自定义过滤函数，可直接调用
func FilterAlarmsCustom(alarms []Alarm, filters []Rule, logic string) []Alarm {
	var result []Alarm
	for _, alarm := range alarms {
		if !matchesCustomRule(alarm, filters, logic) {
			result = append(result, alarm)
		}
	}
	return result
}

// matchesCustomRule 检查自定义规则
func matchesCustomRule(alarm Alarm, filters []Rule, logic string) bool {
	var matches []bool
	for _, condition := range filters {
		matches = append(matches, matchesCondition(alarm, condition))
	}

	if logic == "AND" {
		for _, match := range matches {
			if !match {
				return false
			}
		}
		return len(matches) > 0
	} else { // OR
		for _, match := range matches {
			if match {
				return true
			}
		}
		return false
	}
}
