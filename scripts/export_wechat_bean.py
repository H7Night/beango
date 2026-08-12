# -*- coding: utf-8 -*-
"""从 beancount 导出 Assets:WeChat 交易明细"""
import sys
sys.stdout.reconfigure(encoding='utf-8')
sys.path.insert(0, r'C:\Users\user\Abandon\Projects\beango\beancount\.venv\Lib\site-packages')

from beancount import loader
from beancount.core import data
from decimal import Decimal

entries, errors, options = loader.load_file(r'C:\Users\user\Abandon\Projects\beango\beancount\main.bean')
if errors:
    print('ERRORS:', file=sys.stderr)
    for e in errors:
        print(e, file=sys.stderr)

print('date\tamount\tnarration\tpayee\tuuid\tstatus')
for entry in entries:
    if not isinstance(entry, data.Transaction):
        continue
    for posting in entry.postings:
        if posting.account == 'Assets:WeChat':
            amt = posting.units.number
            uuid = entry.meta.get('uuid', '')
            status = entry.meta.get('status', '')
            print(f"{entry.date}\t{amt}\t{entry.narration}\t{entry.payee}\t{uuid}\t{status}")
