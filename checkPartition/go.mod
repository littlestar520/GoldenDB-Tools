module GoldenDB

go 1.24

require github.com/go-sql-driver/mysql v1.7.1 // 声明依赖官方驱动

replace github.com/go-sql-driver/mysql v1.7.1 => ./mysql-master
