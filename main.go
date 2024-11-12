package main

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/ScooballyD/chirpy/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		fmt.Printf("unable to open database: %v", err)
	}
	dbQ := database.New(db)

	fmt.Println("starting server")
	StartServer(dbQ)
}
