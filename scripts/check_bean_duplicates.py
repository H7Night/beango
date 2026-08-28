# -*- coding: utf-8 -*-
"""检查 beancount 账本文件中的重复 uuid（合并新账单前先去重）。

用法: python scripts/check_bean_duplicates.py [文件或目录...]
  不带参数: 检查默认账本目录（../beancount 或当前目录）下所有 .bean 文件
  带文件:   仅检查指定文件
  带目录:   检查目录下所有 .bean 文件（排除 .git）

说明:
  - 同一 uuid 出现多次即为重复交易（合并账单时可能重复入账）；
  - 缺失 uuid 的交易（手工补记等）仅提示，不视为错误。
"""
import sys, re, os, glob
from collections import Counter

DATE_RE = re.compile(r'^(\d{4}-\d{2}-\d{2})\s\*')
UUID_RE = re.compile(r'^\s+uuid:\s*"([^"]+)"')


def check_bean_text(text, label):
    lines = text.split('\n')
    cur = None
    recs = []
    for ln in lines:
        m = DATE_RE.match(ln)
        if m:
            cur = {'date': m.group(1), 'uuid': None, 'first': ln.strip()[:60]}
            recs.append(cur)
        if cur is None:
            continue
        u = UUID_RE.search(ln)
        if u:
            cur['uuid'] = u.group(1)

    cc = Counter(r['uuid'] for r in recs if r['uuid'])
    dups = {k: v for k, v in cc.items() if v > 1}
    missing = [r for r in recs if not r['uuid']]

    if not dups:
        print(f'OK:   {label} ({len(recs)} 个交易块，无重复 uuid)')
        if missing:
            print(f'       提示: {len(missing)} 个交易缺失 uuid（手工补记，不视为错误）:')
            for r in missing[:10]:
                print(f'       {r["date"]} {r["first"]}')
        return True

    print(f'FAIL: {label} ({len(recs)} 个交易块)')
    for uuid, n in dups.items():
        print(f'   重复 uuid {uuid} x{n}:')
        for r in recs:
            if r['uuid'] == uuid:
                print(f'       {r["date"]} {r["first"]}')
    if missing:
        print(f'   缺失 uuid 的交易 {len(missing)} 个:')
        for r in missing[:10]:
            print(f'       {r["date"]} {r["first"]}')
    return False


def main():
    args = sys.argv[1:]
    if not args:
        script_dir = os.path.dirname(os.path.abspath(__file__))
        cand = os.path.join(script_dir, '..', 'beancount')
        root = cand if os.path.isdir(cand) else os.getcwd()
        paths = [p for p in glob.glob(os.path.join(root, '**', '*.bean'), recursive=True)
                 if '.git' not in p]
    elif os.path.isdir(args[0]):
        paths = [p for p in glob.glob(os.path.join(args[0], '**', '*.bean'), recursive=True)
                 if '.git' not in p]
    else:
        paths = args

    all_ok = True
    for p in sorted(paths):
        try:
            text = open(p, encoding='utf-8').read()
            all_ok = check_bean_text(text, os.path.relpath(p, os.getcwd())) and all_ok
        except Exception as e:
            print(f'FAIL: {p}: {e}')
            all_ok = False
    print(f'\n共检查 {len(paths)} 个文件')
    sys.exit(0 if all_ok else 1)


if __name__ == '__main__':
    main()
