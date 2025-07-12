package model

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type CommodityMapConfig struct {
	CommodityMap map[string][]string `yaml:"commodity_map"`
}

var loadedMap map[string]string

func LoadCommodityMap(path string) (map[string]string, error) {
	if loadedMap != nil {
		return loadedMap, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取commodity配置失败: %w", err)
	}

	var config CommodityMapConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("格式化commodity配置失败: %w", err)
	}

	result := make(map[string]string)
	for transType, keywords := range config.CommodityMap {
		for _, keyword := range keywords {
			result[keyword] = transType
		}
	}
	loadedMap = result
	return result, nil
}
