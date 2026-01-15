package alarm

import (
	"database/sql"
	"fmt"
)

type DDLError struct {
	resourceid string
	content    string
	createtime string
	updatetime string
	Reserve4   string
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
type DN struct {
	Name     string
	Host     string
	Port     int
	Username string
	Password string
}

func GetClusterID(mds *sql.DB) []string {
	// 重构
	GetClusterIDSQL := "select distinct cluster_id from mds.db_info where cluster_id < 30;"
	ClusterIDList := []string{}
	rows, err := mds.Query(GetClusterIDSQL)
	if err != nil {
		fmt.Println("GetClusterID Query Error:", err)
		return ClusterIDList
	}
	defer rows.Close()
	for rows.Next() {
		var clusterid string
		err := rows.Scan(&clusterid)
		if err != nil {
			fmt.Println("GetClusterID Scan Error:", err)
		}
		ClusterIDList = append(ClusterIDList, clusterid)
	}
	return ClusterIDList
}
func GetClusterName(mds *sql.DB, id string) string {
	// 重构
	GetClusterNameSQL := "select cluster_name from mds.cluster_info where cluster_id=?"
	var ClusterName string
	rows, err := mds.Query(GetClusterNameSQL, id)
	if err != nil {
		fmt.Println("GetClusterName Query Error:", err)
		return ClusterName
	}
	defer rows.Close()
	for rows.Next() {
		err := rows.Scan(&ClusterName)
		if err != nil {
			fmt.Println("GetClusterName Scan Error:", err)
		}
	}
	return ClusterName
}
func GetDNinfo(mds *sql.DB, clusterid string, name string, password string) []string {
	sqlstr := "select db_ip, db_port from mds.db_info where cluster_id=? and db_role=0"
	DSNList := []string{}
	rows, err := mds.Query(sqlstr, clusterid)
	if err != nil {
		fmt.Println("GetDNinfo Query Error:", err)
		return DSNList
	}
	defer rows.Close()
	for rows.Next() {
		var dbip string
		var dbport int
		err := rows.Scan(&dbip, &dbport)
		if err != nil {
			fmt.Println("GetDNinfo Scan Error:", err)
		}
		dsn := name + ":" + password + "@tcp(" +
			dbip + ":" + fmt.Sprintf("%d", dbport) + ")/" +
			"" + "?loadbalance=false&blacklist=-1"
		DSNList = append(DSNList, dsn)
	}
	return DSNList

}

func GetSchema(mds *sql.DB, id string) []string {
	sqlstr := "select database_name from mds.dictionary_info where type=1 and cluster_id=? and database_name not in ('_gdb_sysdb','heartbeat_info','processlist','sys')"
	SchemaList := []string{}
	rows, err := mds.Query(sqlstr, id)
	if err != nil {
		fmt.Println("GetSchema Query Error:", err)
		return SchemaList
	}
	defer rows.Close()
	for rows.Next() {
		var schemaname string
		err := rows.Scan(&schemaname)
		if err != nil {
			fmt.Println(err)
		}
		SchemaList = append(SchemaList, schemaname)
	}
	return SchemaList
}
