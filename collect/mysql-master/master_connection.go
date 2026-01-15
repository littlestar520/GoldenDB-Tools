// Go MySQL Driver - A MySQL-Driver for Go's database/sql package

package mysql

import (
	"database/sql"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
)

func GetMaster(cfg *Config) (string, error) {
	var masterIP string
	var temp string

	for _, Addr := range cfg.MasterAddrs {
		dsn := cfg.User + ":" + cfg.Passwd + "@" + cfg.Net + "(" + Addr + ")/" + cfg.DBName
		db, err := sql.Open("mysql", dsn)
		if err != nil {
			continue
		}
		defer db.Close()
		err = db.QueryRow("show variables like 'bind_address';").Scan(&temp, &masterIP)
		if err != nil {
			continue
		}
		// 获取完整的ip:port
		for _, addr := range cfg.MasterAddrs {
			parts := strings.Split(addr, ":")
			host := parts[0]
			ips, _ := net.LookupIP(host)
			if strings.Contains(ips[0].String(), masterIP) {
				masterIP = addr
				break
			}
		}
		if masterIP != "" {
			break
		}
	}

	if masterIP != "" {
		return masterIP, nil
	}

	return masterIP, errors.New("no valid address")
}

func Priority(cfg *Config, masterIP string) []string {
	var (
		priority []string
		key      string
		value    string
		Addrs    []string
	)
	mparts := strings.Split(masterIP, ":")
	mhost := mparts[0]
	mips, _ := net.LookupIP(mhost)
	masterIpGroupExist := false
	// 主CN所在group优先级最高
	for key, value = range cfg.ProxyGroup {
		Addrs = nil
		Addrs = append(Addrs, strings.Split(value, ",")...)
		for _, addr := range Addrs {
			parts := strings.Split(addr, ":")
			host := parts[0]
			ips, _ := net.LookupIP(host)
			if strings.Contains(mips[0].String(), ips[0].String()) {
				priority = append(priority, key)
				masterIpGroupExist = true
				break
			}
		}
		if priority != nil {
			break
		}
	}
	// 其他依次排序
	for i := 0; i <= cfg.ProxyCount; i++ {
		group := "proxygroup" + strconv.Itoa(i)
		if(masterIpGroupExist){
			if group != key {
				priority = append(priority, group)
			}
		}else {
			priority = append(priority, group)
		}
	}
	// 打印优先级
	if len(priority) > 0 {
		msg := "Priority: "
		for _, s := range priority {
			msg += s + " "
		}
		msg += "\n"
		fmt.Println(msg)
	}
	return priority
}

func IsRepetition(cfg *Config) error {
	ipMap := make(map[string]bool)

	for _, addr := range cfg.MasterAddrs {
		// 分割地址和端口
		parts := strings.Split(addr, ":")
		host := parts[0]

		// 解析域名
		ips, err := net.LookupIP(host)
		if err != nil {
			return err
		}
		// 域名对应多个IP则报错
		if len(ips) != 1 {
			errMsg := fmt.Sprintf("ERROR: The IPS:%v of URL:%s is not unique when masterConnection", ips, host)
			return errors.New(errMsg)
		}

		ip := ips[0].String()

		// 判断IP是否已存在
		if ipMap[ip] {
			errMsg := fmt.Sprintf("ERROR: IP:%s is not unique when masterConnection", ip)
			return errors.New(errMsg)
		} else {
			ipMap[ip] = true
		}
	}
	return nil
}
