package database

import (
	"context"
	"github.com/jackc/pgx/v5"
	"log"
	"os"
)

// GetPostgresInstance returns a Postgres database connection
// the function will fatally log and close the program if it can't establish a connection
func GetPostgresInstance(uri string) *pgx.Conn {
	conn, err := pgx.Connect(context.Background(), uri)
	if err != nil {
		log.Printf("Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	log.Println("Connected to Postgres database")
	return conn
}
