# -*- coding: utf-8 -*-
"""查询 beancount 账本中指定账户在各时点的余额，并列出其全部交易。

用法（需在安装了 beancount 的 Python 环境运行，如账本仓库的 .venv）:
  python scripts/account_balance.py <账本文件> <账户> [截止日期 YYYY-MM-DD ...]

示例:
  python account_balance.py ../beancount/main.bean Liabilities:CMBCreditCard:7046 2026-08-23 2026-08-25
  python account_balance.py ../beancount/main.bean Assets:AliPay:Balance

说明:
  - 通过 beancount loader + booking 计算余额，包含 pad 补差（与 bean-check 一致）；
  - 截止日期可选：输出该日期末（含当天交易）的余额；不传则输出当前余额；
  - 同时列出该账户所有交易（日期/对方/备注/金额），便于与真实账单对账。
"""
import io, sys, datetime
from collections import defaultdict

sys.stdout.reconfigure(encoding='utf-8')

from beancount import loader
from beancount.parser import booking


def main():
    if len(sys.argv) < 3:
        print(__doc__)
        sys.exit(1)
    ledger, account = sys.argv[1], sys.argv[2]
    dates = []
    for d in sys.argv[3:]:
        try:
            dates.append(datetime.date.fromisoformat(d))
        except ValueError:
            print(f'无效日期: {d}')
            sys.exit(1)
    dates = sorted(dates)

    entries, errors, options = loader.load_file(ledger)
    for e in errors:
        print(f'加载错误: {e.message}')
    entries, balance_errors = booking.book(entries, options)
    for e in balance_errors:
        print(f'断言错误: {e.message}')

    # 该账户的全部交易（按日期+金额）
    print(f'\n=== {account} 全部交易 ===')
    for e in entries:
        for p in getattr(e, 'postings', []):
            if p.account == account:
                payee = getattr(e, 'payee', None) or ''
                narration = getattr(e, 'narration', '') or ''
                print(f'{e.date}  {p.units.number:>10} {p.units.currency}  {payee[:20]}  {narration[:40]}')

    # 各时点余额（含 pad 补差，逐条累计）
    print(f'\n=== {account} 余额 ===')
    bal = defaultdict(float)
    snap = {}
    for e in entries:
        if getattr(e, 'date', None) is None:
            continue
        for p in getattr(e, 'postings', []):
            if p.account == account:
                bal[p.units.currency] += float(p.units.number)
        for d in dates:
            if e.date == d:
                snap[d] = dict(bal)
    for d in dates:
        snap_bal = snap.get(d, {})
        fmt = {k: round(v, 2) for k, v in snap_bal.items()}
        print(f'{d} 末: {fmt}')
    fmt = {k: round(v, 2) for k, v in bal.items()}
    print(f'当前: {fmt}')


if __name__ == '__main__':
    main()
