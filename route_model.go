package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	r "gopkg.in/gorethink/gorethink.v3"
)

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
