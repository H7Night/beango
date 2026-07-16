package main

import (
	"beango/middleware"
	"beango/model"
	"beango/routes"
	"beango/service"
	"beango/utils"
	"flag"
	"fmt"
	"os"
	"path"

	"github.com/gin-gonic/gin"
)

func main() {
	// CLI 参数
	sourceType := flag.String("type", "", "账单类型: alipay 或 wechat")
	outputDir := flag.String("output", "", "输出目录 (默认: ./out)")
	merge := flag.Bool("merge", false, "合并模式：追加到已有 bean 文件")

	flag.Parse()

	// 如果指定了 -type，走 CLI 模式
	if *sourceType != "" {
		// 非 flag 参数：文件路径
		args := flag.Args()
		if len(args) < 1 {
			fmt.Fprintf(os.Stderr, "用法: beango -type <alipay|wechat> [选项] <文件路径>\n")
			fmt.Fprintf(os.Stderr, "选项:\n")
			flag.PrintDefaults()
			os.Exit(1)
		}
		filePath := args[0]

		if err := service.RunCLI(*sourceType, filePath, *outputDir, *merge); err != nil {
			fmt.Fprintf(os.Stderr, "错误: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// 无参数 → Web 模式
	runWebServer()
}

func runWebServer() {
	err := utils.InitLogging()
	if err != nil {
		panic(err)
	}
	defer utils.CloseLogging()

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

	// 预加载 account_map
	_ = model.LoadAccountMap()

	// 注册路由
	routes.RegisterAccountMapRoutes(r)
	routes.RegisteImportRoutes(r)
	routes.RegisterBeangoConfig(r)

	// Serve static files for the frontend and handle SPA fallback
	r.NoRoute(func(c *gin.Context) {
		requestedPath := c.Request.URL.Path
		filepath := path.Join("web/dist", requestedPath)
		if _, err := os.Stat(filepath); err == nil {
			c.File(filepath)
			return
		}
		c.File("web/dist/index.html")
	})

	if err := r.Run(":10777"); err != nil {
		panic(err)
	}
}
