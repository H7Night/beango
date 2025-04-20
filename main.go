package main

import (
	"beango/core"
	"beango/middleware"
	"beango/routes"
	"github.com/gin-gonic/gin"
)

func main() {
	core.ConnectDatabase()
	r := gin.Default()
	r.Use(middleware.CorsMiddleware())
	db := core.GetDB()

	routes.RegisterAccountMappingRoutes(r, db)
	routes.RegisteImportRoutes(r)
	r.Run(":10777")
}
