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

func main() {
	err := loadConfig()
	if err != nil {
		log.Fatal("Failed to load configuration file: ", err)
	}

	//Create the database
	database := getDatabase(config.String(MongodbUri), config.String(MongodbDb))
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

// loadConfig loads the configuration file to be used by the application
func loadConfig() error {
	config.WithOptions(config.ParseEnv)
	config.AddDriver(yaml.Driver)

	return config.LoadFiles("config.yml")
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
