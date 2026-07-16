package model

import (
	"fmt"
	"log"
)

const accountMapPath = "config/account_map.yml"

// accountMapFile mirrors the YAML file structure
type accountMapFile struct {
	AccountMaps map[string]AccountMapEntry `yaml:"account_maps"`
}

// AccountMapEntry is the YAML-stored value for each keyword
type AccountMapEntry struct {
	Account string `yaml:"account"`
	Type    string `yaml:"type"`
}

// AccountMap is the API-facing struct (converted from map on read)
type AccountMap struct {
	Keyword string `json:"keyword"`
	Account string `json:"account"`
	Type    string `json:"type"`
}

var accountMapsCache []AccountMap
var accountMapsLoaded bool

// loadAccountMapFile reads the YAML file
func loadAccountMapFile() (*accountMapFile, error) {
	var af accountMapFile
	if err := readYAML(accountMapPath, &af); err != nil {
		return nil, fmt.Errorf("读取账户映射配置失败: %w", err)
	}
	if af.AccountMaps == nil {
		af.AccountMaps = make(map[string]AccountMapEntry)
	}
	return &af, nil
}

// saveAccountMapFile writes the YAML file
func saveAccountMapFile(af *accountMapFile) error {
	return writeYAML(accountMapPath, af)
}

// mapToSlice converts the map to a sorted slice for cache/API
func mapToSlice(m map[string]AccountMapEntry) []AccountMap {
	result := make([]AccountMap, 0, len(m))
	for keyword, entry := range m {
		result = append(result, AccountMap{
			Keyword: keyword,
			Account: entry.Account,
			Type:    entry.Type,
		})
	}
	return result
}

// LoadAccountMap 加载账户映射到缓存
func LoadAccountMap() error {
	af, err := loadAccountMapFile()
	if err != nil {
		log.Printf("无法加载账户映射: %v", err)
		return err
	}
	accountMapsCache = mapToSlice(af.AccountMaps)
	accountMapsLoaded = true
	return nil
}

// LoadAccountMapFromDB 保留旧函数名以兼容现有调用
func LoadAccountMapFromDB() error {
	return LoadAccountMap()
}

func GetAccountMap() []AccountMap {
	if !accountMapsLoaded {
		_ = LoadAccountMap()
	}
	return accountMapsCache
}

// CreateAccountMap 创建账户映射
func CreateAccountMap(am AccountMap) error {
	af, err := loadAccountMapFile()
	if err != nil {
		return err
	}

	if _, exists := af.AccountMaps[am.Keyword]; exists {
		return fmt.Errorf("关键词 '%s' 已存在", am.Keyword)
	}

	af.AccountMaps[am.Keyword] = AccountMapEntry{
		Account: am.Account,
		Type:    am.Type,
	}

	if err := saveAccountMapFile(af); err != nil {
		return err
	}

	accountMapsCache = mapToSlice(af.AccountMaps)
	return nil
}

// UpdateAccountMap 更新账户映射（通过 keyword 定位）
func UpdateAccountMap(keyword string, am AccountMap) error {
	af, err := loadAccountMapFile()
	if err != nil {
		return err
	}

	if _, exists := af.AccountMaps[keyword]; !exists {
		return fmt.Errorf("关键词 '%s' 不存在", keyword)
	}

	// 如果 keyword 有变化，删除旧 key，写入新 key
	if keyword != am.Keyword {
		delete(af.AccountMaps, keyword)
	}

	af.AccountMaps[am.Keyword] = AccountMapEntry{
		Account: am.Account,
		Type:    am.Type,
	}

	if err := saveAccountMapFile(af); err != nil {
		return err
	}

	accountMapsCache = mapToSlice(af.AccountMaps)
	return nil
}

// DeleteAccountMap 删除账户映射
func DeleteAccountMap(keyword string) error {
	af, err := loadAccountMapFile()
	if err != nil {
		return err
	}

	if _, exists := af.AccountMaps[keyword]; !exists {
		return fmt.Errorf("关键词 '%s' 不存在", keyword)
	}

	delete(af.AccountMaps, keyword)

	if err := saveAccountMapFile(af); err != nil {
		return err
	}

	accountMapsCache = mapToSlice(af.AccountMaps)
	return nil
}

// GetAllAccountMap 获取所有账户映射
func GetAllAccountMap() ([]AccountMap, error) {
	af, err := loadAccountMapFile()
	if err != nil {
		return nil, err
	}
	return mapToSlice(af.AccountMaps), nil
}

// GetAccountByKeyword 根据关键词查找账户（缓存查询）
func GetAccountByKeyword(keyword string) (AccountMap, bool) {
	for _, mapping := range GetAccountMap() {
		if mapping.Keyword == keyword {
			return mapping, true
		}
	}
	return AccountMap{}, false
}

// RefreshAccountMapCache 刷新缓存
func RefreshAccountMapCache() error {
	return LoadAccountMap()
}
