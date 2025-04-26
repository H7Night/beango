package routes

import (
	"beango/model"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
)

func RegisterAccountMapRoutes(router *gin.Engine) {
	group := router.Group("/account_map")
	group.GET("", func(c *gin.Context) {
		maps, err := model.GetAllAccountMap()
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"error": err,
			})
		}
		c.JSON(http.StatusOK, gin.H{"data": maps})
	})
	group.POST("/create", func(c *gin.Context) {
		var accountMap model.AccountMap
		if err := c.ShouldBind(&accountMap); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}
		if err := model.CreateAccountMap(accountMap); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "success create", "data": accountMap})
	})
	group.PUT("/update/:id", func(c *gin.Context) {
		idStr := c.Param("id")
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
		var accountMap model.AccountMap
		if err := c.ShouldBind(&accountMap); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return

		}
		if err := model.UpdateAccountMap(id, accountMap); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "success update", "data": accountMap})
	})
	group.DELETE("/delete/:id", func(c *gin.Context) {
		idStr := c.Param("id")
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
		if err := model.DeleteAccountMap(id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "success delete", "data": id})
	})
}
