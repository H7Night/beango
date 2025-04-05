package main

import (
	"beango/core"
	"beango/service"

	"github.com/gin-gonic/gin"
)

func main() {
	core.ConnectDatabase()
	r := gin.Default()
	r.POST("/upload/alipay_csv", service.ImportAlipayCSV)
	r.POST("/upload/wechat_csv", service.ImportWechatCSV)
	r.Run(":10777")
}
