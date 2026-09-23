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

type cliOptions struct {
	sourceType, outputDir string
	merge, passAll        bool
	binanceSync, dryRun   bool
	from, to              string
	symbols               []string
	args                  []string
}

// parseArgs 解析命令行参数，支持 flag 与位置参数任意顺序。
// 标准 flag 包在遇到第一个位置参数后会停止解析，导致
// "beango -type alipay file.csv -output ./out -merge" 中
// -output/-merge 失效，因此这里自行解析。
// 支持 "-flag value" 与 "-flag=value" 两种写法。
func parseArgs(raw []string) (cliOptions, error) {
	var options cliOptions
	value := func(raw []string, i *int, arg string) (string, error) {
		if *i+1 >= len(raw) {
			return "", fmt.Errorf("选项 %s 缺少参数", arg)
		}
		*i++
		return raw[*i], nil
	}
	for i := 0; i < len(raw); i++ {
		arg := raw[i]
		switch {
		case arg == "-h" || arg == "-help" || arg == "--help":
			return cliOptions{}, errHelp
		case arg == "-type" || arg == "--type":
			item, err := value(raw, &i, arg)
			if err != nil {
				return cliOptions{}, err
			}
			options.sourceType = item
		case strings.HasPrefix(arg, "-type="):
			options.sourceType = strings.TrimPrefix(arg, "-type=")
		case arg == "-output" || arg == "--output":
			item, err := value(raw, &i, arg)
			if err != nil {
				return cliOptions{}, err
			}
			options.outputDir = item
		case strings.HasPrefix(arg, "-output="):
			options.outputDir = strings.TrimPrefix(arg, "-output=")
		case arg == "-merge" || arg == "--merge":
			options.merge = true
		case arg == "-p" || arg == "--pass":
			options.passAll = true
		case arg == "-sync" || arg == "--sync":
			options.binanceSync = true
		case arg == "-dry-run" || arg == "--dry-run":
			options.dryRun = true
		case arg == "--from" || arg == "-from":
			item, err := value(raw, &i, arg)
			if err != nil {
				return cliOptions{}, err
			}
			options.from = item
		case arg == "--to" || arg == "-to":
			item, err := value(raw, &i, arg)
			if err != nil {
				return cliOptions{}, err
			}
			options.to = item
		case arg == "--symbols" || arg == "-symbols":
			item, err := value(raw, &i, arg)
			if err != nil {
				return cliOptions{}, err
			}
			for _, symbol := range strings.Split(item, ",") {
				if symbol = strings.TrimSpace(symbol); symbol != "" {
					options.symbols = append(options.symbols, symbol)
				}
			}
		case strings.HasPrefix(arg, "--from="):
			options.from = strings.TrimPrefix(arg, "--from=")
		case strings.HasPrefix(arg, "--to="):
			options.to = strings.TrimPrefix(arg, "--to=")
		case strings.HasPrefix(arg, "--symbols="):
			for _, symbol := range strings.Split(strings.TrimPrefix(arg, "--symbols="), ",") {
				if symbol = strings.TrimSpace(symbol); symbol != "" {
					options.symbols = append(options.symbols, symbol)
				}
			}
		case strings.HasPrefix(arg, "-"):
			return cliOptions{}, fmt.Errorf("未知选项: %s", arg)
		default:
			options.args = append(options.args, arg)
		}
	}
	return options, nil
}

func usage() {
	fmt.Fprintf(os.Stderr, "用法: beango -type <alipay|wechat|binance> [选项] [文件路径]\n")
	fmt.Fprintf(os.Stderr, "选项:\n")
	fmt.Fprintf(os.Stderr, "  -type string\n    \t账单类型: alipay 或 wechat\n")
	fmt.Fprintf(os.Stderr, "  -output string\n    \t输出目录 (默认: ./test/out)\n")
	fmt.Fprintf(os.Stderr, "  -merge\n    \t合并模式：追加到已有 bean 文件\n")
	fmt.Fprintf(os.Stderr, "  -p, --pass\n    \t全量确认：所有条目标记为已确认 (*)\n")
	fmt.Fprintf(os.Stderr, "  -sync\n    \tBinance API 同步模式\n")
	fmt.Fprintf(os.Stderr, "  --from/--to YYYY-MM-DD\n    \tBinance 同步时间范围\n")
	fmt.Fprintf(os.Stderr, "  --symbols BTCUSDT,ETHUSDT\n    \tBinance API 查询交易对\n")
	fmt.Fprintf(os.Stderr, "  --dry-run\n    \tBinance 只预览，不写 bean 文件\n")
}

func main() {
	// CLI 参数（flag 与位置参数可任意顺序）
	options, err := parseArgs(os.Args[1:])
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
	if options.sourceType != "" {
		if options.sourceType == "binance" {
			if err := service.RunBinanceCLI(service.BinanceCLIOptions{Sync: options.binanceSync, DryRun: options.dryRun, From: options.from, To: options.to, Symbols: options.symbols, Input: firstArg(options.args), OutputDir: options.outputDir}); err != nil {
				fmt.Fprintf(os.Stderr, "错误: %v\n", err)
				os.Exit(1)
			}
			return
		}
		// 非 flag 参数：文件路径
		if len(options.args) < 1 {
			usage()
			os.Exit(1)
		}
		filePath := options.args[0]

		if err := service.RunCLI(options.sourceType, filePath, options.outputDir, options.merge, options.passAll); err != nil {
			fmt.Fprintf(os.Stderr, "错误: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// 无参数 → Web 模式
	runWebServer()
}

func firstArg(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return args[0]
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
	webDir := model.WebDir()
	r.NoRoute(func(c *gin.Context) {
		requestedPath := c.Request.URL.Path
		filepath := path.Join(webDir, requestedPath)
		if _, err := os.Stat(filepath); err == nil {
			c.File(filepath)
			return
		}
		c.File(path.Join(webDir, "index.html"))
	})

	if err := r.Run(":" + model.ServerPort()); err != nil {
		panic(err)
	}
}
