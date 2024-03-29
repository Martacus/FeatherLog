package main

import (
	"github.com/gin-gonic/gin"
	rethink "gopkg.in/rethinkdb/rethinkdb-go.v6"
	"log"
	"net/http"
)

type JsonLog struct {
	Domain string   `json:"domain"`
	Path   string   `json:"path"`
	Tags   []string `json:"tags"`
	Log    string   `json:"log"`
}

var dbName = "lw_logger"

func main() {
	// Connect to RethinkDB server
	session, rethinkErr := rethink.Connect(rethink.ConnectOpts{
		Address: "localhost:28015", // default
	})
	if rethinkErr != nil {
		log.Fatalln(rethinkErr)
	}

	cdbErr := createDatabase(session)
	if cdbErr != nil {
		log.Fatalln(cdbErr)
	}

	ctbErr := createTableIfNotExists(session, "logs")
	if ctbErr != nil {
		log.Fatalln(ctbErr)
	}

	//Gin Server
	gServer := gin.Default()

	// ET
	gServer.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	gServer.GET("/log/:domain", func(c *gin.Context) {
		domain := c.Param("domain")
		var results []JsonLog
		res, err := rethink.DB(dbName).Table(domain).Run(session)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}

		if err = res.All(&results); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, results)
	})

	//
	gServer.POST("/log", func(c *gin.Context) {
		var logEntry JsonLog
		if err := c.BindJSON(&logEntry); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := createTableIfNotExists(session, logEntry.Domain); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		err := rethink.DB(dbName).Table(logEntry.Domain).Insert(logEntry).Exec(session)
		if err != nil {
			log.Println("Failed to insert logEntry:", err)
		}

		log.Printf("Log inserted %s\n", logEntry.Log)
	})
	err := gServer.Run()
	if err != nil {
		log.Fatalln(err)
	}

}

func createTableIfNotExists(session *rethink.Session, tbName string) error {
	existingTables, err := checkTablesExists(session, tbName)
	if err != nil {
		log.Fatalln(err)
		return err
	}

	for tableName, exists := range existingTables {
		if exists {
			log.Printf("Table %s exists\n", tableName)
		} else {
			if err := createTable(session, tableName); err != nil {
				log.Fatalln(err)
				return err
			}
		}
	}

	return nil
}

func createTable(session *rethink.Session, tbName string) error {
	_, err := rethink.DB(dbName).TableCreate(tbName).Run(session)
	if err != nil {
		log.Fatalln(err)
		return err
	}
	log.Printf("Table %s created\n", tbName)

	return nil
}

func checkTablesExists(session *rethink.Session, tableNames ...string) (map[string]bool, error) {
	res, err := rethink.DB(dbName).TableList().Run(session)
	if err != nil {
		return nil, err
	}

	var tables []string
	if err := res.All(&tables); err != nil {
		return nil, err
	}

	existsMap := make(map[string]bool)

	for _, tableName := range tableNames {
		existsMap[tableName] = false
	}

	for _, t := range tables {
		if _, ok := existsMap[t]; ok {
			existsMap[t] = true
		}
	}

	return existsMap, nil
}

func createDatabase(session *rethink.Session) error {
	dbs, err := rethink.DBList().Run(session)
	if err != nil {
		log.Fatalf("Error listing databases: %s", err)
		return err
	}

	var dbList []string
	err = dbs.All(&dbList)
	if err != nil {
		log.Fatalf("Error reading database list: %s", err)
		return err
	}

	dbExists := false
	for _, db := range dbList {
		if db == dbName {
			dbExists = true
			break
		}
	}

	if !dbExists {
		_, err = rethink.DBCreate(dbName).RunWrite(session)
		if err != nil {
			log.Fatalf("Error creating database %s: %s", dbName, err)
			return err
		} else {
			log.Printf("Database %s created successfully.", dbName)
		}
	} else {
		log.Printf("Database %s already exists.", dbName)
	}
	return nil
}
