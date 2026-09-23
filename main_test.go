package main

import "testing"

func TestParseBinanceArgs(t *testing.T) {
	got, err := parseArgs([]string{"-type", "binance", "-sync", "--from", "2026-01-01", "--to", "2026-09-22", "--symbols", "BTCUSDT,ETHUSDT"})
	if err != nil {
		t.Fatal(err)
	}
	if got.sourceType != "binance" || !got.binanceSync {
		t.Fatalf("options=%+v", got)
	}
	if len(got.symbols) != 2 || got.symbols[0] != "BTCUSDT" || got.symbols[1] != "ETHUSDT" {
		t.Fatalf("symbols=%v", got.symbols)
	}
}

func TestParseArgsAlipayMergeAndPass(t *testing.T) {
	got, err := parseArgs([]string{"-type", "alipay", "file.csv", "-output", "out", "-merge", "-p"})
	if err != nil {
		t.Fatal(err)
	}
	if got.sourceType != "alipay" || got.outputDir != "out" || !got.merge || !got.passAll {
		t.Fatalf("options=%+v", got)
	}
	if len(got.args) != 1 || got.args[0] != "file.csv" {
		t.Fatalf("args=%v", got.args)
	}
}

func TestParseArgsWechatEqualsForm(t *testing.T) {
	got, err := parseArgs([]string{"-type=wechat", "wx.xlsx"})
	if err != nil {
		t.Fatal(err)
	}
	if got.sourceType != "wechat" || len(got.args) != 1 || got.args[0] != "wx.xlsx" {
		t.Fatalf("options=%+v", got)
	}
}

func TestParseArgsHelp(t *testing.T) {
	if _, err := parseArgs([]string{"-h"}); err == nil {
		t.Fatal("help 应返回错误")
	}
}

func TestParseArgsUnknownFlag(t *testing.T) {
	if _, err := parseArgs([]string{"-type", "alipay", "--nope"}); err == nil {
		t.Fatal("未知选项应返回错误")
	}
}
