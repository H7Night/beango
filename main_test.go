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
