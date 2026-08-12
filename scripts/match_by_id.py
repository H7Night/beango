# -*- coding: utf-8 -*-
"""按交易单号匹配账本 vs 账单，找出差异"""
import sys, csv
from decimal import Decimal
from collections import Counter, defaultdict
sys.stdout.reconfigure(encoding='utf-8')

# ---------- 账本 ----------
bean = []  # (date, amt, narration, payee, uuid, status)
with open(r'out/bean_wechat.tsv', encoding='utf-8') as f:
    next(f)
    for line in f:
        p = line.rstrip('\n').split('\t')
        uuid = p[4] if len(p) > 4 else ''
        status = p[5] if len(p) > 5 else ''
        bean.append(dict(date=p[0], amt=Decimal(p[1]), narration=p[2], payee=p[3], uuid=uuid, status=status))

bean_2025 = [b for b in bean if b['date'] >= '2025-01-01' and b['date'] <= '2026-08-12']
print(f"账本 2025-2026 微信交易: {len(bean_2025)} 笔, 净额 = {sum(b['amt'] for b in bean_2025)}")

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

from datetime import datetime, timedelta
def excel_to_dt(v):
    f = float(v)
    return (datetime(1899, 12, 30) + timedelta(days=f)).strftime('%Y-%m-%d %H:%M:%S')

def excel_to_date(v):
    return excel_to_dt(v)[:10]

bill = []  # (date, dt, t_type, party, goods, io, amt, pay, status, txn_no, merchant_no, remark)
for p in [r'out/wechat_2025.csv', r'out/wechat_2026.csv']:
    for r in load_wechat(p):
        bill.append(dict(date=excel_to_date(r[0]), dt=excel_to_dt(r[0]), t_type=r[1], party=r[2],
                         goods=r[3], io=r[4], amt=Decimal(r[5]), pay=r[6], status=r[7],
                         txn=r[8], merchant=r[9], remark=r[10]))
print(f"账单 2025-2026 全部交易: {len(bill)} 笔")

# ---------- 匹配: 按交易单号 ----------
bill_ids = Counter(b['txn'] for b in bill)
bean_ids = Counter(b['uuid'] for b in bean_2025 if b['uuid'])

print(f"\n=== 按交易单号匹配 ===")
# 账单有、账本无
missing = [b for b in bill if b['txn'] not in bean_ids]
print(f"账单有但账本无(漏记?): {len(missing)} 笔")
for b in sorted(missing, key=lambda x: x['date']):
    print(f"  {b['date']} {b['io']:>3} {b['amt']:>8} | {b['t_type']} | {b['party'][:15]} | {b['goods'][:20]} | {b['pay']} | {b['status']} | {b['txn']}")

# 账本有、账单无
extra = [b for b in bean_2025 if b['uuid'] and b['uuid'] not in bill_ids]
print(f"\n账本有但账单无(多记/记错?): {len(extra)} 笔")
for b in sorted(extra, key=lambda x: x['date']):
    print(f"  {b['date']} {b['amt']:>8} | {b['narration'][:25]} | {b['payee'][:12]} | {b['uuid']}")

# 重复: 同单号在账本出现多次
dup_bean = [uid for uid, c in bean_ids.items() if c > 1]
print(f"\n账本中重复单号: {len(dup_bean)} 个")
for uid in dup_bean:
    for b in bean_2025:
        if b['uuid'] == uid:
            print(f"  {b['date']} {b['amt']:>8} | {b['narration'][:25]} | {b['uuid']}")

# 账单重复单号
dup_bill = [uid for uid, c in bill_ids.items() if c > 1]
print(f"\n账单中重复单号: {len(dup_bill)} 个")
for uid in dup_bill:
    for b in bill:
        if b['txn'] == uid:
            print(f"  {b['date']} {b['io']:>3} {b['amt']:>8} | {b['t_type']} | {b['status']} | {b['txn']}")

# 无单号的账本记录
no_uid = [b for b in bean_2025 if not b['uuid']]
print(f"\n账本无单号的交易: {len(no_uid)} 笔")
for b in sorted(no_uid, key=lambda x: x['date']):
    print(f"  {b['date']} {b['amt']:>8} | {b['narration'][:25]} | {b['payee'][:12]}")
