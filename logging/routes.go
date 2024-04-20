package logging

import (
	"LowLogBackend/utility"
	"context"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"net/http"
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

func CreateRoutes(engine *gin.Engine, database *mongo.Database) {
	//Gets all logs for a domain
	engine.GET("/log/:domain", func(c *gin.Context) {
		var results []JsonLog
		domain := c.Param("domain")
		coll := database.Collection(domain)

		err := utility.ExecuteQueryWithTimeout(func(ctx context.Context) error {
			opts := options.Find().SetSort(bson.D{{"timestamp", -1}})
			cur, findErr := coll.Find(ctx, bson.D{{}}, opts)
			if findErr != nil {
				return findErr
			}
			return cur.All(ctx, &results)
		})

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "success", "data": results})
	})

	//Gets a list of domains
	engine.GET("/domain/list", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		collections, err := database.ListCollectionNames(ctx, bson.D{{}})
		if err != nil {
			log.Println("Error: ", err)
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
			return
		}

		domains := make([]Domain, 0)
		for _, collection := range collections {
			domains = append(domains, Domain{Domain: collection})
		}

		c.JSON(http.StatusOK, gin.H{"status": "success", "data": domains})
	})

	//Posts a new log to a domain
	engine.POST("/log", func(c *gin.Context) {
		var logEntry JsonLog
		if err := c.BindJSON(&logEntry); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
			return
		}
		logEntry.Timestamp = time.Now().UTC().UnixMilli()

		coll := database.Collection(logEntry.Domain)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		result, err := coll.InsertOne(ctx, logEntry)
		if err != nil {
			log.Println("Failed to insert logEntry:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
			return
		}

		log.Printf("Log inserted %s\n", logEntry.Log)
		c.JSON(http.StatusOK, gin.H{"status": "success", "data": result})
	})
}
