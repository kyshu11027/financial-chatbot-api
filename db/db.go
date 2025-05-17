package db

import (
	"database/sql"
	"finance-chatbot/api/logger"
	"fmt"
	"os"

	_ "github.com/lib/pq"
)

var DB *sql.DB

// InitDB initializes the database connection
func InitDB() error {
	// Get database connection string from environment variable
	dbURL := os.Getenv("SUPABASE_CONN_URI")
	if dbURL == "" {
		return fmt.Errorf("SUPABASE_CONN_URI environment variable not set")
	}

	// Open database connection
	var err error
	DB, err = sql.Open("postgres", dbURL)
	if err != nil {
		return fmt.Errorf("error opening database: %v", err)
	}

	// Test the connection
	err = DB.Ping()
	if err != nil {
		return fmt.Errorf("error connecting to the database: %v", err)
	}

	logger.Get().Info("Successfully connected to Supabase database")
	return nil
}

// CloseDB closes the database connection
func CloseDB() {
	if DB != nil {
		DB.Close()
	}
}
