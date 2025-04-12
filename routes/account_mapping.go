package routes

import (
	"beango/model"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net/http"
	"strconv"
)

func RegisterAccountMappingRoutes(router *gin.Engine, db *gorm.DB) {
	group := router.Group("/account_mappings")
	group.GET("", func(c *gin.Context) {
		mappings, err := model.GetAllAccountMapping(db)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, mappings)
	})
	group.POST("/create", func(c *gin.Context) {
		var body model.AccountMapping
		if err := c.ShouldBind(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}
		if err := model.CreateAccountMapping(db, body.Keyword, body.Account); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "success create"})
	})
	group.PUT("/update", func(c *gin.Context) {
		idStr := c.Query("id")
		if idStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Missing id parameter"})
			return
		}
		id, err := strconv.ParseUint(idStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}
		var body model.AccountMapping
		if err := c.ShouldBind(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return

		}
		if err := model.UpdateAccountMapping(db, uint(id), body.Keyword, body.Account); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "success update"})
	})
	group.DELETE("/delete", func(c *gin.Context) {
		idStr := c.Query("id")
		if idStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Missing id parameter"})
			return
		}
		id, err := strconv.ParseUint(idStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}
		if err := model.DeleteAccountMapping(db, uint(id)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "success delete"})
	})
}
