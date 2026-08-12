# -*- coding: utf-8 -*-
"""核对：账本 Assets:WeChat 余额 vs 微信账单零钱交易净额"""
import sys, csv
from decimal import Decimal
sys.stdout.reconfigure(encoding='utf-8')

# ---------- 1. 账本 ----------
bean = []
with open(r'out/bean_wechat.tsv', encoding='utf-8') as f:
    next(f)
    for line in f:
        p = line.rstrip('\n').split('\t')
        date, amt, narration, payee = p[0], Decimal(p[1]), p[2], p[3]
        uuid = p[4] if len(p) > 4 else ''
        status = p[5] if len(p) > 5 else ''
        bean.append(dict(date=date, amt=amt, narration=narration, payee=payee, uuid=uuid, status=status))

total_bean = sum(b['amt'] for b in bean)
bean_2025_plus = sum(b['amt'] for b in bean if b['date'] >= '2025-01-01')
print(f"[账本] Assets:WeChat 全部交易净额 = {total_bean}")
print(f"[账本] 2025-01-01 起净额 = {bean_2025_plus}")

# 2024-12-31 为止的余额（起始）
start_bal = sum(b['amt'] for b in bean if b['date'] < '2025-01-01')
print(f"[账本] 2024-12-31 为止余额(起始) = {start_bal}")

# ---------- 2. 账单 ----------
def load_wechat(path):
    rows = []
    with open(path, encoding='utf-8') as f:
        r = csv.reader(f)
        started = False
        for line in r:
            if not started:
                if line and line[0] == '交易时间':
                    started = True
                continue
            if len(line) >= 11:
                rows.append(line)
    return rows

def excel_to_date(v):
    from datetime import datetime, timedelta
    f = float(v)
    return (datetime(1899, 12, 30) + timedelta(days=f)).strftime('%Y-%m-%d')

def excel_to_dt(v):
    from datetime import datetime, timedelta
    f = float(v)
    return (datetime(1899, 12, 30) + timedelta(days=f)).strftime('%Y-%m-%d %H:%M:%S')

all_rows = []
for p in [r'out/wechat_2025.csv', r'out/wechat_2026.csv']:
    for r in load_wechat(p):
        all_rows.append(r)

# 影响零钱余额的交易
zl_in = Decimal('0')   # 零钱流入
zl_out = Decimal('0')  # 零钱流出
zl_rows = []
for r in all_rows:
    t_type, party, goods, io, amt, pay = r[1], r[2], r[3], r[4], r[5], r[6]
    status = r[7]
    if status in ('已全额退款', '交易关闭', '对方已退还'):
        continue  # 全额退款/关闭不产生净影响（但支出和退款分开列出的，不能跳过！）
    a = Decimal(amt)
    affects = False
    sign = 0
    if '零钱' in pay:  # 支付方式为零钱（支出）
        sign = -1 if io == '支出' else 1
        affects = True
    elif io == '收入' and '存入零钱' in status:
        sign = 1
        affects = True
    elif io == '收入' and pay == '/':
        # 收入但支付方式为/，看状态
        if '零钱' in status:
            sign = 1
            affects = True
    if affects:
        zl_rows.append((r, sign * a))
        if sign > 0:
            zl_in += a
        else:
            zl_out += a

print(f"\n[账单] 影响零钱的交易笔数 = {len(zl_rows)}")
print(f"[账单] 零钱流入 = {zl_in}, 零钱流出 = {zl_out}, 净额 = {zl_in - zl_out}")

# 按支付方式全部分类看一下中性交易
print("\n[账单] 中性交易(收/支=/)明细:")
for r in all_rows:
    if r[4] == '/':
        print("  ", excel_to_dt(r[0]), r[1], r[2], r[3][:15], r[5], r[6], r[7])
