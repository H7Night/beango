package service

import (
	"beango/model"
	"beango/utils"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func GetFileTree(c *gin.Context) {
	outputFolder := model.GetConfigString("outputFolder", "./output")
	tree, err := utils.BuildFileTree(outputFolder)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取目录树失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"fileTree": tree})
}

func GetFileContent(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少参数 path"})
		return
	}

	data, err := os.ReadFile(path)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取文件失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"content": string(data),
	})
}
