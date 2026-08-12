# -*- coding: utf-8 -*-
"""单号匹配 + 金额比对: 找出账本中金额与账单不一致的记录"""
import sys, csv
from decimal import Decimal
from collections import Counter, defaultdict
sys.stdout.reconfigure(encoding='utf-8')

from datetime import datetime, timedelta
def excel_to_date(v):
    f = float(v)
    return (datetime(1899, 12, 30) + timedelta(days=f)).strftime('%Y-%m-%d')

bean = []
with open(r'out/bean_wechat.tsv', encoding='utf-8') as f:
    next(f)
    for line in f:
        p = line.rstrip('\n').split('\t')
        bean.append(dict(date=p[0], amt=Decimal(p[1]), narration=p[2], payee=p[3],
                         uuid=p[4] if len(p) > 4 else '', status=p[5] if len(p) > 5 else ''))
bean_2025 = [b for b in bean if '2025-01-01' <= b['date'] <= '2026-08-12']

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

bill_by_id = defaultdict(list)
for b in bill:
    bill_by_id[b['txn']].append(b)

print("=== 单号相同但金额/日期不一致的账本记录 ===")
for b in bean_2025:
    if not b['uuid']:
        continue
    matches = bill_by_id.get(b['uuid'], [])
    if not matches:
        continue
    for m in matches:
        if m['amt'] != abs(b['amt']) or m['date'] != b['date']:
            sign = -1 if m['io'] == '支出' else 1
            print(f"账本: {b['date']} {b['amt']:>8} | {b['narration'][:25]} | {b['uuid']}")
            print(f"账单: {m['date']} {m['io']} {m['amt']:>8} | {m['t_type']} | {m['party'][:15]} | {m['status']} | {m['txn']}")
            print()

# 差额: 单号能匹配上的总金额差异
print("\n=== 单号匹配上的记录金额差异汇总 ===")
diff = Decimal('0')
count = 0
for b in bean_2025:
    if not b['uuid']:
        continue
    matches = bill_by_id.get(b['uuid'], [])
    if matches:
        m = matches[0]
        expected = -m['amt'] if m['io'] == '支出' else m['amt']
        if expected != b['amt']:
            diff += (expected - b['amt'])
            count += 1
print(f"金额不一致的记录数: {count}, 合计差异(期望-实际): {diff}")

# 按日期+金额模糊找 2026-05-17 -9 可能的真实交易
def affects(b):
    if '零钱' in b['pay']:
        return True
    if b['io'] == '收入' and ('存入零钱' in b['status'] or '零钱' in b['status']):
        return True
    if b['io'] == '/' and ('零钱充值' in b['t_type'] or '零钱提现' in b['t_type']):
        return True
    return False

print("\n=== 账单中 2026-05-17 附近的零钱交易 ===")
for b in bill:
    if '2026-05-15' <= b['date'] <= '2026-05-19' and affects(b):
        print(f"  {b['date']} {b['io']:>3} {b['amt']:>8} | {b['t_type']} | {b['party'][:18]} | {b['goods'][:20]} | {b['pay']} | {b['status']} | {b['txn']}")
