# -*- coding: utf-8 -*-
"""核对修复: 按单号逐笔核对后修正 beancount 账本 Assets:WeChat 记录"""
import sys, shutil, io
sys.stdout.reconfigure(encoding='utf-8')

BASE = r'C:\Users\user\Abandon\Projects\beango\beancount'

def read(p):
    with io.open(p, encoding='utf-8') as f:
        return f.read()

def write(p, s):
    with io.open(p, 'w', encoding='utf-8', newline='') as f:
        f.write(s)

def apply(path, old, new):
    full = BASE + '\\' + path
    s = read(full)
    n = s.count(old)
    if n != 1:
        print(f"SKIP {path}: 期望 1 处匹配, 实际 {n} 处")
        return False
    shutil.copy2(full, full + '.bak')
    write(full, s.replace(old, new))
    print(f"OK   {path}")
    return True

# 1. amom 转账收入 1000 记成 0.00 -> 修正
apply(r'2026\0-default\02.bean',
"""    Income:RedEnvelope                     -0.00 CNY
    Assets:WeChat                           0.00 CNY""",
"""    Income:RedEnvelope                   -1000.00 CNY
    Assets:WeChat                         1000.00 CNY""")

# 2. 删除 2026-05-17 停简单虚假记录
apply(r'2026\0-default\05.bean',
"""2026-05-17 * "停简单" "停车费用（支付）-粤ETH163"
    time: "20:37:59"
    uuid: "4200002891202510315594739019"
    status: "支付成功"
    Expenses:Transport:Car:Parking          9.00 CNY
    Assets:WeChat                          -9.00 CNY

""",
"")

# 3. 删除 2026-05-20 已全额退款支出 TK2026052066394
apply(r'2026\0-default\05.bean',
"""2026-05-20 * "购票支付" "TK2026052066394,2026-05-25 07:02 珠江村镇银行公交站（顺源名车专修店）-三元里地铁A2口停靠点"
    time: "23:13:46"
    uuid: "4200003094202605209828381166"
    status: "支付成功"
    Expenses:Transport:Bus                 15.00 CNY
    Assets:WeChat                         -15.00 CNY

""",
"")

# 4a. 补记 2026-05-28 部分退款支出(已退款 12)
apply(r'2026\0-default\05.bean',
"""    Expenses:Transport:Bus                 15.00 CNY
    Assets:WeChat                         -15.00 CNY

2026-05-28 * "芬记美食" "芬记小厨\"""",
"""    Expenses:Transport:Bus                 15.00 CNY
    Assets:WeChat                         -15.00 CNY

2026-05-28 * "购票支付" "TK2026052835974,2026-05-29 18:00 三元里地铁A2口停靠点-三水广场公交站（必胜客餐厅）"
    time: "14:07:23"
    uuid: "4200003068202605287055056695"
    status: "WeChat - 已退款(￥12.00)"
    Expenses:Transport:Bus                 15.00 CNY
    Assets:WeChat                         -15.00 CNY

2026-05-28 * "芬记美食" "芬记小厨\"""")

# 4b. 补记 2026-05-29 退款 12
apply(r'2026\0-default\05.bean',
"""    Expenses:Transport:Bus                 15.00 CNY
    Assets:WeChat                         -15.00 CNY

2026-05-29 * "蚂蚁财富-蚂蚁（杭州）基金销售有限公司" "蚂蚁财富-广发纳斯达克100ETF联接(QDII)A-买入\"""",
"""    Expenses:Transport:Bus                 15.00 CNY
    Assets:WeChat                         -15.00 CNY

2026-05-29 * "购票支付" "购票支付"
    time: "17:34:37"
    uuid: "50103307462026052977486271498"
    status: "WeChat - 已退款￥12.00"
    Income:Rebate                         -12.00 CNY
    Assets:WeChat                          12.00 CNY

2026-05-29 * "蚂蚁财富-蚂蚁（杭州）基金销售有限公司" "蚂蚁财富-广发纳斯达克100ETF联接(QDII)A-买入\"""")

# 5. 删除 2025-12-31 差值修补
apply(r'2025\0-default\12.bean',
"""2025-12-31 * "微信" "差值修补"
    Expenses:Other                      0.01 CNY
    Assets:WeChat                       -0.01 CNY

""",
"")

# 6. 2025-11-10 提现 7.6 补记手续费 0.01
apply(r'2025\0-default\11.bean',
"""    Assets:WeChat                      -7.59 CNY
    Assets:CEB:1027                     7.59 CNY""",
"""    Assets:WeChat                      -7.60 CNY
    Assets:CEB:1027                     7.59 CNY
    Expenses:Invest:Cost                 0.01 CNY""")

# 7. 删除 2025-03-27 何仲贤重复记录
apply(r'2025\0-default\03.bean',
"""2025-03-27 * "何仲贤 10370851273" "网上支付-银联无卡 财付通 微信支付"
    time: "00:00:00"
    Assets:CEB:1027                        -1.00 CNY
    Assets:WeChat                           1.00 CNY

2025-03-27 * "何仲贤 10370851273" "微信零钱提现"
    time: "00:00:00"
    Assets:WeChat                          -1.00 CNY
    Assets:CEB:1027                         1.00 CNY

""",
"")

print("\n全部完成")
