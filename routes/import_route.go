package routes

import (
	"beango/service"

	"github.com/gin-gonic/gin"
)

func RegisteImportRoutes(r *gin.Engine) {
	r.POST("/upload/alipay_csv", service.ImportAlipayCSV)
	r.POST("/upload/wechat_csv", service.ImportWechatCSV)
	// 新增
	r.GET("/api/files/tree", service.GetFileTree)
	r.GET("/api/files/content", service.GetFileContent)
}
