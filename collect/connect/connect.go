package connect

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"io"
	"os"
)

const EncryptionKey = "879kf28Ls987kF982k789lK87982k789"

type MDS struct {
	Name     string `json:"name"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
}
type CN struct {
	Name     string `json:"name"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Schema   string `json:"schema"`
	Username string `json:"username"`
	Password string `json:"password"`
}
type CNConnect struct {
	DSN      string
	Username string
	Password string
	Schema   string
}

type MDSDemo struct {
	DSN      string
	Name     string
	Username string
	Password string
}

// 加密函数
func Decrypt(input string) (string, error) {
	key := []byte(EncryptionKey)
	data, err := base64.StdEncoding.DecodeString(input)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// Extract nonce (first 12 bytes) and ciphertext
	if len(data) < 12 {
		return "", fmt.Errorf("invalid ciphertext")
	}
	nonce, ciphertext := data[:12], data[12:]

	plaintext, err := aesgcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// 解密函数
func Encrypt(input string) (string, error) {
	key := []byte(EncryptionKey)
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	ciphertext := aesgcm.Seal(nil, nonce, []byte(input), nil)

	// Combine nonce and ciphertext, then base64 encode
	result := append(nonce, ciphertext...)
	return base64.StdEncoding.EncodeToString(result), nil
}

func GetMDS() []MDSDemo {
	var MDSList []MDS
	file, err := os.ReadFile("config/mds.json")
	if err != nil {
		panic(err)
	}
	var MDSDemosList []MDSDemo
	err = json.Unmarshal(file, &MDSList)
	if err != nil {
		panic(err)
	}
	for _, v := range MDSList {
		v.Password, _ = Decrypt(v.Password)
		dsn := v.Username + ":" + v.Password + "@tcp(" +
			v.Host + ":" + fmt.Sprintf("%d", v.Port) + ")/" +
			"mds" + "?loadbalance=false&blacklist=-1"

		MDSDemosList = append(MDSDemosList, MDSDemo{DSN: dsn, Username: v.Username, Password: v.Password, Name: v.Name})
	}
	return MDSDemosList
}
func GetCN() []CNConnect {
	var CNList []CN
	var DemoList []CNConnect
	file, err := os.ReadFile("config/mds.json")
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(file, &CNList)
	if err != nil {
		panic(err)
	}
	for _, v := range CNList {
		v.Password, _ = Decrypt(v.Password)
		dsn := v.Username + ":" + v.Password + "@tcp(" +
			v.Host + ":" + fmt.Sprintf("%d", v.Port) + ")/" +
			v.Schema + "?loadbalance=false&blacklist=-1"
		DemoList = append(DemoList, CNConnect{DSN: dsn, Username: v.Username, Password: v.Password, Schema: v.Schema})
	}
	return DemoList
}

func GetDBConnect(dsn string) *sql.DB {
	return GetDBConnects([]string{dsn})[0]
}
func GetDBConnects(dsns []string) []*sql.DB {
	// 初始化一个切片，用于存储所有成功创建的数据库连接
	dbList := make([]*sql.DB, 0, len(dsns))
	// 遍历每一个传入的DSN
	for _, dsn := range dsns {
		open, err := sql.Open("mysql", dsn)
		if err != nil {
			panic("dsn: " + dsn + ", open fail: " + err.Error())
		}
		// Ping校验连接可用性，sql.Open是懒加载，必须Ping才会真正建联
		err = open.Ping()
		if err != nil {
			panic("dsn: " + dsn + ", ping fail: " + err.Error())
		}
		// 连接成功，加入切片
		dbList = append(dbList, open)
	}
	return dbList
}
