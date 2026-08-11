package main

import (
	"beango/middleware"
	"beango/model"
	"beango/routes"
	"beango/service"
	"beango/utils"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/gin-gonic/gin"
)

// errHelp 表示用户请求了帮助（-h/-help）
var errHelp = errors.New("help requested")

// parseArgs 解析命令行参数，支持 flag 与位置参数任意顺序。
// 标准 flag 包在遇到第一个位置参数后会停止解析，导致
// "beango -type alipay file.csv -output ./out -merge" 中
// -output/-merge 失效，因此这里自行解析。
// 支持 "-flag value" 与 "-flag=value" 两种写法。
func parseArgs(raw []string) (sourceType, outputDir string, merge bool, args []string, err error) {
	for i := 0; i < len(raw); i++ {
		arg := raw[i]
		switch {
		case arg == "-h" || arg == "-help" || arg == "--help":
			return "", "", false, nil, errHelp
		case arg == "-type" || arg == "--type":
			if i+1 >= len(raw) {
				return "", "", false, nil, fmt.Errorf("选项 %s 缺少参数", arg)
			}
			i++
			sourceType = raw[i]
		case strings.HasPrefix(arg, "-type="):
			sourceType = strings.TrimPrefix(arg, "-type=")
		case arg == "-output" || arg == "--output":
			if i+1 >= len(raw) {
				return "", "", false, nil, fmt.Errorf("选项 %s 缺少参数", arg)
			}
			i++
			outputDir = raw[i]
		case strings.HasPrefix(arg, "-output="):
			outputDir = strings.TrimPrefix(arg, "-output=")
		case arg == "-merge" || arg == "--merge":
			merge = true
		case strings.HasPrefix(arg, "-"):
			return "", "", false, nil, fmt.Errorf("未知选项: %s", arg)
		default:
			args = append(args, arg)
		}
	}
	return sourceType, outputDir, merge, args, nil
}

func usage() {
	fmt.Fprintf(os.Stderr, "用法: beango -type <alipay|wechat> [选项] <文件路径>\n")
	fmt.Fprintf(os.Stderr, "选项:\n")
	fmt.Fprintf(os.Stderr, "  -type string\n    \t账单类型: alipay 或 wechat\n")
	fmt.Fprintf(os.Stderr, "  -output string\n    \t输出目录 (默认: ./test/out)\n")
	fmt.Fprintf(os.Stderr, "  -merge\n    \t合并模式：追加到已有 bean 文件\n")
}

func main() {
	// CLI 参数（flag 与位置参数可任意顺序）
	sourceType, outputDir, merge, args, err := parseArgs(os.Args[1:])
	if err != nil {
		if errors.Is(err, errHelp) {
			usage()
			os.Exit(2)
		}
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		usage()
		os.Exit(2)
	}

	// 如果指定了 -type，走 CLI 模式
	if sourceType != "" {
		// 非 flag 参数：文件路径
		if len(args) < 1 {
			usage()
			os.Exit(1)
		}
		filePath := args[0]

		if err := service.RunCLI(sourceType, filePath, outputDir, merge); err != nil {
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
