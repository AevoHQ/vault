package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	r "gopkg.in/gorethink/gorethink.v3"
)

// Schema is a schema for scopes, defining what type each field should be.
type Schema map[string]interface{}

// GetSchema retrieves the schema for a given scope.
func GetSchema(scope string, session *r.Session) (Schema, error) {
	res, err := r.Table("schema").Get(scope).Run(session)
	if err != nil {
		return nil, err
	}
	defer res.Close()

	var result Schema
	if err := res.One(&result); err != nil {
		return nil, err
	}

	return result, nil
}

func routeSchema(router gin.IRouter, dataSession *r.Session, session *r.Session) {
	router.GET("/:scope/schema", func(c *gin.Context) {
		result, err := GetSchema(c.Param("scope"), session)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "schema not found"})
			return
		}

		c.JSON(http.StatusOK, result)
	})

	router.POST("/:scope/schema", func(c *gin.Context) {
		var schema Schema

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

			if xs, ok := v.([]interface{}); ok {
				strings := make([]string, len(xs))
				success := true
				for _, v := range xs {
					if strVal, okStr := v.(string); okStr {
						strings = append(strings, strVal)
					} else {
						success = false
					}
				}
				if success {
					continue
				}
			}

			if v == "bool" {
				schema[k] = []bool{false, true}
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

		res, err := r.Table("schema").Insert(schema, r.InsertOpts{Conflict: "replace"}).RunWrite(session)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "storage error"})
			return
		}

		if res.Inserted != 0 {
			if _, err := r.TableCreate(schema["id"], r.TableCreateOpts{PrimaryKey: "time"}).RunWrite(dataSession); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "duplicate schema"})
				return
			}
		}

		c.JSON(http.StatusOK, schema)
	})
}
