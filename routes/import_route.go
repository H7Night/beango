package routes

import (
	"beango/service"

	"github.com/gin-gonic/gin"
)

func RegisteImportRoutes(r *gin.Engine) {
	r.POST("/import", service.ImportCSV)
	r.POST("/upload/alipay_csv", service.ImportAlipayCSV)
	r.POST("/upload/wechat_csv", service.ImportWechatCSV)
	r.POST("/upload/alipay_zip", service.ImportAlipayZip)
}
