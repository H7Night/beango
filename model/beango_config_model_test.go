package model

import (
	"os"
	"path/filepath"
	"testing"
)

// resetBeangoConfigCache 清空 beango.yml 的内存缓存，避免测试间相互污染。
func resetBeangoConfigCache() {
	beangoCacheMu.Lock()
	beangoCache = nil
	beangoCacheMu.Unlock()
}

func TestResolveBeangoConfigPathPrimary(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(".."); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })

	got, err := resolveBeangoConfigPath()
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if got != beangoConfigPrimary {
		t.Fatalf("仓库根目录应命中 %s，实际 %q", beangoConfigPrimary, got)
	}
}

func TestResolveBeangoConfigPathFallback(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "beango.yml"), []byte("beango:\n  outputFolder: \"./tmpout\"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(wd)
		resetBeangoConfigCache()
	})

	resetBeangoConfigCache()

	got, err := resolveBeangoConfigPath()
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if got != beangoConfigFallback {
		t.Fatalf("无 config/ 时应回退 %s，实际 %q", beangoConfigFallback, got)
	}
	if v := GetConfigString("outputFolder", "DEF"); v != "./tmpout" {
		t.Fatalf("应读取回退配置 outputFolder=./tmpout，实际 %q", v)
	}
}

func TestResolveBeangoConfigPathMissing(t *testing.T) {
	dir := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })

	if _, err := resolveBeangoConfigPath(); err == nil {
		t.Fatal("两个候选都不存在时应返回错误")
	}
}
