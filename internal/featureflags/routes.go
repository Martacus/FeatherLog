package featureflags

import (
	"context"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gookit/config/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"gofeather/internal/utility"
	"log"
	"net/http"
	"time"
)

const (
	CollectionName = "featureflags"
	Route          = "feature_flags_route"
)

type FFUserData struct {
	Roles []string `json:"roles"`
}

// FeatureFlag is an object representing a feature flag.
type FeatureFlag struct {
	Name    string          `json:"name"`
	Enabled bool            `json:"enabled"`
	Filters []FeatureFilter `json:"filters"`
}

type RoleFilter struct {
	Type  string   `json:"type"`
	Roles []string `json:"roles"`
}

type TimeFilter struct {
	Type      string    `json:"type"`
	TimeStart time.Time `json:"time_start"`
	TimeStop  time.Time `json:"time_stop"`
}
type FeatureFilter interface {
	CanUse(data FFUserData) bool
}

func (f RoleFilter) CanUse(data FFUserData) bool {
	roleSet := make(map[string]struct{})
	for _, role := range data.Roles {
		roleSet[role] = struct{}{}
	}
	for _, role := range f.Roles {
		if _, exists := roleSet[role]; !exists {
			return false
		}
	}
	return true
}

func (f TimeFilter) CanUse(_ FFUserData) bool {
	now := time.Now()
	return !now.Before(f.TimeStart) && !now.After(f.TimeStop)
}

// CreateRoutes creates the routes for the feature flag service
func CreateRoutes(engine *gin.Engine, database *mongo.Database) {
	identifier := config.String(Route)

	//Retrieves all flags
	engine.GET("/"+identifier+"/flags", func(c *gin.Context) {
		var results []FeatureFlag
		coll := database.Collection(CollectionName)

		err := utility.ExecuteQueryWithTimeout(func(ctx context.Context) error {
			cur, findErr := coll.Find(ctx, bson.D{{}})
			if findErr != nil {
				return findErr
			}
			return cur.All(ctx, &results)
		})

		if err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}

		c.JSON(http.StatusOK, results)
	})

	//Retrieve a single flag
	engine.GET("/"+identifier+"/flag/:name", func(c *gin.Context) {
		var retrievedFlag FeatureFlag

		coll := database.Collection(CollectionName)
		flagName := c.Param("name")
		filter := bson.D{{"name", flagName}}

		err := utility.ExecuteQueryWithTimeout(func(ctx context.Context) error {
			findErr := coll.FindOne(ctx, filter).Decode(&retrievedFlag)
			if findErr != nil {
				if errors.Is(findErr, mongo.ErrNoDocuments) {
					return nil
				}
				return findErr
			}
			return nil
		})

		if err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "success", "data": retrievedFlag})
	})

	engine.GET("/"+identifier+"/flag/:name/check", func(c *gin.Context) {
		var retrievedFlag FeatureFlag

		coll := database.Collection(CollectionName)
		flagName := c.Param("name")
		filter := bson.D{{"name", flagName}}

		err := utility.ExecuteQueryWithTimeout(func(ctx context.Context) error {
			findErr := coll.FindOne(ctx, filter).Decode(&retrievedFlag)
			if findErr != nil {
				if errors.Is(findErr, mongo.ErrNoDocuments) {
					return nil
				}
				return findErr
			}
			return nil
		})

		if err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}

		var testRoles = make([]string, 0)
		testRoles = append(testRoles, "yest")

		for i := 0; i < len(retrievedFlag.Filters); i++ {
			if !retrievedFlag.Filters[i].CanUse(FFUserData{Roles: testRoles}) {
				c.JSON(http.StatusOK, gin.H{"status": "success", "data": gin.H{"enabled": false}})
				return
			}
		}

		c.JSON(http.StatusOK, gin.H{"enabled": true})
	})

	//Create a flag
	engine.POST("/"+identifier+"/flag", func(c *gin.Context) {
		var flag FeatureFlag
		if err := c.BindJSON(&flag); err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}

		coll := database.Collection(CollectionName)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		result, err := coll.InsertOne(ctx, flag)
		if err != nil {
			log.Println("Failed to insert flag:", err)
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}

		log.Printf("Flag created %v\n", flag)
		c.JSON(http.StatusOK, result)
	})
}

func (ff *FeatureFlag) UnmarshalBSON(data []byte) error {
	var raw struct {
		Name    string          `bson:"name"`
		Enabled bool            `bson:"enabled"`
		Filters []bson.RawValue `bson:"filters"`
	}
	if err := bson.Unmarshal(data, &raw); err != nil {
		return err
	}
	ff.Name = raw.Name
	ff.Enabled = raw.Enabled
	ff.Filters = make([]FeatureFilter, len(raw.Filters))

	for i, rawFilter := range raw.Filters {
		var filterType struct {
			Type string `bson:"type"`
		}
		if err := rawFilter.Unmarshal(&filterType); err != nil {
			return err
		}
		switch filterType.Type {
		case "RoleFilter":
			var filter RoleFilter
			if err := rawFilter.Unmarshal(&filter); err != nil {
				return err
			}
			ff.Filters[i] = filter
		case "TimeFilter":
			var filter TimeFilter
			if err := rawFilter.Unmarshal(&filter); err != nil {
				return err
			}
			ff.Filters[i] = filter
		default:
			return fmt.Errorf("unknown filter type: %s", filterType.Type)
		}
	}
	return nil
}
