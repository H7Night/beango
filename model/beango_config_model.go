package model

import (
	"fmt"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

// beangoConfigPath 引导配置文件路径。
// beango.yml 是全局配置的入口，其自身路径无法从配置中读取，故保留为常量；
// 其余可配置项（配置文件目录、文件名、端口、目录等）均从 beango.yml 读取。
const beangoConfigPath = "config/beango.yml"

// DefaultOutputFolder 输出根目录兜底值。
// out 目录已迁移至 test/out，实际路径由配置 outputFolder 控制。
const DefaultOutputFolder = "./test/out"

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

// beangoCacheMu 保护 beango.yml 的内存缓存（转换按行调用配置，避免重复读文件）
var (
	beangoCacheMu sync.RWMutex
	beangoCache   *beangoConfigFile
)

// loadBeangoConfig reads the beango.yml file (with in-memory cache)
func loadBeangoConfig() (*beangoConfigFile, error) {
	beangoCacheMu.RLock()
	if beangoCache != nil {
		c := beangoCache
		beangoCacheMu.RUnlock()
		return c, nil
	}
	beangoCacheMu.RUnlock()

	var bcf beangoConfigFile
	if err := readYAML(beangoConfigPath, &bcf); err != nil {
		return nil, fmt.Errorf("读取配置失败: %w", err)
	}
	if bcf.Beango == nil {
		bcf.Beango = make(map[string]string)
	}

	beangoCacheMu.Lock()
	beangoCache = &bcf
	beangoCacheMu.Unlock()
	return &bcf, nil
}

// saveBeangoConfig writes to beango.yml and invalidates the cache
func saveBeangoConfig(bcf *beangoConfigFile) error {
	if err := writeYAML(beangoConfigPath, bcf); err != nil {
		return err
	}
	beangoCacheMu.Lock()
	beangoCache = nil
	beangoCacheMu.Unlock()
	return nil
}

// ---- 配置辅助函数（运行环境相关，均从 beango.yml 读取，带默认值兜底） ----

// ConfigFolder 配置文件目录
func ConfigFolder() string {
	return GetConfigString("configFolder", "./config")
}

// AccountMapPath 账户映射文件路径
func AccountMapPath() string {
	return filepath.Join(ConfigFolder(), GetConfigString("accountMapFile", "account_map.yml"))
}

// CommodityMapPath 商品映射文件路径
func CommodityMapPath() string {
	return filepath.Join(ConfigFolder(), GetConfigString("commodityMapFile", "commodity_map.yml"))
}

// ServerPort Web 服务端口
func ServerPort() string {
	return GetConfigString("serverPort", "10777")
}

// WebDir 前端静态资源目录
func WebDir() string {
	return GetConfigString("webDir", "./web/dist")
}

// DefaultExpenseAccount 未匹配支出兜底账户
func DefaultExpenseAccount() string {
	return GetConfigString("defaultExpenseAccount", "Expenses:Other")
}

// DefaultIncomeAccount 未匹配收入兜底账户
func DefaultIncomeAccount() string {
	return GetConfigString("defaultIncomeAccount", "Income:Other")
}

// DefaultAssetAccount 未匹配资产/负债兜底账户
func DefaultAssetAccount() string {
	return GetConfigString("defaultAssetAccount", "Assets:Other")
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
