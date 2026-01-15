package info

import (
	"database/sql"
	"fmt"
)

type Cluster struct {
	ClusterName string
	ClusterID   string
	Issingle    bool
}
type DN struct {
	Cluster  Cluster
	tenant   string
	Name     string
	Host     string
	Port     int
	Username string
	Password string
	isMaster bool
}
type CN struct {
	Cluster  Cluster
	tenant   string
	Name     string
	Host     string
	Port     int
	Username string
	Password string
}
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

func GetClusterInfo(mds *sql.DB) []Cluster {
	// 重构
	GetClusterIDSQL := "select cluster_id,cluster_name,issingle from mds.cluster_info where cluster_id <> '1024';"
	ClusterList := []Cluster{}
	rows, err := mds.Query(GetClusterIDSQL)
	if err != nil {
		fmt.Println("GetClusterID Query Error:", err)
	}
	defer rows.Close()
	for rows.Next() {
		var ClusterID string
		var ClusterName string
		var single int
		err := rows.Scan(&ClusterID, &ClusterName, &single)
		if err != nil {
			fmt.Println("GetClusterID Scan Error:", err)
		}
		ClusterList = append(ClusterList, Cluster{ClusterName: ClusterName, ClusterID: ClusterID, Issingle: (single == 1)})
	}
	return ClusterList

}
func GetMasterDN(mds *sql.DB, clusterid string, user string, password string) []string {
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
		dsn := user + ":" + password + "@tcp(" +
			dbip + ":" + fmt.Sprintf("%d", dbport) + ")/" +
			"" + "?loadbalance=false&blacklist=-1"
		DSNList = append(DSNList, dsn)
	}
	return DSNList

}
func GetSchema(mds *sql.DB, id string) []string {
	// id 是cluster_id
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
