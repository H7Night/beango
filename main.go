package main

import (
	"beango/core"
	"beango/routes"
	"github.com/gin-gonic/gin"
)

func main() {
	core.ConnectDatabase()
	r := gin.Default()
	db := core.GetDB()

	routes.RegisterAccountMappingRoutes(r, db)
	routes.RegisteImportRoutes(r)
	r.Run(":10777")
}
