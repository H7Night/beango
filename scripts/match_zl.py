# -*- coding: utf-8 -*-
"""精确核对: 只匹配影响零钱余额的交易(账单零钱相关 vs 账本 Assets:WeChat)"""
import sys, csv
from decimal import Decimal
from collections import Counter
sys.stdout.reconfigure(encoding='utf-8')

from datetime import datetime, timedelta
def excel_to_date(v):
    f = float(v)
    return (datetime(1899, 12, 30) + timedelta(days=f)).strftime('%Y-%m-%d')

# ---------- 账本 ----------
bean = []
with open(r'out/bean_wechat.tsv', encoding='utf-8') as f:
    next(f)
    for line in f:
        p = line.rstrip('\n').split('\t')
        bean.append(dict(date=p[0], amt=Decimal(p[1]), narration=p[2], payee=p[3],
                         uuid=p[4] if len(p) > 4 else '', status=p[5] if len(p) > 5 else ''))
bean_2025 = [b for b in bean if '2025-01-01' <= b['date'] <= '2026-08-12']
bean_ids = Counter(b['uuid'] for b in bean_2025 if b['uuid'])

# ---------- 账单 ----------
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

bill = []
for p in [r'out/wechat_2025.csv', r'out/wechat_2026.csv']:
    for r in load_wechat(p):
        bill.append(dict(date=excel_to_date(r[0]), t_type=r[1], party=r[2], goods=r[3],
                         io=r[4], amt=Decimal(r[5]), pay=r[6], status=r[7], txn=r[8]))

# 影响零钱的交易筛选:
# - 支付方式=零钱 (支出/收入)
# - 收入且状态=已存入零钱
# - 中性交易(充值/提现): 收/支=/, 交易类型=零钱充值/零钱提现
def affects_wechat(b):
    if '零钱' in b['pay']:
        return True
    if b['io'] == '收入' and ('存入零钱' in b['status'] or '零钱' in b['status']):
        return True
    if b['io'] == '/' and ('零钱充值' in b['t_type'] or '零钱提现' in b['t_type']):
        return True
    return False

zl_bill = [b for b in bill if affects_wechat(b)]
print(f"账单中影响零钱的交易: {len(zl_bill)} 笔")
print(f"  其中收/支=/: {sum(1 for b in zl_bill if b['io']=='/')} 笔 (充值/提现)")

# 按单号匹配
missing = [b for b in zl_bill if b['txn'] not in bean_ids]
print(f"\n=== 账单零钱交易但账本 Assets:WeChat 无此单号: {len(missing)} 笔 ===")
net_missing = sum((-b['amt'] if b['io'] == '支出' else b['amt']) for b in missing)
print(f"这些缺失交易的净金额影响(负=账本少记支出/多记收入): {net_missing}")
for b in sorted(missing, key=lambda x: x['date']):
    sign = -1 if b['io'] == '支出' else (1 if b['io'] == '收入' else 0)
    amt = b['amt'] * sign
    print(f"  {b['date']} {b['io']:>3} {b['amt']:>8} | {b['t_type']:<10} | {b['party'][:16]:<16} | {b['goods'][:18]:<18} | {b['status'][:10]:<10} | {b['txn']}")

# 账本有、零钱账单无
extra = [b for b in bean_2025 if b['uuid'] and b['uuid'] not in {x['txn'] for x in zl_bill}]
print(f"\n=== 账本 Assets:WeChat 有单号但不在零钱账单中: {len(extra)} 笔 ===")
for b in sorted(extra, key=lambda x: x['date']):
    print(f"  {b['date']} {b['amt']:>8} | {b['narration'][:25]:<25} | {b['payee'][:12]:<12} | {b['uuid']}")

# 重复单号在账本
dup = [uid for uid, c in bean_ids.items() if c > 1]
print(f"\n=== 账本中重复单号: {len(dup)} 个 ===")
for uid in dup:
    for b in bean_2025:
        if b['uuid'] == uid:
            print(f"  {b['date']} {b['amt']:>8} | {b['narration'][:30]} | {b['uuid']}")

# 无单号账本记录
no_uid = [b for b in bean_2025 if not b['uuid']]
print(f"\n=== 账本无单号记录: {len(no_uid)} 笔 ===")
for b in sorted(no_uid, key=lambda x: x['date']):
    print(f"  {b['date']} {b['amt']:>8} | {b['narration'][:30]:<30} | {b['payee'][:12]:<12}")
