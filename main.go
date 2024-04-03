package main

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"net/http"
	"os"
	"time"
)

type JsonLog struct {
	Domain    string `json:"domain"`
	Group     string `json:"group"`
	Tag       string `json:"tag"`
	Log       string `json:"log"`
	Timestamp int64  `json:"timestamp"`
}

type Domain struct {
	Domain string `json:"domain"`
}

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, will try to use run env vars.")
	}

	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		log.Fatal("You must set your 'MONGODB_URI' environment variable. See\n\t https://www.mongodb.com/docs/drivers/go/current/usage-examples/#environment-variable")
	}

	dbName := os.Getenv("MONGODB_DB")
	if dbName == "" {
		log.Fatal("You must set your 'dbName' environment variable. See\n\t https://www.mongodb.com/docs/drivers/go/current/usage-examples/#environment-variable")
	}

	database := getDatabase(uri, dbName)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := database.Client().Disconnect(ctx); err != nil {
			log.Fatalf("Failed to disconnect from MongoDB: %v", err)
		}
	}()

	//Create a new gin server
	gServer := gin.Default()

	//Gets all logs for a domain
	gServer.GET("/log/:domain", func(c *gin.Context) {
		var results []JsonLog
		domain := c.Param("domain")
		coll := database.Collection(domain)

		err := executeQueryWithTimeout(func(ctx context.Context) error {
			opts := options.Find().SetSort(bson.D{{"timestamp", -1}})
			cur, findErr := coll.Find(ctx, bson.D{{}}, opts)
			if findErr != nil {
				return findErr
			}
			return cur.All(ctx, &results)
		})

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, results)
	})

	//Gets a list of domains
	gServer.GET("/domain/list", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		collections, err := database.ListCollectionNames(ctx, bson.D{{}})
		if err != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		domains := make([]Domain, 0)
		for _, collection := range collections {
			domains = append(domains, Domain{Domain: collection})
		}

		c.JSON(http.StatusOK, domains)
	})

	//Posts a new log to a domain
	gServer.POST("/log", func(c *gin.Context) {
		var logEntry JsonLog
		if err := c.BindJSON(&logEntry); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		logEntry.Timestamp = time.Now().UTC().UnixMilli()

		coll := database.Collection(logEntry.Domain)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		result, err := coll.InsertOne(ctx, logEntry)
		if err != nil {
			log.Println("Failed to insert logEntry:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		log.Printf("Log inserted %s\n", logEntry.Log)
		c.JSON(http.StatusOK, result)
	})

	//Run server after establishing routes
	runErr := gServer.Run()
	if runErr != nil {
		log.Fatalln(runErr)
	}
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

// executeQueryWithTimeout executes a MongoDB operation with a 10-second timeout
// the function will log and return an error if the operation fails
// more context may be added in the future
func executeQueryWithTimeout(op func(ctx context.Context) error) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := op(ctx)
	if err != nil {
		log.Printf("MongoDB operation failed: %v", err)
		return err
	}
	return nil
}
