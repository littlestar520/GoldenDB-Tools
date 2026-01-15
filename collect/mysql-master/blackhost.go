// Go MySQL Driver - A MySQL-Driver for Go's database/sql package

package mysql

import (
	"sync"
	"time"
)

var (
	globalBlacklistLock sync.RWMutex
	// 坏CN
	globalBlacklist map[string]int64
)

// 获取当前毫秒级时间戳
func nowTime() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

// 坏CN加入map
func addToGlobalBlacklist(addr string, blackList int64) {
	globalBlacklistLock.Lock()
	defer globalBlacklistLock.Unlock()

	timeout := nowTime() + blackList
	if globalBlacklist == nil {
		globalBlacklist = make(map[string]int64)
	}
	globalBlacklist[addr] = timeout
}

// 返回所有坏CN
func getGlobalBlacklist() map[string]int64 {
	var blacklistClone map[string]int64
	globalBlacklistLock.Lock()
	defer globalBlacklistLock.Unlock()

	// 深拷贝一个map
	blacklistClone = cloneMap(globalBlacklist)

	// 剔除超时的CN
	for key, value := range blacklistClone {
		if value < nowTime() {
			delete(blacklistClone, key)
			delete(globalBlacklist, key)
		}
	}

	return blacklistClone

}

func cloneMap(tags map[string]int64) map[string]int64 {
	cloneTags := make(map[string]int64)
	for k, v := range tags {
		cloneTags[k] = v
	}
	return cloneTags
}

// 过滤坏CN
func filterBlackHost(addrs []string) []string {

	filterBlackList := make([]string, len(addrs))
	copy(filterBlackList, addrs)

	blackListMap := getGlobalBlacklist()
	globalBlacklistLock.Lock()
	defer globalBlacklistLock.Unlock()

	for key := range blackListMap {
		if contains(addrs, key) {
			filterBlackList = removeElement(filterBlackList, key)
		}
	}
	return filterBlackList
}

// 从[]string删除指定元素
func removeElement(arr []string, elem string) []string {
	var result []string
	for _, val := range arr {
		if val != elem {
			result = append(result, val)
		}
	}
	return result
}

// string类型切片中是否存在元素v
func contains(elems []string, v string) bool {
	for _, s := range elems {
		if v == s {
			return true
		}
	}
	return false
}
