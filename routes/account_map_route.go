package routes

import (
	"beango/model"
	"net/http"

	"github.com/gin-gonic/gin"
)

func RegisterAccountMapRoutes(router *gin.Engine) {
	group := router.Group("/account_map")
	group.GET("", func(c *gin.Context) {
		maps, err := model.GetAllAccountMap()
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"error": err,
			})
			return
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
	group.PUT("/update/:keyword", func(c *gin.Context) {
		keyword := c.Param("keyword")
		if keyword == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Missing keyword parameter"})
			return
		}
		var accountMap model.AccountMap
		if err := c.ShouldBind(&accountMap); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}
		if err := model.UpdateAccountMap(keyword, accountMap); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "success update", "data": accountMap})
	})
	group.DELETE("/delete/:keyword", func(c *gin.Context) {
		keyword := c.Param("keyword")
		if keyword == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Missing keyword parameter"})
			return
		}
		if err := model.DeleteAccountMap(keyword); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "success delete", "data": keyword})
	})
}
