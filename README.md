# Beango

将支付宝/微信交易账单转换为 [Beancount](https://beancount.github.io/) 格式的纯 Go 工具。支持 **CLI 命令行** 和 **Web 界面** 两种使用方式。

## 快速开始

### CLI 模式

```bash
# 转换支付宝账单
beango -type alipay alipay_record.csv

# 转换微信账单
beango -type wechat wechat_record.xlsx

# 指定输出目录
beango -type alipay alipay_record.csv -output ./my-bean

# 合并模式（追加到已有 bean 文件）
beango -type wechat wechat_record.xlsx -merge
```

### Web 模式

```bash
# 直接运行，启动 Web 服务器（默认端口 10777）
beango
```

浏览器打开 `http://localhost:10777`，上传账单文件即可转换。

## 配置文件

所有配置均为 YAML 文件，存放在 `config/` 目录下：

| 文件                       | 用途                                                |
| -------------------------- | --------------------------------------------------- |
| `config/account_map.yml`   | 关键词 → Beancount 账户映射（交易分类、支付方式等） |
| `config/beango.yml`        | 全局设置（输出目录等）                              |
| `config/commodity_map.yml` | 商品关键词 → 交易类型映射（支出/收入/跳过）         |

### account_map.yml

通过关键词将交易匹配到对应的 Beancount 账户。支持三种映射类型：

- `asset`：资产/负债账户（如银行卡、支付宝余额、花呗）
- `expense`：支出分类
- `income`：收入分类

可在 Web 界面中在线管理映射，或直接编辑 YAML 文件。

### beango.yml

```yaml
beango:
  outputFolder: "./test/out" # 输出根目录（转换输出、中间文件与日志均在此）
  defaultFolder: "0-default" # 普通交易子目录
  securitFolder: "1-securities" # 证券交易子目录
```

## 开发

### 环境要求

- Go 1.24+
- Node.js 22+（前端开发）
- pnpm（前端包管理）

### 构建

```powershell
# 完整构建（前端 + 后端）
.\scripts\build.ps1

# 仅构建后端
.\scripts\build.ps1 -BackendOnly

# 仅构建前端
.\scripts\build.ps1 -FrontendOnly
```

### 开发模式

```powershell
# 同时启动前后端开发服务器
.\scripts\dev.ps1
```

### 测试

```powershell
.\scripts\test.ps1
```

### Docker

```bash
# 构建镜像
docker build -t beango-local .

# 启动
docker-compose up -d
```

## 项目结构

```
beango/
├── config/            # YAML 配置文件
│   ├── account_map.yml
│   ├── beango.yml
│   └── commodity_map.yml
├── model/             # 数据模型 & 配置读写
├── service/           # 业务逻辑（转换管线 + CLI）
├── routes/            # Web API 路由
├── middleware/         # Gin 中间件
├── utils/             # 工具函数
├── beango-web/        # Vue.js 前端
├── scripts/           # 构建/开发/测试脚本
│   └── sql/           # 数据库初始化脚本 (init.sql)
├── test/              # 测试数据与输出 (test/out 为输出根目录)
├── main.go            # 入口（CLI/Web 模式分发）
├── Dockerfile
└── docker-compose.yaml
```

## 转换流程

```
支付宝 CSV (GBK) ──→ UTF8 ──→ CSV 解析 ──→ TransAlipay ──→ .bean 文件
微信 Excel       ──→ 清洗 ──→ 行解析   ──→ TransWechat ──→ .bean 文件
```

转换后的 `.bean` 文件输出到 `outputFolder` 配置目录（默认 `./test/out`），按 `年份/月份/` 目录结构组织，可直接被 Beancount 主文件 `include`。
