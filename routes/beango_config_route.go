package routes

import (
	"beango/model"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func RegisterBeangoConfig(router *gin.Engine) {
	group := router.Group("/beango_config")
	group.GET("", func(c *gin.Context) {
		configs, err := model.GetAllBeangoConfig()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message": "success",
			"data":    configs,
		})
	})
	group.POST("/create", func(c *gin.Context) {
		var beangoConfig model.BeangoConfig
		if err := c.ShouldBindJSON(&beangoConfig); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}
		if err := model.CreateBeangoConfig(beangoConfig); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message": "create successfully",
			"data":    beangoConfig,
		})
	})
	group.PUT("/update/:id", func(c *gin.Context) {
		var beangoConfig model.BeangoConfig
		idStr := c.Param("id")
		if idStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "id is required",
			})
			return
		}
		id, err := strconv.ParseUint(idStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}
		if err := c.ShouldBindJSON(&beangoConfig); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}
		if err := model.UpdateBeangoConfig(id, beangoConfig); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message": "update success",
			"data":    beangoConfig,
		})
	})
	group.DELETE("/delete/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		if idStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "id is required",
			})
			return
		}
		id, err := strconv.ParseUint(idStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}
		if err := model.DeleteBeangoConfig(id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message": "delete success",
			"data":    id,
		})
	})
}
