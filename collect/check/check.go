package check

import (
	"database/sql"
	"fmt"
)

type TableInfo struct {
	Schema string
	Name   string
}

func GetTables(dn *sql.DB, schema []string) []TableInfo {
	tmsql := "select table_name from information_schema.tables where table_schema=? and CREATE_OPTIONS like '%partitioned%';"
	var tables []TableInfo
	for _, s := range schema {
		rows, err := dn.Query(tmsql, s)
		if err != nil {
			fmt.Println("查询分区表失败：", err)
			continue
		}
		defer rows.Close()
		for rows.Next() {
			var table string
			err := rows.Scan(&table)
			if err != nil {
				fmt.Println("解析分区表失败：", err)
				continue
			}
			tables = append(tables, TableInfo{Schema: s, Name: table})
		}
		if err := rows.Err(); err != nil {
			fmt.Println("查询分区表失败：", err)
		}
	}
	return tables
}

func CheckTablePartitionsInfo(dnlist []*sql.DB, table string, schema string) bool {
	fmt.Println("正在检查分区表：", table)
	if len(dnlist) == 0 {
		return false
	}
	var base []string
	for i, dn := range dnlist {
		parts := GetTablePartitionsInfo(dn, table, schema)
		if len(parts) == 0 {
			return false
		}
		if i == 0 {
			base = parts
			continue
		}
		if !equalOrder(base, parts) {
			return false
		}
	}
	return true
}

func equalOrder(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func GetTablePartitionsInfo(dn *sql.DB, table string, schema string) []string {
	sql := "SELECT DISTINCT(PARTITION_NAME) FROM INFORMATION_SCHEMA.PARTITIONS WHERE  TABLE_NAME = ? AND TABLE_SCHEMA = ? AND PARTITION_NAME IS NOT NULL ORDER BY PARTITION_NAME "

	rows, err := dn.Query(sql, table, schema)
	if err != nil {
		fmt.Println("查询分区信息失败：", err)
		return []string{}
	}
	defer rows.Close()

	var partitions []string

	for rows.Next() {
		var par string
		err := rows.Scan(&par)
		if err != nil {
			fmt.Println("解析分区信息失败：", err)
			return []string{}
		}
		partitions = append(partitions, par)
	}
	return partitions
}
