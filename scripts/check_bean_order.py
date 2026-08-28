# -*- coding: utf-8 -*-
"""校验 beancount 账本文件按日期+时分秒正序（Chronological Order）排列。

用法: python scripts/check_bean_order.py [文件或目录...]
  不带参数: 检查默认账本目录（../beancount 或当前目录）下所有 .bean 文件
  带文件:   仅检查指定文件
  带目录:   检查目录下所有 .bean 文件（排除 .git）

说明:
  - 交易块按 (日期, time) 正序检查，同日期内按 time 正序；
  - balance 断言、open/pad 等无 time 的条目不参与排序校验
    （通常放当日末尾表示日终快照，如 "2026-08-23 balance ..."）。
"""
import io, sys, re, os, glob

DATE_RE = re.compile(r'^(\d{4}-\d{2}-\d{2})\s')
TIME_RE = re.compile(r'^\s+time:\s*"(\d{2}:\d{2}:\d{2})"')


def check_bean_text(text):
    """提取每个交易块的 (日期, time)，返回 (序列, 乱序对列表)。"""
    lines = text.split('\n')
    seq = []
    cur_date = None
    cur_time = None
    for ln in lines:
        m = DATE_RE.match(ln)
        t = TIME_RE.match(ln)
        if m:
            if cur_date is not None and cur_time is not None:
                seq.append((cur_date, cur_time))
            cur_date = m.group(1)
            cur_time = None
        elif t and cur_date is not None:
            cur_time = t.group(1)
    if cur_date is not None and cur_time is not None:
        seq.append((cur_date, cur_time))

    bad = []
    for i in range(len(seq) - 1):
        if seq[i] > seq[i + 1]:
            bad.append((seq[i], seq[i + 1]))
    return seq, bad


def check_bean_file(path):
    raw = open(path, 'rb').read()
    text = raw.decode('utf-8')
    seq, bad = check_bean_text(text)
    if bad:
        print(f'FAIL: {os.path.relpath(path, os.getcwd())} ({len(seq)} 个交易块)')
        for a, b in bad[:20]:
            print(f'   乱序: {a} -> {b}')
        return False
    print(f'OK:   {os.path.relpath(path, os.getcwd())} ({len(seq)} 个交易块)')
    return True


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
            all_ok = check_bean_file(p) and all_ok
        except Exception as e:
            print(f'FAIL: {p}: {e}')
            all_ok = False
    print(f'\n共检查 {len(paths)} 个文件')
    sys.exit(0 if all_ok else 1)


if __name__ == '__main__':
    main()
