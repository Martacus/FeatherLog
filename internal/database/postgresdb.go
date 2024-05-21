package database

import (
	"context"
	"errors"
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

func RunPostgresInitializationScript(conn *pgx.Conn) {
	initSQL, err := os.ReadFile("database/init.sql")
	if err != nil {
		log.Printf("Unable to read init.sql: %v", err)
		return
	}

	tx, err := conn.Begin(context.Background())
	if err != nil {
		log.Printf("Unable to start a transaction for init script: %v", err)
		return
	}

	defer func(tx pgx.Tx, ctx context.Context) {
		err := tx.Rollback(ctx)
		if err != nil {
			if !errors.Is(err, pgx.ErrTxClosed) {
				log.Printf("Unable to roleback init script transaction: %v", err)
			}
		}
	}(tx, context.Background())

	if _, err := tx.Exec(context.Background(), string(initSQL)); err != nil {
		log.Printf("Error executing init.sql: %v", err)
		return
	}

	if err := tx.Commit(context.Background()); err != nil {
		log.Printf("Error committing transaction: %v", err)
		return
	}

	log.Println("init.sql script executed successfully")
}

func ExecuteTransaction(conn *pgx.Conn, ctx context.Context, fn func(tx pgx.Tx) (interface{}, error)) (interface{}, error) {
	tx, err := conn.Begin(ctx)
	if err != nil {
		log.Printf("Unable to begin transaction: %v", err)
		return nil, err
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			log.Printf("Error rolling back transaction: %v", err)
		}
	}()

	result, err := fn(tx)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		log.Printf("Error committing transaction: %v", err)
		return "", err
	}

	return result, nil
}
