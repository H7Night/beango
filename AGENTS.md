# AGENTS.md — beango

支付宝/微信交易账单 → [Beancount](https://beancount.github.io/) 格式的 Go 工具（CLI + Web）。

## Project
- Go 1.24.2（module `beango`），入口 `main.go`：CLI 转换（flag 与位置参数任意顺序）或 Web 服务（gin，默认端口 10777）
- 配置（YAML，`config/`）：`beango.yml`（输出目录/端口/兜底账户）、`account_map.yml`（关键词→Beancount 账户）、`commodity_map.yml`（商品→支出/收入/跳过）
- 输出：`test/out/<运行日期>/<年份>/<0-default|1-securities>/<月>.bean`（按年月分组，交易按时序倒序写）
- 账本本体在独立仓库 `../beancount`（本目录 `beancount` 为符号链接），`main.bean` 按年份 include；校验用其 `.venv` 里的 beancount

## Commands
- 构建: `go build -o bin/beango.exe .`
- 测试: `go test ./...`
- CLI: `bin/beango.exe -type <alipay|wechat> <文件> [-output DIR] [-merge]`
  - 支付宝 CSV 为 **GBK** 编码（`utils/ConvertGBKtoUTF8withBom` 自动转）；微信为 xlsx
  - 非 `-merge` 会**删除并重建**当日输出目录，故多文件转换需先转第一个（非 merge），其余用 `-merge` 追加
- 账本校验: `cd ../beancount && .venv/Scripts/python.exe -m beancount.scripts.check main.bean`
- 脚本（`scripts/`，python）：`sort_bean_files.py` 账本按时序正序排序；`check_bean_order.py` 校验排序；`check_bean_duplicates.py` 检查重复 uuid；`account_balance.py` 查询账户时点余额（含 pad）；`reconcile*.py`/`match_*.py` 对账

## Architecture
- `service/cli_service.go` / `import_service.go`：文件解析入口（GBK 转换、CSV/xlsx 清洗）
- `service/transaction_alipay_service.go` / `transaction_wechat_service.go`：记录→beancount 条目，按「支付方式→对方→商品/分类」顺序匹配账户；`TransAlipay` 校验表头（首行含"交易时间"）+≥1 条记录
- `service/export_service.go`：`TransToBeancount` 按年月分组写 .bean，`-merge` 追加+排序；收益发放（`xxx-收益发放`）分到 `1-securities`
- `model/`：配置加载（account_map / commodity_map / beango.yml 各路径与兜底账户）
- `utils/`：GBK→UTF8、日志、输出目录初始化

## Conventions
- 提交信息：`fix(scope): 中文描述` / `feat(scope): 中文描述`（scope 如 alipay/account_map/scripts）；一个 bug 或一个 feat 单独一个提交，不合并
- 新增商户：先在 `config/account_map.yml` 加关键词映射（`account` + `type: expense|income|asset`），避免落兜底账户（`Expenses:Other` 等）
- 收益发放/基金类收入账户用账本中实际开户的 `Income:Funds`（`Income:Fund` 不存在）
- beancount 账本约定：文件按日期+时分秒**正序**；`balance` 断言检查的是**当日交易之前**（即前一日末）的余额，表达"某日还清后为 0"需把断言日期放到**次日**
- 合并账单去重：以 `uuid` 为准（脚本 `check_bean_duplicates.py`），手工补记可无 uuid

## Notes
（留空，后续按需补充）
