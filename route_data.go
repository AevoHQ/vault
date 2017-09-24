package main

import (
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	r "gopkg.in/gorethink/gorethink.v3"
)

func getTruncated(scope string, time time.Time, session *r.Session) (state, error) {
	res, err := r.Table(scope).
		Filter(r.Row.Field(primaryKey).Lt(time)).
		Max(primaryKey).Run(session)
	if err != nil {
		return nil, err
	}
	defer res.Close()

	var data state
	if err := res.One(&data); err != nil {
		return nil, err
	}

	return data, nil
}

func routeData(router gin.IRouter, dataSession *r.Session, modelSession *r.Session) {

	router.GET("/:scope/data", func(c *gin.Context) {
		scope := c.Param("scope")

		model, err := getModel(scope, modelSession)
		if err != nil {
			return
		}

		states, err := getStates(scope, dataSession, c)
		if err != nil {
			return
		}

		result := make([][]float64, len(states))

		for stateIndex, state := range states {
			resultVectors := make(map[string][]float64)

			for _, factorScope := range model.Factors {

				factorSchema, _ := getSchema(factorScope, modelSession)
				value, err := getTruncated(factorScope, state[primaryKey].(time.Time), dataSession)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "error retrieving truncated factor data"})
					return
				}

				for fieldName, fieldType := range factorSchema {

					if fieldName == "id" {
						continue
					}

					if fieldType == "number" {
						resultVectors[fieldName] = []float64{value[fieldName].(float64)}
						continue
					}

					if xs, ok := fieldType.([]interface{}); ok {
						vector := make([]float64, len(xs))
						for i, x := range xs {
							if value[fieldName] == x {
								vector[i] = 1
								break
							}
						}
						fmt.Println(fieldName, vector)
						resultVectors[fieldName] = vector
						continue
					}
				}

			}

			sortedKeys := make([]string, len(resultVectors))
			i := 0
			for k := range resultVectors {
				sortedKeys[i] = k
				i++
			}
			sort.Strings(sortedKeys)

			var things []float64
			for _, key := range sortedKeys {
				things = append(things, resultVectors[key]...)
			}
			fmt.Println(things)
			result[stateIndex] = things
		}

		c.JSON(http.StatusOK, result)

	})

}
