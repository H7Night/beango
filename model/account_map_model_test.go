package model

import (
	"os"
	"testing"
	"unicode/utf8"
)

func TestSortBySpecificity(t *testing.T) {
	in := []AccountMap{
		{Keyword: "余额", Account: "A", Type: "asset"},
		{Keyword: "余额宝", Account: "B", Type: "asset"},
		{Keyword: "招商银行", Account: "C", Type: "asset"},
		{Keyword: "招商银行信用卡(2035)", Account: "D", Type: "asset"},
	}
	got := SortBySpecificity(in)
	for i := 0; i < len(got)-1; i++ {
		li := utf8.RuneCountInString(got[i].Keyword)
		lj := utf8.RuneCountInString(got[i+1].Keyword)
		if li < lj {
			t.Fatalf("长度应降序，位置 %d: %q(len=%d) -> %q(len=%d)", i, got[i].Keyword, li, got[i+1].Keyword, lj)
		}
		if li == lj && got[i].Keyword > got[i+1].Keyword {
			t.Fatalf("同长应关键词升序，位置 %d: %q -> %q", i, got[i].Keyword, got[i+1].Keyword)
		}
	}
	if in[0].Keyword != "余额" {
		t.Fatalf("不应修改入参，实际 %q", in[0].Keyword)
	}
}

func TestGetAccountMapSortedBySpecificity(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(".."); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })

	if err := LoadAccountMap(); err != nil {
		t.Fatalf("LoadAccountMap: %v", err)
	}
	got := GetAccountMap()
	if len(got) < 2 {
		t.Fatalf("映射过少: %d", len(got))
	}
	for i := 0; i < len(got)-1; i++ {
		li := utf8.RuneCountInString(got[i].Keyword)
		lj := utf8.RuneCountInString(got[i+1].Keyword)
		if li < lj {
			t.Fatalf("GetAccountMap 未按具体度排序，位置 %d: %q(len=%d) -> %q(len=%d)", i, got[i].Keyword, li, got[i+1].Keyword, lj)
		}
	}
}
