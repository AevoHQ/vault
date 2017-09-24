package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	r "gopkg.in/gorethink/gorethink.v3"
)

// Model is a database model for a specific scope.
// ID is the scope and Factors is a list of scopes which ID depends on.
type Model struct {
	Factors []string `json:"factors" gorethink:"factors"`
	ID      string   `json:"id" gorethink:"id"`
}

// GetModel retrieves the model for a given scope.
func GetModel(scope string, session *r.Session) (Model, error) {
	res, err := r.Table("model").Get(scope).Run(session)
	if err != nil {
		return Model{}, err
	}
	defer res.Close()

	var result Model
	if err := res.One(&result); err != nil {
		return Model{}, err
	}

	return result, nil
}

func routeModel(router gin.IRouter, session *r.Session) {

	router.GET("/:scope/model", func(c *gin.Context) {
		model, err := GetModel(c.Param("scope"), session)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "model not found"})
			return
		}

		c.JSON(http.StatusOK, model)
	})

	router.POST("/:scope/model", func(c *gin.Context) {
		var factors []string
		if c.BindJSON(&factors) != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON data"})
			return
		}

		model := Model{Factors: factors, ID: c.Param("scope")}

		if _, err := r.Table("model").Insert(model, r.InsertOpts{Conflict: "replace"}).RunWrite(session); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "storage error"})
			return
		}

		c.JSON(http.StatusOK, model)

	})
}
