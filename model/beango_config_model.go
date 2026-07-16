package model

import (
	"fmt"
	"strconv"
	"time"
)

const beangoConfigPath = "config/beango.yml"

// beangoConfigFile mirrors the YAML file structure
type beangoConfigFile struct {
	Beango map[string]string `yaml:"beango"`
}

type BeangoConfig struct {
	ID          uint64    `json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	ConfigKey   string    `json:"config_key"`
	ConfigValue string    `json:"config_value"`
	Note        string    `json:"note"`
}

// loadBeangoConfig reads the beango.yml file
func loadBeangoConfig() (*beangoConfigFile, error) {
	var bcf beangoConfigFile
	if err := readYAML(beangoConfigPath, &bcf); err != nil {
		return nil, fmt.Errorf("读取配置失败: %w", err)
	}
	if bcf.Beango == nil {
		bcf.Beango = make(map[string]string)
	}
	return &bcf, nil
}

// saveBeangoConfig writes to beango.yml
func saveBeangoConfig(bcf *beangoConfigFile) error {
	return writeYAML(beangoConfigPath, bcf)
}

func GetBeangoConfigValue(key string) (string, error) {
	bcf, err := loadBeangoConfig()
	if err != nil {
		return "", err
	}
	val, ok := bcf.Beango[key]
	if !ok {
		return "", nil
	}
	return val, nil
}

func GetConfigString(key, defaultVal string) string {
	bcf, err := loadBeangoConfig()
	if err != nil {
		fmt.Printf("读取配置失败 key=%s: %v\n", key, err)
		return defaultVal
	}
	val, ok := bcf.Beango[key]
	if !ok || val == "" {
		return defaultVal
	}
	return val
}

func GetConfigBool(key string, defaultVal bool) bool {
	val := GetConfigString(key, "")
	if val == "" {
		return defaultVal
	}
	res, err := strconv.ParseBool(val)
	if err != nil {
		fmt.Printf("布尔配置解析失败 key=%s: %v\n", key, err)
		return defaultVal
	}
	return res
}

func GetConfigInt(key string, defaultVal int) int {
	val := GetConfigString(key, "")
	if val == "" {
		return defaultVal
	}
	res, err := strconv.Atoi(val)
	if err != nil {
		fmt.Printf("整数配置解析失败 key=%s: %v\n", key, err)
		return defaultVal
	}
	return res
}

// GetAllBeangoConfig 获取所有配置（转为 BeangoConfig 列表以兼容 Web API）
func GetAllBeangoConfig() ([]BeangoConfig, error) {
	bcf, err := loadBeangoConfig()
	if err != nil {
		return nil, err
	}
	var configs []BeangoConfig
	id := uint64(1)
	for k, v := range bcf.Beango {
		configs = append(configs, BeangoConfig{
			ID:          id,
			ConfigKey:   k,
			ConfigValue: v,
		})
		id++
	}
	return configs, nil
}

// CreateBeangoConfig 新增配置项
func CreateBeangoConfig(config BeangoConfig) error {
	bcf, err := loadBeangoConfig()
	if err != nil {
		return err
	}
	bcf.Beango[config.ConfigKey] = config.ConfigValue
	return saveBeangoConfig(bcf)
}

// UpdateBeangoConfig 更新配置项
func UpdateBeangoConfig(id uint64, config BeangoConfig) error {
	bcf, err := loadBeangoConfig()
	if err != nil {
		return err
	}
	// 先找到旧 key（通过 id 对应的 key）
	allConfigs, _ := getAllConfigsAsList(bcf)
	var oldKey string
	for _, c := range allConfigs {
		if c.ID == id {
			oldKey = c.ConfigKey
			break
		}
	}
	if oldKey != "" && oldKey != config.ConfigKey {
		delete(bcf.Beango, oldKey)
	}
	bcf.Beango[config.ConfigKey] = config.ConfigValue
	return saveBeangoConfig(bcf)
}

// DeleteBeangoConfig 删除配置项
func DeleteBeangoConfig(id uint64) error {
	bcf, err := loadBeangoConfig()
	if err != nil {
		return err
	}
	allConfigs, _ := getAllConfigsAsList(bcf)
	for _, c := range allConfigs {
		if c.ID == id {
			delete(bcf.Beango, c.ConfigKey)
			return saveBeangoConfig(bcf)
		}
	}
	return fmt.Errorf("config with id %d not found", id)
}

// getAllConfigsAsList helper: converts beangoConfigFile to []BeangoConfig with stable IDs
func getAllConfigsAsList(bcf *beangoConfigFile) ([]BeangoConfig, error) {
	var configs []BeangoConfig
	id := uint64(1)
	for k, v := range bcf.Beango {
		configs = append(configs, BeangoConfig{
			ID:          id,
			ConfigKey:   k,
			ConfigValue: v,
		})
		id++
	}
	return configs, nil
}
