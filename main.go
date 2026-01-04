package main

import (
	"beango/middleware"
	"beango/model"
	"beango/routes"
	"beango/utils"
	"fmt"

	"github.com/gin-gonic/gin"
)

func main() {
	f, err := utils.InitLogging()
	if err != nil {
		panic(err)
	}
	defer f.Close()

	model.ConnectDatabase()
	gin.SetMode(gin.DebugMode)
	r := gin.Default()
	r.Use(middleware.CorsMiddleware())
	r.Use(middleware.ResponseLoggingMiddleware())

	r.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("[%s] %s %s %d %s \"%s\" %s\n",
			param.TimeStamp.Format("2006-01-02 15:04:05"),
			param.ClientIP,
			param.Method,
			param.StatusCode,
			param.Path,
			param.Request.UserAgent(),
			param.Latency,
		)
	}))
	r.GET("/error", func(c *gin.Context) {
		c.JSON(500, gin.H{"message": "error"})
	})
	// 注册路由
	routes.RegisterAccountMapRoutes(r)
	routes.RegisteImportRoutes(r)
	routes.RegisterBeangoConfig(r)
	if err := r.Run(":10777"); err != nil {
		panic(err)
	}
}
