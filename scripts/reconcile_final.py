# -*- coding: utf-8 -*-
"""最终核对: 账本 Assets:WeChat vs 账单零钱交易, 精确分解差异"""
import sys, csv
from decimal import Decimal
from collections import defaultdict, Counter
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

def affects_wechat(b):
    if '零钱' in b['pay']:
        return True
    if b['io'] == '收入' and ('存入零钱' in b['status'] or '零钱' in b['status']):
        return True
    if b['io'] == '/' and ('零钱充值' in b['t_type'] or '零钱提现' in b['t_type']):
        return True
    return False

# 账单零钱交易的期望影响金额(支出-, 收入+, 中性按类型)
bill_zl = [b for b in bill if affects_wechat(b)]
def bill_signed(b):
    if b['io'] == '支出':
        return -b['amt']
    if b['io'] == '收入':
        return b['amt']
    # 中性
    if '提现' in b['t_type']:
        return -b['amt']
    if '充值' in b['t_type']:
        return b['amt']
    return Decimal('0')

expected_net = sum(bill_signed(b) for b in bill_zl)
print(f"账单零钱交易期望净额(2025-01-01起, 不含手续费) = {expected_net}")

bean_net = sum(b['amt'] for b in bean_2025)
print(f"账本 Assets:WeChat 净额(2025-01-01起) = {bean_net}")
print(f"差异(账本-期望) = {bean_net - expected_net}")

# ---------- 按单号对齐 ----------
bill_by_id = defaultdict(list)
for b in bill_zl:
    bill_by_id[b['txn']].append(b)

matched_diff = Decimal('0')     # 单号匹配上的金额差异(账本 - 期望)
matched_items = []
missing = []                     # 账单有、账本无
for b in bill_zl:
    if b['txn'] in {x['uuid'] for x in bean_2025}:
        pass
    else:
        missing.append(b)

# 账本中有单号的, 逐一对比
bean_extra = []
bean_matched = set()
for b in bean_2025:
    if not b['uuid']:
        bean_extra.append(b)  # 无单号, 稍后人工
        continue
    matches = bill_by_id.get(b['uuid'], [])
    if not matches:
        bean_extra.append(b)
        continue
    m = matches[0]
    expect = bill_signed(m)
    if expect != b['amt']:
        matched_items.append((b, m, expect))
        matched_diff += (expect - b['amt'])
    bean_matched.add(b['uuid'])

print(f"\n=== 单号匹配上但金额不一致: {len(matched_items)} 笔, 合计(期望-账本) = {matched_diff} ===")
for b, m, expect in matched_items:
    print(f"  账本 {b['date']} {b['amt']:>9} vs 期望 {expect:>9} | {b['narration'][:22]} | {m['t_type']} | {m['status']} | {b['uuid']}")

# 缺失明细(账单零钱交易, 账本无单号)
missing_no_rec = [b for b in missing if b['txn'] not in {x['uuid'] for x in bean_2025}]
# 但账本可能有相同金额/日期的无单号记录(如 Jhon -5)
print(f"\n=== 账单零钱交易但账本无此单号: {len(missing)} 笔 ===")
miss_net = sum(bill_signed(b) for b in missing)
print(f"  缺失交易期望净额 = {miss_net}")
for b in sorted(missing, key=lambda x: x['date']):
    print(f"  {b['date']} {bill_signed(b):>9} | {b['io']:>2} {b['amt']:>7} | {b['t_type']:<12} | {b['party'][:14]:<14} | {b['status'][:12]:<12} | {b['txn']}")

# 账本无单号记录
print(f"\n=== 账本无单号的 Assets:WeChat 记录: {len(bean_extra)} 笔, 净额 = {sum(b['amt'] for b in bean_extra)} ===")
for b in sorted(bean_extra, key=lambda x: x['date']):
    print(f"  {b['date']} {b['amt']:>9} | {b['narration'][:24]:<24} | {b['payee'][:12]:<12} | {b['status'][:14]}")

# 账本重复单号
ids = Counter(b['uuid'] for b in bean_2025 if b['uuid'])
print(f"\n=== 账本重复单号 ===")
for uid, c in ids.items():
    if c > 1:
        for b in bean_2025:
            if b['uuid'] == uid:
                print(f"  {b['date']} {b['amt']:>9} | {b['narration'][:28]} | {uid}")
