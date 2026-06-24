package db

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

// DB is the global database connection pool
var DB *sql.DB

// InitDB connects to the database and ensures our table exists
func InitDB() {
	// 1. Connection string matching our docker-compose setup
	connStr := "host=127.0.0.1 port=5432 user=admin password=supersecretpassword dbname=lucid_ci sslmode=disable"
	var err error
	DB, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to open DB connection: %v\n", err)
	}

	// 2. Test the connection
	err = DB.Ping()
	if err != nil {
		log.Fatalf("Failed to ping DB. Is Docker running? Error: %v\n", err)
	}

	fmt.Println("✅ Successfully connected to PostgreSQL!")

	// 3. Create the table automatically if this is the first time running
	createTableQuery := `
	CREATE TABLE IF NOT EXISTS scan_results (
		id SERIAL PRIMARY KEY,
		repo_name VARCHAR(255),
		commit_sha VARCHAR(255),
		status VARCHAR(50),
		created_at TIMESTAMP
	);`

	_, err = DB.Exec(createTableQuery)
	if err != nil {
		log.Fatalf("Failed to create table: %v\n", err)
	}
}
