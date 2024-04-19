package main

import (
	"fmt"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/rnr-capital/newsfeed-backend/utils/dotenv"
)

func main() {
	if err := dotenv.LoadDotEnvs(); err != nil {
		panic(err)
	}

	db, err := GetCustomizedConnection("testing")
	if err != nil {
		fmt.Println("Failed to connect to database:", err)
		return
	}

	// Ping the database to verify connection
	err = db.Exec("SELECT 1").Error
	if err != nil {
		fmt.Println("Failed to ping database:", err)
		return
	}

	fmt.Println("Successfully connected to database")
}

func GetCustomizedConnection(dbName string) (*gorm.DB, error) {
	sslmode := "require"
	if dbName == "testing" {
		sslmode = "prefer"
	}

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASS"),
		dbName,
		os.Getenv("DB_PORT"),
		sslmode,
	)

	return gorm.Open(postgres.Open(dsn), &gorm.Config{})
}
