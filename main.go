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
	db := model.GetDB()

	routes.RegisterAccountMappingRoutes(r, db)
	routes.RegisteImportRoutes(r)
	r.Run(":10777")
}
