package main

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/gookit/config/v2"
	"github.com/gookit/config/v2/yaml"
	"github.com/jackc/pgx/v5"
	_ "github.com/lib/pq"
	"github.com/swaggo/files"
	"github.com/swaggo/gin-swagger"
	"go.mongodb.org/mongo-driver/mongo"
	"gofeather/internal/auth"
	"gofeather/internal/constants"
	"gofeather/internal/database"
	"gofeather/internal/featureflags"
	"gofeather/internal/logging"
	"log"
	"time"
)

//	@title			GoFeather API
//	@version		0.1
//	@description	This is the api docs for the featherlog application, it will show all routes, even the disabled ones.
//	@termsOfService	http://swagger.io/terms/

//	@license.name	Apache 2.0
//	@license.url	http://www.apache.org/licenses/LICENSE-2.0.html

// @host	localhost:*
func main() {
	err := loadConfig()
	if err != nil {
		log.Fatal("Failed to load configuration file: ", err)
	}

	var mongoDB *mongo.Database
	var postgresConn *pgx.Conn

	//Creating the mongodb instance
	if config.Bool(constants.MongodbDbEnabled) {
		mongoDB = database.GetMongoDBInstance(config.String(constants.MongodbUri), config.String(constants.MongodbDb))
		defer func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := mongoDB.Client().Disconnect(ctx); err != nil {
				log.Fatalf("Failed to disconnect from MongoDB: %v", err)
			}
		}()
	}

	//Creating a postgres connection
	if config.Bool(constants.PostgresEnabled) {
		postgresConn = database.GetPostgresInstance(config.String(constants.PostgresURL))
		defer func(conn *pgx.Conn, ctx context.Context) {
			err := conn.Close(ctx)
			if err != nil {
				log.Fatalf("Failed to disconnect from Postgres: %v", err)
			}
		}(postgresConn, context.Background())
		database.RunPostgresInitializationScript(postgresConn)
	}

	//Create a new gin server
	log.Println("Starting REST API")
	server := gin.Default()

	//Setting up routes
	if config.Bool(constants.LogFeature) {
		logging.CreateRoutes(server, mongoDB)
	}
	if config.Bool(constants.FeatureFlagFeature) {
		featureflags.Init(server, mongoDB)
	}

	if config.Bool(constants.AuthFeature) {
		auth.Init(server, postgresConn)
	}

	//Setup swagger route
	server.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	//Run server after establishing routes
	runErr := server.Run()
	if runErr != nil {
		log.Fatalln(runErr)
	}
}

// loadConfig loads the configuration file to be used by the application
func loadConfig() error {
	config.WithOptions(config.ParseEnv)
	config.AddDriver(yaml.Driver)

	return config.LoadFiles("config/config.yml")
}
