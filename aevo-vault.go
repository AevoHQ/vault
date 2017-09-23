package main

import (
	"log"
	"time"

	"net/http"

	"github.com/gin-gonic/gin"
	r "gopkg.in/gorethink/gorethink.v3"
)

func main() {
	route()
}

func route() {

	router := gin.Default()
	context := router.Group("/context")

	dataSession, err := r.Connect(r.ConnectOpts{
		Address:  "localhost:28015",
		Database: "test",
	})
	if err != nil {
		log.Fatalln(err)
	}
	routeState(context, dataSession)

	modelSession, err := r.Connect(r.ConnectOpts{
		Address:  "localhost:28015",
		Database: "model",
	})
	if err != nil {
		log.Fatalln(err)
	}
	routeContext(context, dataSession, modelSession)
	routeModel(context, modelSession)

	router.Run(":3000")

}

const primaryKey = "time"

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
	router.GET("/:context", func(c *gin.Context) {
		res, err := r.Table(c.Param("context")).OrderBy(r.OrderByOpts{
			Index: primaryKey,
		}).Run(session)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "context not registered"})
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

	router.POST("/:context", func(c *gin.Context) {
		storeState := func(data dataInput) {
			if c.BindJSON(&data) != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON data"})
				return
			}

			data.parseMapTime()

			_, err := r.Table(c.Param("context")).Insert(data, r.InsertOpts{Conflict: "replace"}).RunWrite(session)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "context not registered"})
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

func routeContext(router gin.IRouter, dataSession *r.Session, session *r.Session) {
	router.GET("/:context/context", func(c *gin.Context) {
		res, err := r.Table("context").Get(c.Param("context")).Run(session)
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

	router.POST("/:context/context", func(c *gin.Context) {
		var context map[string]interface{}

		if c.BindJSON(&context) != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON data"})
			return
		}

		invalids := []string{}
		for k, v := range context {

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

		context["id"] = c.Param("context")

		if _, err := r.Table("context").Insert(context).RunWrite(session); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "duplicate context"})
			return
		}

		if _, err := r.TableCreate(context["id"], r.TableCreateOpts{PrimaryKey: "time"}).RunWrite(dataSession); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "duplicate context"})
		}

		c.JSON(http.StatusOK, context)
	})
}

////////

type model struct {
	Factors []string `json:"factors" gorethink:"factors"`
	ID      string   `json:"id" gorethink:"id"`
}

func routeModel(router gin.IRouter, session *r.Session) {

	router.GET("/:context/model", func(c *gin.Context) {
		res, err := r.Table("model").Get(c.Param("context")).Run(session)
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

	router.POST("/:context/model", func(c *gin.Context) {
		var factors []string
		if c.BindJSON(&factors) != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON data"})
			return
		}

		newModel := model{Factors: factors, ID: c.Param("context")}

		if _, err := r.Table("model").Insert(newModel, r.InsertOpts{Conflict: "replace"}).RunWrite(session); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "storage error"})
			return
		}

		c.JSON(http.StatusOK, newModel)

	})
}
