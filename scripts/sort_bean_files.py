# -*- coding: utf-8 -*-
"""
将 beancount 账本文件按日期正序（Chronological Order）排列。

用法: python scripts/sort_bean_files.py [账本目录] [文件...]
  不带参数: 排序默认账本目录（../beancount 或当前目录）下所有 .bean 文件
  带目录:   排序指定目录下所有 .bean 文件（排除 .git）
  带文件:   仅排序指定文件

设计: beancount 是单过式（single-pass）解析引擎，按时间线模拟账户余额，
      文件内交易应按日期正序排列。beango 生成的月度文件为倒序，本脚本修正。
      同一日期的所有条目（交易/断言/open 等）合并为日期组，组内保持原顺序。
"""
import io, sys, re, os, glob

DATE_RE = re.compile(r'^(\d{4}-\d{2}-\d{2})\s')
TIME_RE = re.compile(r'^\s+time:\s*"(\d{2}:\d{2}:\d{2})"')


def sort_bean_text(text):
    """按日期正序排列 beancount 文本，返回排序后的文本。"""
    lines = text.split('\n')

    # 头部：文件开头到第一个日期行之前（注释/设置等）
    header = []
    i = 0
    while i < len(lines):
        if DATE_RE.match(lines[i]):
            break
        header.append(lines[i])
        i += 1

    def block_time(blk):
        """提取块内的 time 字段；无 time 返回 '00:00:00'（排在当日最前，如 balance 断言）。"""
        for line in blk:
            m = TIME_RE.match(line)
            if m:
                return m.group(1)
        return '00:00:00'

    # 解析为 (日期, 块行列表)
    all_blocks = []
    cur_date = None
    cur = []
    for line in lines[i:]:
        m = DATE_RE.match(line)
        if m:
            if cur_date is not None:
                all_blocks.append((cur_date, cur))
            cur_date = m.group(1)
            cur = [line]
        else:
            if cur_date is not None:
                cur.append(line)
            else:
                header.append(line)
    if cur_date is not None:
        all_blocks.append((cur_date, cur))

    # 按日期分组；组内按 time 正序（稳定排序）
    groups = []  # (date, [ (time, block_lines) ])
    for date_s, blk in all_blocks:
        t = block_time(blk)
        if groups and groups[-1][0] == date_s:
            groups[-1][1].append((t, blk))
        else:
            groups.append((date_s, [(t, blk)]))
    groups.sort(key=lambda x: x[0])
    for _, blks in groups:
        blks.sort(key=lambda x: x[0])  # 按 time 正序，同 time 保持原顺序（稳定）

    # 重建
    out = []
    while header and not header[-1].strip():
        header.pop()
    out.extend(header)
    if header:
        out.append('')
    for idx, (date_s, timed_blocks) in enumerate(groups):
        # 组内拼接各块：单行指令（open/close/pad）之间不插空行，
        # 其余（交易/断言块）之间插一个空行
        glines = []
        last_single = False
        for t, blk in timed_blocks:
            blk2 = list(blk)
            while blk2 and not blk2[-1].strip():
                blk2.pop()
            is_single = len([l for l in blk2 if l.strip()]) == 1  # 仅一行（如 open/close）
            if glines and glines[-1].strip() and blk2 and blk2[0].strip():
                if not (is_single and last_single):
                    glines.append('')
            glines.extend(blk2)
            last_single = is_single
        while glines and not glines[-1].strip():
            glines.pop()
        out.extend(glines)
        if idx < len(groups) - 1:
            out.append('')
    out.append('')
    return '\n'.join(out)


def sort_bean_file(path):
    raw = open(path, 'rb').read()
    nl = b'\r\n' if b'\r\n' in raw else b'\n'
    text = raw.decode('utf-8')
    newtext = sort_bean_text(text)
    if nl == b'\r\n':
        newtext = newtext.replace('\n', '\r\n')
    if newtext != text:
        open(path, 'wb').write(newtext.encode('utf-8'))
        return True
    return False


if __name__ == '__main__':
    args = [a for a in sys.argv[1:]]
    if not args:
        script_dir = os.path.dirname(os.path.abspath(__file__))
        cand = os.path.join(script_dir, '..', 'beancount')
        root = cand if os.path.isdir(cand) else os.getcwd()
        paths = [p for p in glob.glob(os.path.join(root, '**', '*.bean'), recursive=True) if '.git' not in p]
    else:
        root = args[0]
        if os.path.isdir(root):
            paths = [p for p in glob.glob(os.path.join(root, '**', '*.bean'), recursive=True) if '.git' not in p]
        else:
            paths = args
    changed = 0
    for p in paths:
        try:
            if sort_bean_file(p):
                print(f"OK: {os.path.relpath(p, os.getcwd())}")
                changed += 1
        except Exception as e:
            print(f"FAIL: {p}: {e}")
    print(f"\n共排序 {changed} 个文件")
