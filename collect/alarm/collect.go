package alarm

import (
	"GoldenDB/log"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

var logger *log.Logger

func SetLogger(l *log.Logger) {
	logger = l
}

type Reserve4 struct {
	DstInfo        string `json:"dstinfo"`
	DstType        string `json:"dstType"`
	DstClusterId   string `json:"dstClusterId"`
	Count          int    `json:"count"`
	DstClusterName string `json:"dstClusterName"`
	DstGroupId     string `json:"dstGroupId"`
	RecoveryFlag   int    `json:"recoveryFlag"`
}

type Alarm struct {
	Alarmid     int
	Alarmsource string
	Code        int
	Almlevel    int
	Content     string
	Createtime  string
	Updatetime  string
	Reserve4    Reserve4
}
type Re struct {
	Cluster string `json:"cluster"`
	App     string `json:"app"`
	Tenant  string `json:"tenant"`
	Host    string `json:"host"`
}
type AlarmInfo struct {
	AlarmTitle   string `json:"alarmTitle"`
	Dn           string `json:"dn"`
	Resource     Re     `json:"resource"`
	EventType    string `json:"eventType"`
	EventId      int    `json:"eventId"`
	CreateTime   string `json:"createTime"`
	Priority     int    `json:"priority"`
	AlarmContent string `json:"alarmContent"`
}

/*
ob的告警如下
SeverityMapping = {
    '停服' : 1,
    '严重' : 2,
    '警告' : 3,
    '注意' : 4,
    '提醒' : 5
}
gdb的告警如下：
1: "紧急告警",
2: "重要告警",
3: "次要告警",
4: "警告告警",
8: "通知",
*/

var AlarmLevelMap = map[int]string{
	1: "紧急告警",
	2: "重要告警",
	3: "次要告警",
	4: "警告告警",
	5: "通知", // 兼容amp平台ob告警接口
	8: "通知",
}

// 实现 sql.Scanner 接口：让Reserve4支持直接从数据库JSON字段Scan
func (r *Reserve4) Scan(value interface{}) error {
	// 1. 处理NULL值（数据库中reserve4字段为NULL时）
	if value == nil {
		*r = Reserve4{} // 赋空结构体，避免nil指针
		return nil
	}

	// 2. 将数据库返回的值转为[]byte（数据库JSON字段返回的是[]uint8，即[]byte）
	jsonBytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("Reserve4 Scan失败：不支持的类型 %T（仅支持JSON字节数据）", value)
	}

	// 3. 空JSON字符串处理
	if len(jsonBytes) == 0 || string(jsonBytes) == "null" {
		*r = Reserve4{}
		return nil
	}

	// 4. JSON反序列化为Reserve4结构体
	var reserve4 Reserve4
	if err := json.Unmarshal(jsonBytes, &reserve4); err != nil {
		return fmt.Errorf("Reserve4 JSON反序列化失败：%w，原始数据：%s", err, string(jsonBytes))
	}

	// 5. 赋值给当前结构体
	*r = reserve4
	return nil
}

// 全局过滤配置
var filterConfig *FilterConfig

// 初始化过滤配置
func init() {
	// 尝试加载过滤配置，如果失败则使用空配置
	config, err := LoadFilterConfig("config/alarm_filter.json")
	if err != nil {
		fmt.Printf("加载过滤配置失败: %v，将继续使用默认配置（不过滤）\n", err)
	} else {
		filterConfig = config
		fmt.Printf("加载过滤配置成功，启用状态: %v, 规则数量: %d\n", filterConfig.Enabled, len(filterConfig.Rules))
	}
}

// LogFilterStatus 记录过滤配置状态
func LogFilterStatus() {
	if logger == nil {
		return
	}
	if filterConfig == nil {
		logger.Warn("告警过滤配置未加载 (filterConfig is nil)")
		return
	}
	logger.Info("告警过滤配置状态: Enabled=%v, 规则数量=%d", filterConfig.Enabled, len(filterConfig.Rules))
	for i, rule := range filterConfig.Rules {
		logger.Info("规则 #%d: %s (Enabled=%v)", i+1, rule.Name, rule.Enabled)
	}
}

// 采集告警
func GetAlarm(mds *sql.DB) []Alarm {
	var AlarmList []Alarm
	if logger != nil {
		logger.Info("采集告警")
	}
	sqlstr := "select alarmid,alarmsource,code,almlevel,content,createtime,updatetime,reserve4 from goldendb_omm.gdb_alarming"
	rows, err := mds.Query(sqlstr)
	if err != nil {
		if logger != nil {
			logger.Error("GetAlarm error: %v", err)
		}
		return nil
	}
	defer rows.Close()

	for rows.Next() {
		var alarms Alarm
		err := rows.Scan(&alarms.Alarmid, &alarms.Alarmsource, &alarms.Code, &alarms.Almlevel, &alarms.Content, &alarms.Createtime, &alarms.Updatetime, &alarms.Reserve4)
		if err != nil {
			if logger != nil {
				logger.Error("Scan error: %v", err)
			}
			continue
		}
		if logger != nil {
			logger.Info("采集到告警: ID=%d, Code=%d, Content=%s", alarms.Alarmid, alarms.Code, alarms.Content)
		}
		AlarmList = append(AlarmList, alarms)
	}

	// 应用过滤
	if filterConfig != nil && filterConfig.Enabled {
		originalCount := len(AlarmList)
		for _, alarm := range AlarmList {
			fmt.Printf("原始告警: %+v\n", alarm)
		}
		AlarmList = FilterAlarms(AlarmList, filterConfig)
		for _, alarm := range AlarmList {
			fmt.Printf("过滤后的告警: %+v\n", alarm)
		}
		filteredCount := originalCount - len(AlarmList)
		if filteredCount > 0 {
			if logger != nil {
				logger.Info("过滤了 %d 条告警，剩余 %d 条告警", filteredCount, len(AlarmList))
			}
		}
	}

	return AlarmList
}

