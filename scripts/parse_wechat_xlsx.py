# -*- coding: utf-8 -*-
"""解析微信支付账单 xlsx，输出 CSV 到 stdout"""
import zipfile, re, sys, csv
import xml.etree.ElementTree as ET

NS = 'http://schemas.openxmlformats.org/spreadsheetml/2006/main'

def parse_xlsx(path):
    z = zipfile.ZipFile(path)
    ss = z.read('xl/sharedStrings.xml').decode('utf-8')
    root = ET.fromstring(ss)
    strings = []
    for si in root.iter('{%s}si' % NS):
        text = ''.join(t.text or '' for t in si.iter('{%s}t' % NS))
        strings.append(text)
    sheet = z.read('xl/worksheets/sheet1.xml').decode('utf-8')
    sroot = ET.fromstring(sheet)
    rows = []
    for row in sroot.iter('{%s}row' % NS):
        cells = {}
        for c in row.iter('{%s}c' % NS):
            ref = c.get('r')
            col = re.match(r'[A-Z]+', ref).group()
            t = c.get('t')
            v = c.find('{%s}v' % NS)
            val = None
            if v is not None:
                val = v.text
                if t == 's':
                    val = strings[int(val)]
            cells[col] = val
        rows.append(cells)
    return rows

if __name__ == '__main__':
    sys.stdout.reconfigure(encoding='utf-8')
    w = csv.writer(sys.stdout)
    for path in sys.argv[1:]:
        rows = parse_xlsx(path)
        w.writerow(['== FILE ==', path])
        for r in rows:
            w.writerow([r.get(c) for c in sorted(r.keys())])
