package main

import (
	"GoldenDB/alarm"
	"GoldenDB/check"
	"GoldenDB/connect"
	"database/sql"
	"fmt"
	"os"
)

func main() {
	args := os.Args
	if len(args) < 2 {
		fmt.Println("用法: -s | -p <明文密码>")
		return
	}
	if args[1] == "-s" {
		startCheck()
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
func startCheck() {
	// 存放所有DDL失败的表信息
	var ErrorList []check.ErrorInfo
	// MDS连接信息
	mdsList := connect.GetMDS()
	// 连接所有的MDS
	for _, mds := range mdsList {
		dbConnect := connect.GetDBConnect(mds.DSN)
		defer dbConnect.Close()

		if err := dbConnect.Ping(); err != nil {
			fmt.Println("MDS连接失败")
		}

		// 获取所有集群ID
		IDList := alarm.GetClusterID(dbConnect)

		for _, id := range IDList {
			//通过集群ID找到集群名
			clusterName := alarm.GetClusterName(dbConnect, id)
			// 连接集群下的所有DN节点
			schemas := alarm.GetSchema(dbConnect, id)
			dninfo := alarm.GetDNinfo(dbConnect, id, mds.Username, mds.Password)

			// 所有DN连接
			var dnlist []*sql.DB
			for _, dndsn := range dninfo {
				dn := connect.GetDBConnect(dndsn)
				defer dn.Close()
				dnlist = append(dnlist, dn)
			}
			// 获取所有分区表信息
			tables := check.GetTables(dnlist[0], schemas)

			// 检查所有分区表的分区信息是否一致
			for _, table := range tables {
				status := check.CheckTablePartitionsInfo(dnlist, table.Name, table.Schema)
				if !status {
					fmt.Println("分区表", table, "分区信息不一致")
					ErrorList = append(ErrorList, check.ErrorInfo{
						Cluster: clusterName,
						Schema:  table.Schema,
						Table:   table.Name,
						Error:   fmt.Sprintf("租户%s的%s库的%s分区信息不一致", clusterName, table.Schema, table.Name),
					})
					continue
				}
				fmt.Println("分区表", table, "分区信息一致")
			}

		}

	}
	check.GenJson(ErrorList)
}
