package test

import "database/sql"

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
