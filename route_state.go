package main

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	r "gopkg.in/gorethink/gorethink.v3"
)

var primaryKey = "time"

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

func getStates(scope string, session *r.Session, c *gin.Context) ([]state, error) {
	res, err := r.Table(c.Param("scope")).OrderBy(r.OrderByOpts{
		Index: primaryKey,
	}).Run(session)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "scope not registered"})
		return nil, err
	}
	defer res.Close()

	var data []state
	if err := res.All(&data); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error scanning database result"})
		return nil, err
	}

	return data, nil
}

func routeState(router gin.IRouter, session *r.Session) {
	router.GET("/:scope", func(c *gin.Context) {
		data, err := getStates(c.Param("scope"), session, c)
		if err != nil {
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
