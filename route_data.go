package main

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	r "gopkg.in/gorethink/gorethink.v3"
)

// GetRecent retrieves the most recent scope entry before a specific time.
func GetRecent(scope string, time time.Time, session *r.Session) (State, error) {
	res, err := r.Table("data").
		GetAllByIndex("scope", scope).
		Filter(r.Row.Field(primaryKey).Lt(time)).
		Max(primaryKey).Run(session)
	if err != nil {
		return nil, err
	}
	defer res.Close()

	var data State
	if err := res.One(&data); err != nil {
		return nil, err
	}

	return data, nil
}

// DataPoint is a single data point, containing a primary label and values for its model factors.
type DataPoint struct {
	Label   State            `json:"label"`
	Factors map[string]State `json:"factors"`
	Time    time.Time        `json:"time"`
}

func routeData(router gin.IRouter, session *r.Session) {

	router.GET("/:scope/data", func(c *gin.Context) {
		scope := c.Param("scope")

		model, err := GetModel(scope, session)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "model not registered"})
			return
		}

		states, err := GetStates(scope, session)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error retrieving scope states"})
			return
		}

		dataSet := make([]DataPoint, len(states))
		for index, state := range states {
			dataSet[index] = DataPoint{
				Label:   state,
				Factors: make(map[string]State),
				Time:    state[primaryKey].(time.Time),
			}
		}

		for _, dataPoint := range dataSet {
			for _, factorScope := range model.Factors {
				factorState, err := GetRecent(factorScope, dataPoint.Label[primaryKey].(time.Time), session)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "error retrieving truncated factor data"})
					return
				}
				delete(factorState, primaryKey)
				dataPoint.Factors[factorScope] = factorState
			}
			delete(dataPoint.Label, primaryKey)
		}

		c.JSON(http.StatusOK, dataSet)

	})

}
