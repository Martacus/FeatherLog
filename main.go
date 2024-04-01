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

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		log.Fatal("You must set your 'MONGODB_URI' environment variable. See\n\t https://www.mongodb.com/docs/drivers/go/current/usage-examples/#environment-variable")
	}

	dbName := os.Getenv("MONGODB_DB")
	if dbName == "" {
		log.Fatal("You must set your 'dbName' environment variable. See\n\t https://www.mongodb.com/docs/drivers/go/current/usage-examples/#environment-variable")
	}

	client, mongoErr := mongo.Connect(context.TODO(), options.Client().ApplyURI(uri))
	if mongoErr != nil {
		panic(mongoErr)
	}
	defer func() {
		if err := client.Disconnect(context.TODO()); err != nil {
			panic(err)
		}
	}()

	database := client.Database(dbName)

	//Gin Server
	gServer := gin.Default()

	gServer.GET("/log/:domain", func(c *gin.Context) {
		domain := c.Param("domain")
		coll := database.Collection(domain)

		var results []JsonLog
		cur, err := coll.Find(context.TODO(), bson.D{{}})
		if err != nil {
			panic(err)
		}

		curAllErr := cur.All(context.TODO(), &results)
		if curAllErr != nil {
			panic(curAllErr)
		}

		c.JSON(200, results)
	})

	gServer.GET("/log/:domain/group/list", func(c *gin.Context) {
		domain := c.Param("domain")
		coll := database.Collection(domain)

		res, err := coll.Distinct(context.TODO(), "group", bson.D{{}})
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		log.Println(len(res))

		c.JSON(200, res)
	})

	gServer.GET("/domain/list", func(c *gin.Context) {
		collections, err := database.ListCollectionNames(context.TODO(), bson.D{{}})
		if err != nil {
			panic(err)
		}

		c.JSON(200, collections)
	})

	gServer.POST("/log", func(c *gin.Context) {
		var logEntry JsonLog
		if err := c.BindJSON(&logEntry); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		logEntry.Timestamp = time.Now().UTC().UnixMilli()

		coll := database.Collection(logEntry.Domain)
		result, err := coll.InsertOne(context.TODO(), logEntry)
		if err != nil {
			log.Println("Failed to insert logEntry:", err)
		}

		log.Printf("Log inserted %s\n", logEntry.Log)
		c.JSON(200, result)
	})

	err := gServer.Run()
	if err != nil {
		log.Fatalln(err)
	}
}
