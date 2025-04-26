package main

import (
	"beango/middleware"
	"beango/model"
	"beango/routes"
	"github.com/gin-gonic/gin"
)

func main() {
	model.ConnectDatabase()
	r := gin.Default()
	r.Use(middleware.CorsMiddleware())

	routes.RegisterAccountMapRoutes(r)
	routes.RegisteImportRoutes(r)
	routes.RegisterBeangoConfig(r)
	r.Run(":10777")
}
