package tools

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"text/tabwriter"
	"time"
)

type ErrorInfo struct {
	Cluster string
	Schema  string
	Table   string
	Error   string
}

// PrintSliceAsTable 将一个 JSON 类型切片（假设为 []map[string]interface{}）转换为表格并输出。
// 如果 useCSV 为 true，则输出到指定的 CSV 文件路径；否则，在屏幕上以表格形式输出。
func PrintSliceAsTable(data []map[string]interface{}, useCSV bool, csvOutputPath string) error {
	if len(data) == 0 {
		fmt.Println("No data to print.")
		return nil
	}

	// 获取所有唯一的键作为表头，并排序以保持一致性
	headers := make([]string, 0)
	headerSet := make(map[string]struct{})
	for _, row := range data {
		for key := range row {
			if _, exists := headerSet[key]; !exists {
				headerSet[key] = struct{}{}
				headers = append(headers, key)
			}
		}
	}
	sort.Strings(headers) // 排序表头

	if useCSV {
		// 输出到 CSV 文件
		file, err := os.Create(csvOutputPath)
		if err != nil {
			return err
		}
		defer file.Close()

		writer := csv.NewWriter(file)
		defer writer.Flush()

		// 写入表头
		if err := writer.Write(headers); err != nil {
			return err
		}

		// 写入数据行
		for _, row := range data {
			record := make([]string, len(headers))
			for i, header := range headers {
				value, exists := row[header]
				if exists {
					record[i] = valueToString(value)
				} else {
					record[i] = "" // 缺失值为空
				}
			}
			if err := writer.Write(record); err != nil {
				return err
			}
		}

		fmt.Printf("Data written to CSV file: %s\n", csvOutputPath)
		return nil
	}

	// 如果不使用 CSV，在屏幕上以表格形式输出，使用 tabwriter
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', tabwriter.AlignRight|tabwriter.Debug)
	defer writer.Flush()

	// 写入表头
	fmt.Fprint(writer, "|")
	for _, header := range headers {
		fmt.Fprintf(writer, " %s |", header)
	}
	fmt.Fprintln(writer)

	// 写入分隔线
	fmt.Fprint(writer, "|")
	for range headers {
		fmt.Fprint(writer, " --- |")
	}
	fmt.Fprintln(writer)

	// 写入数据行
	for _, row := range data {
		fmt.Fprint(writer, "|")
		for _, header := range headers {
			value, exists := row[header]
			if exists {
				fmt.Fprintf(writer, " %s |", valueToString(value))
			} else {
				fmt.Fprint(writer, "  |")
			}
		}
		fmt.Fprintln(writer)
	}

	return nil
}

// valueToString 将 interface{} 转换为字符串
func valueToString(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case int:
		return fmt.Sprintf("%d", val)
	case float64:
		return fmt.Sprintf("%f", val)
	case bool:
		return fmt.Sprintf("%t", val)
	case nil:
		return ""
	default:
		jsonBytes, _ := json.Marshal(val)
		return string(jsonBytes)
	}
}

func GenJson(errorInfo []ErrorInfo) {
	name := time.Now().Format("20060102") + "-ddlerror.json"
	b, err := json.MarshalIndent(errorInfo, "", "  ")
	if err != nil {
		fmt.Println("生成JSON失败：", err)
	}
	if err := os.WriteFile(name, b, 0644); err != nil {
		fmt.Println("写入文件失败：", err)
	}
}
