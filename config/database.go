package config

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/denisenkom/go-mssqldb"
)

var DB *sql.DB

func ConnectDB(cfg *EnvConfig) {
	connStr := fmt.Sprintf(
		"server=%s;port=%s;user id=%s;password=%s;database=%s;encrypt=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName,
	)

	var err error
	DB, err = sql.Open("mssql", connStr)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}

	DB.SetMaxOpenConns(25)
	DB.SetMaxIdleConns(10)

	if err = DB.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	log.Println("Database connected successfully")
}
