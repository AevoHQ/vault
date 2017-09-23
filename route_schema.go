package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	r "gopkg.in/gorethink/gorethink.v3"
)

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
