package service

import (
	"reflect"
	"testing"
)

func TestExtractUUID(t *testing.T) {
	entry := `2026-08-05 * "招商银行" "信用卡还款"
    time: "17:15:06"
    uuid: "2026080500003001520093859760"
    status: "还款成功"`
	if got := extractUUID(entry); got != "2026080500003001520093859760" {
		t.Errorf("extractUUID 期望 %q，实际 %q", "2026080500003001520093859760", got)
	}

	noUUID := `2026-08-05 * "招商银行" "信用卡还款"
    time: "17:15:06"`
	if got := extractUUID(noUUID); got != "" {
		t.Errorf("无 uuid 时应返回空串，实际 %q", got)
	}
}

func TestDedupByUUID(t *testing.T) {
	mkEntry := func(id string) string {
		return "2026-08-05 * \"x\" \"y\"\n    uuid: \"" + id + "\""
	}
	noUUID := "2026-08-05 * \"x\" \"y\"\n    time: \"10:00:00\""

	t.Run("无uuid不去重", func(t *testing.T) {
		existing := []string{noUUID}
		incoming := []string{noUUID}
		got := dedupByUUID(existing, incoming)
		if len(got) != 1 {
			t.Fatalf("期望 1 条，实际 %d", len(got))
		}
	})

	t.Run("相同uuid去重", func(t *testing.T) {
		existing := []string{mkEntry("uuid-1")}
		incoming := []string{mkEntry("uuid-1"), mkEntry("uuid-2")}
		got := dedupByUUID(existing, incoming)
		want := []string{mkEntry("uuid-2")}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("期望 %v，实际 %v", want, got)
		}
	})

	t.Run("incoming内部重复去重", func(t *testing.T) {
		existing := []string{mkEntry("uuid-1")}
		incoming := []string{mkEntry("uuid-2"), mkEntry("uuid-2"), mkEntry("uuid-3")}
		got := dedupByUUID(existing, incoming)
		want := []string{mkEntry("uuid-2"), mkEntry("uuid-3")}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("期望 %v，实际 %v", want, got)
		}
	})

	t.Run("全部已存在则全部丢弃", func(t *testing.T) {
		existing := []string{mkEntry("uuid-1")}
		incoming := []string{mkEntry("uuid-1")}
		if got := dedupByUUID(existing, incoming); len(got) != 0 {
			t.Errorf("期望 0 条，实际 %d", len(got))
		}
	})
}
