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
type MDSDemo struct {
	DSN      string
	Username string
	Password string
}

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

		MDSDemosList = append(MDSDemosList, MDSDemo{DSN: dsn, Username: v.Username, Password: v.Password})
	}
	return MDSDemosList
}

func TestDSN(dsn string) bool {
	open, err := sql.Open("mysql", dsn)
	if err != nil {
		panic(err)
		return false
	}
	err = open.Ping()
	if err != nil {
		panic(err)
		return false
	}
	return true
}
func GetDBConnect(dsn string) *sql.DB {
	open, err := sql.Open("mysql", dsn)
	if err != nil {
		panic(err)
	}
	err = open.Ping()
	if err != nil {
		panic(err)
	}
	return open
}
