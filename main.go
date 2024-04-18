package main

import (
	"LowLogBackend/logging"
	"context"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"os"
	"time"
)

const (
	MongoDBURIEnvVar    = "MONGODB_URI"
	MongoDBNameEnvVar   = "MONGODB_DB"
	MongoDBDocsURL      = "https://www.mongodb.com/docs/drivers/go/current/usage-examples/#environment-variable"
	ErrorEnvVarNotFound = "you must set your '%s' environment variable. See more at %s"
)

type MongoDatabaseVariables struct {
	URI    string
	DBName string
}

func main() {
	dbVars, err := getDatabaseVariables()
	if err != nil {
		log.Fatal("Failed to load database variables: ", err)
	}

	database := getDatabase(dbVars.URI, dbVars.DBName)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := database.Client().Disconnect(ctx); err != nil {
			log.Fatalf("Failed to disconnect from MongoDB: %v", err)
		}
	}()

	//Create a new gin server
	gServer := gin.Default()

	//Setting up routes
	logging.CreateRoutes(gServer, database)

	//Run server after establishing routes
	runErr := gServer.Run()
	if runErr != nil {
		log.Fatalln(runErr)
	}
}

func getDatabaseVariables() (*MongoDatabaseVariables, error) {
	if err := godotenv.Load(); err != nil {
		return nil, errors.New("no .env file found, will try to use run environment variables")
	}

	uri := os.Getenv(MongoDBURIEnvVar)
	if uri == "" {
		return nil, fmt.Errorf(ErrorEnvVarNotFound, MongoDBURIEnvVar, MongoDBDocsURL)
	}

	dbName := os.Getenv(MongoDBNameEnvVar)
	if dbName == "" {
		return nil, fmt.Errorf(ErrorEnvVarNotFound, MongoDBNameEnvVar, MongoDBDocsURL)
	}

	return &MongoDatabaseVariables{URI: uri, DBName: dbName}, nil
}

// getDatabase returns a MongoDB database object
// the function will fatally log and close the program if it can't establish a connection
func getDatabase(uri string, dbName string) *mongo.Database {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOptions := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatalf("Failed to ping MongoDB: %v", err)
	}

	log.Println("Successfully connected and pinged MongoDB.")
	return client.Database(dbName)
}