// 封装告警信息
func GenAlarmInfo(alarm Alarm, insight string, eventtype string) AlarmInfo {
	var alarmInfo AlarmInfo
	alarmInfo.AlarmTitle = fmt.Sprint(insight, "-alarm")
	alarmInfo.Dn = insight
	alarmInfo.Resource = Re{
		Cluster: insight,
		App:     alarm.Reserve4.DstType,
		Tenant:  alarm.Reserve4.DstClusterName,
		Host:    alarm.Reserve4.DstInfo,
	}
	alarmInfo.EventType = eventtype
	alarmInfo.EventId = alarm.Alarmid
	alarmInfo.CreateTime = alarm.Createtime
	alarmInfo.Priority = alarm.Almlevel
	if alarm.Almlevel == 8 {
		alarmInfo.Priority = 5
	}
	alarmInfo.AlarmContent = alarm.Content
	return alarmInfo
	/*
		payload = {
		    "alarmTitle": "YZ_OCP-alarm",
		    "dn": "YZ_OCP",
		    "resource":{
		        "cluster": cluster,
		        "app": os.environ.get('app_type'),
		        "tenant": tenant,
		        "host": host
		    },
		    "eventType": isresume,
		    "eventId": os.environ.get('alarm_id'),
		    "createTime": dt,
		    "priority": -,
		    "alarmContent": context
		    }*/
}

func GenAlarmList(alarms []Alarm, insight string, eventtype string) []AlarmInfo {
	var alarmList []AlarmInfo
	for _, alarm := range alarms {
		alarmList = append(alarmList, GenAlarmInfo(alarm, insight, eventtype))
	}
	return alarmList
}

// 将告警实例转化为JSON格式永远推送告警
func ToJSON(alarmInfo AlarmInfo) (string, error) {
	body, err := json.Marshal(alarmInfo) // 或json.MarshalIndent(alarmInfo, "", "    ") 以美化
	if err != nil {
		return "", fmt.Errorf("marshal to JSON error: %w", err)
	}
	return string(body), nil
}

// 发送告警
func SendAlarmToHTTP(alarmInfo AlarmInfo, apiAddress string) error {
	// 将AlarmInfo转换为JSON
	body, err := ToJSON(alarmInfo)
	if logger != nil {
		logger.Info("发送告警: %s", body)
	}
	if err != nil {
		return fmt.Errorf("convert to JSON error: %w", err)
	}

	// 创建HTTP客户端，设置超时
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// 创建POST请求
	req, err := http.NewRequest("POST", apiAddress, strings.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request error: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		// 处理超时或其他错误
		return fmt.Errorf("send alarm error: %w, body: %s", err, body)
	}
	defer resp.Body.Close()

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("send alarm failed, http code %d", resp.StatusCode)
	}

	// 读取并打印响应体
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body error: %w", err)
	}
	if logger != nil {
		logger.Info("响应: %s", string(respBody))
	}

	return nil
}

// deleteAlarm 删除告警函数：设置EventType为"resolve"并发送（使用缓存中的完整信息）
func deleteAlarm(alarm AlarmInfo, address string) error {
	alarm.EventType = "resolve"
	err := SendAlarmToHTTP(alarm, address)
	return err
}
func addAlarm(alarm AlarmInfo, address string) error {
	alarm.EventType = "trigger"
	err := SendAlarmToHTTP(alarm, address)
	return err
}

func ProcessAlarmChanges(currentAlarms []AlarmInfo, cache *sync.Map, alarmURL string) error {
	// 当前告警的map，便于O(1)查找，key为EventId
	currentMap := make(map[int]AlarmInfo)
	for _, a := range currentAlarms {
		currentMap[a.EventId] = a
	}

	// 检测消失的告警（在缓存中但不在当前）
	var deleteErrs []error
	cache.Range(func(key, value interface{}) bool {
		id := key.(int)
		if _, exists := currentMap[id]; !exists {
			alarm := value.(AlarmInfo)
			if err := deleteAlarm(alarm, alarmURL); err != nil {
				deleteErrs = append(deleteErrs, err)
			}
			cache.Delete(id) // 无论成功失败，都从缓存移除
		}
		return true
	})

	// 检测新增的告警（在当前但不在缓存）
	var addErrs []error
	for id, alarm := range currentMap {
		if _, loaded := cache.Load(id); !loaded {
			if err := addAlarm(alarm, alarmURL); err != nil {
				addErrs = append(addErrs, err)
			} else {
				cache.Store(id, alarm) // 成功才添加缓存
			}
		}
	}
	// 汇总错误
	if len(deleteErrs) > 0 || len(addErrs) > 0 {
		return fmt.Errorf("处理变化时有错误: 添加错误(%d), 删除错误(%d)", len(addErrs), len(deleteErrs))
	}
	return nil
}
