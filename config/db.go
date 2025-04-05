package config

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/jackc/pgx/v4/stdlib"
)

var DB *sql.DB

func ConnectDB() {
	var err error
	dsn := os.Getenv("DATABASE_URL") 
	DB, err = sql.Open("pgx", dsn)
	if err != nil {
		panic(err)
	}

	err = DB.Ping()
	if err != nil {
		panic(err)
	}

	fmt.Println("Connected to PostgreSQL!")
}
