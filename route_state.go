package main

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	r "gopkg.in/gorethink/gorethink.v3"
)

const primaryKey = "time"

// State is a single state of a scope.
type State map[string]interface{}

// States is a list of `State`s.
type States []State

// Statelike is an interface for types that may be interpreted and stored as states.
type Statelike interface {
	parseMapTime()
	setScope(scope string)
}

func (state State) parseMapTime() {
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
func (state State) setScope(scope string) {
	state["scope"] = scope
}

func (states States) parseMapTime() {
	for _, state := range states {
		state.parseMapTime()
	}
}

func (states States) setScope(scope string) {
	for _, state := range states {
		state.setScope(scope)
	}
}

// GetStates retrieves all stored states for a given scope in chronological order.
func GetStates(scope string, session *r.Session) (States, error) {
	res, err := r.Table("data").GetAllByIndex("scope", scope).
		OrderBy(primaryKey).Run(session)
	if err != nil {
		return nil, err
	}
	defer res.Close()

	var data States
	if err := res.All(&data); err != nil {
		return nil, err
	}

	return data, nil
}

func routeState(router gin.IRouter, session *r.Session) {
	router.GET("/:scope", func(c *gin.Context) {
		data, err := GetStates(c.Param("scope"), session)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "error retrieving states"})
			return
		}
		c.JSON(http.StatusOK, data)
	})

	router.POST("/:scope", func(c *gin.Context) {
		storeState := func(data Statelike) {
			if c.BindJSON(&data) != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON data"})
				return
			}

			data.parseMapTime()

			data.setScope(c.Param("scope"))
			_, err := r.Table("data").Insert(data, r.InsertOpts{Conflict: "replace"}).RunWrite(session)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "scope not registered"})
				return
			}

			c.JSON(http.StatusOK, data)
		}

		multi := c.DefaultQuery("multi", "false")
		if multi != "true" {
			var data State
			storeState(&data)
		} else {
			var data States
			storeState(&data)
		}
	})
}
