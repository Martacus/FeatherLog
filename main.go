package main

import (
	"LowLogBackend/logging"
	"context"
	"github.com/gin-gonic/gin"
	"github.com/gookit/config/v2"
	"github.com/gookit/config/v2/yaml"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"time"
)

const (
	MongodbUri = "mongodb_uri"
	MongodbDb  = "mongodb_uri"
)

type ConfigVariables struct {
	MongodbUri string
	MongodbDB  string
}

func main() {
	dbVars, err := getDatabaseVariables()
	if err != nil {
		log.Fatal("Failed to load database variables: ", err)
	}

	database := getDatabase(dbVars.MongodbUri, dbVars.MongodbDB)
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

func getDatabaseVariables() (*ConfigVariables, error) {
	config.WithOptions(config.ParseEnv)

	// add driver for support yaml content
	config.AddDriver(yaml.Driver)

	err := config.LoadFiles("config.yml")
	if err != nil {
		return nil, err
	}

	return &ConfigVariables{config.String(MongodbUri), config.String(MongodbDb)}, nil
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
