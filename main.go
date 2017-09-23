package main

import (
	"log"
	"os"
	"time"

	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/urfave/cli"
	r "gopkg.in/gorethink/gorethink.v3"
)

func main() {

	app := cli.NewApp()

	app.Name = "Aevo Vault"
	app.Usage = "Aevo data storage API."

	app.Version = "1.0-dev"
	app.Copyright = "(c) 2017 Harrison Grodin"
	app.Authors = []cli.Author{
		cli.Author{
			Name:  "Harrison Grodin",
			Email: "grodinh@winchesterthurston.org",
		},
	}

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "ip",
			Value:  ":3000",
			Usage:  "server IP address",
			EnvVar: "AEVO_IP",
		},
		cli.StringFlag{
			Name:   "database-ip",
			Value:  "localhost:28015",
			Usage:  "database IP address",
			EnvVar: "AEVO_DATABASE_IP",
		},
		cli.StringFlag{
			Name:        "db-data",
			Value:       "data",
			Usage:       "database for storing data",
			Destination: &dbData,
			EnvVar:      "AEVO_DB_DATA",
		},
		cli.StringFlag{
			Name:        "db-model",
			Value:       "model",
			Usage:       "database for storing schemas and models",
			Destination: &dbModel,
			EnvVar:      "AEVO_DB_MODEL",
		},
		cli.StringFlag{
			Name:        "data-primary-key",
			Value:       "time",
			Usage:       "primary data storage key",
			Destination: &primaryKey,
			EnvVar:      "AEVO_DATA_PRIMARY_KEY",
		},
	}

	app.Action = func(c *cli.Context) error {
		route(
			c.String("ip"),
			c.String("database-ip"),
		)
		return nil
	}

	app.Commands = []cli.Command{
		{
			Name:    "generate",
			Aliases: []string{"gen", "g"},
			Usage:   "generate databases",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "database-ip",
					Value: "localhost:28015",
					Usage: "database IP address",
				},
			},
			Action: func(c *cli.Context) error {

				session, err := r.Connect(r.ConnectOpts{
					Address: c.String("database-ip"),
				})
				if err != nil {
					log.Fatalln(err)
					return nil
				}

				if _, err := r.DBCreate(dbModel).RunWrite(session); err != nil {
					log.Fatalln("Unable to create model database: ", err)
					return nil
				}

				if _, err := r.DBCreate(dbData).RunWrite(session); err != nil {
					log.Fatalln("Unable to create data database: ", err)
					return nil
				}

				session.Use(dbModel)

				if _, err := r.TableCreate("schema").RunWrite(session); err != nil {
					log.Fatalln("Unable to create schema table: ", err)
					return nil
				}

				if _, err := r.TableCreate("model").RunWrite(session); err != nil {
					log.Fatalln("Unable to create model table: ", err)
					return nil
				}

				return nil
			},
		},
	}

	app.Run(os.Args)

}

var primaryKey = "time"
var dbModel, dbData = "model", "data"

func route(IP string, databaseIP string) {

	router := gin.Default()
	scope := router.Group("/scope")

	dataSession, err := r.Connect(r.ConnectOpts{
		Address:  databaseIP,
		Database: dbData,
	})
	if err != nil {
		log.Fatalln(err)
	}
	routeState(scope, dataSession)

	modelSession, err := r.Connect(r.ConnectOpts{
		Address:  databaseIP,
		Database: dbModel,
	})
	if err != nil {
		log.Fatalln(err)
	}
	routeSchema(scope, dataSession, modelSession)
	routeModel(scope, modelSession)

	router.Run(IP)

}

////////

type state map[string]interface{}
type states []state

type dataInput interface {
	parseMapTime()
}

func (state state) parseMapTime() {
	if val, ok := state[primaryKey].(string); ok {

		t, err := time.Parse(time.RFC3339, val)

		if err == nil {
			state[primaryKey] = t
		} else {
			state[primaryKey] = time.Now()
		}

	} else {
		state[primaryKey] = time.Now()
	}
}

func (states states) parseMapTime() {
	for _, state := range states {
		state.parseMapTime()
	}
}

func routeState(router gin.IRouter, session *r.Session) {
	router.GET("/:scope", func(c *gin.Context) {
		res, err := r.Table(c.Param("scope")).OrderBy(r.OrderByOpts{
			Index: primaryKey,
		}).Run(session)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "scope not registered"})
			return
		}
		defer res.Close()

		var data []interface{}
		if err := res.All(&data); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error scanning database result"})
			return
		}

		c.JSON(http.StatusOK, data)
	})

	router.POST("/:scope", func(c *gin.Context) {
		storeState := func(data dataInput) {
			if c.BindJSON(&data) != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON data"})
				return
			}

			data.parseMapTime()

			_, err := r.Table(c.Param("scope")).Insert(data, r.InsertOpts{Conflict: "replace"}).RunWrite(session)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "scope not registered"})
				return
			}

			c.JSON(http.StatusOK, data)
		}

		multi := c.DefaultQuery("multi", "false")
		if multi != "true" {
			var data state
			storeState(&data)
		} else {
			var data states
			storeState(&data)
		}
	})
}

////////

func routeSchema(router gin.IRouter, dataSession *r.Session, session *r.Session) {
	router.GET("/:scope/schema", func(c *gin.Context) {
		res, err := r.Table("schema").Get(c.Param("scope")).Run(session)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error scanning database"})
			return
		}
		defer res.Close()

		var result map[string]interface{}
		if err := res.One(&result); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "model not found"})
			return
		}

		c.JSON(http.StatusOK, result)
	})

	router.POST("/:scope/schema", func(c *gin.Context) {
		var schema map[string]interface{}

		if c.BindJSON(&schema) != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON data"})
			return
		}

		invalids := []string{}
		for k, v := range schema {

			if v == "id" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid field: 'id'"})
				return
			}

			if _, ok := v.([]interface{}); ok {
				continue
			}

			if v == "bool" {
				continue
			}

			if v == "number" {
				continue
			}

			invalids = append(invalids, k)

		}

		if len(invalids) > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid fields", "fields": invalids})
			return
		}

		schema["id"] = c.Param("scope")

		if _, err := r.Table("schema").Insert(schema).RunWrite(session); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "duplicate scope"})
			return
		}

		if _, err := r.TableCreate(schema["id"], r.TableCreateOpts{PrimaryKey: "time"}).RunWrite(dataSession); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "duplicate scope"})
		}

		c.JSON(http.StatusOK, schema)
	})
}

////////

type model struct {
	Factors []string `json:"factors" gorethink:"factors"`
	ID      string   `json:"id" gorethink:"id"`
}

func routeModel(router gin.IRouter, session *r.Session) {

	router.GET("/:scope/model", func(c *gin.Context) {
		res, err := r.Table("model").Get(c.Param("scope")).Run(session)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error scanning database"})
			return
		}
		defer res.Close()

		var result model
		if err := res.One(&result); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "model not found"})
			return
		}

		c.JSON(http.StatusOK, result)
	})

	router.POST("/:scope/model", func(c *gin.Context) {
		var factors []string
		if c.BindJSON(&factors) != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON data"})
			return
		}

		newModel := model{Factors: factors, ID: c.Param("scope")}

		if _, err := r.Table("model").Insert(newModel, r.InsertOpts{Conflict: "replace"}).RunWrite(session); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "storage error"})
			return
		}

		c.JSON(http.StatusOK, newModel)

	})
}
