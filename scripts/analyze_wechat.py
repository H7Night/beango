# -*- coding: utf-8 -*-
"""分析微信账单 CSV: 按支付方式/收支类型统计, 打印零钱相关交易"""
import csv, sys, datetime, collections

def excel_serial_to_dt(v):
    try:
        f = float(v)
    except (TypeError, ValueError):
        return v
    return datetime.datetime(1899, 12, 30) + datetime.timedelta(days=f)

def load(path):
    rows = []
    with open(path, encoding='utf-8') as f:
        r = csv.reader(f)
        started = False
        header = None
        for line in r:
            if not started:
                if line and line[0] == '交易时间':
                    header = line
                    started = True
                continue
            if len(line) >= 11:
                rows.append(line)
    return header, rows

sys.stdout.reconfigure(encoding='utf-8')

for path in [r'out/wechat_2025.csv', r'out/wechat_2026.csv']:
    header, rows = load(path)
    print('=' * 90)
    print(path, 'header:', header)
    print('交易类型分布:', collections.Counter(r[1] for r in rows))
    print('支付方式分布:', collections.Counter(r[6] for r in rows))
    print('状态分布:', collections.Counter(r[7] for r in rows))
    print('收/支分布:', collections.Counter(r[4] for r in rows))
    # 零钱交易
    zl = [r for r in rows if '零钱' in r[6]]
    print(f'\n--- 支付方式含"零钱"的交易: {len(zl)} 笔 ---')
    for r in zl:
        dt = excel_serial_to_dt(r[0])
        print(f'{dt} | {r[1]} | {r[2]} | {r[3][:20]} | {r[4]} | {r[5]} | {r[6]} | {r[7]} | 备注:{r[10]}')
    print()
EOF_MARKER = None
