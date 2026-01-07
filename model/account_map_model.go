package model

import (
	"encoding/json"
	"log"
	"time"
)

var mappings []AccountMap

type AccountMap struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"ID"`
	Keyword   string    `gorm:"type:varchar(64);index" json:"keyword"`
	Account   string    `gorm:"type:varchar(128)" json:"account"`
	Type      string    `gorm:"type:varchar(32)" json:"type"`
	CreatedAt time.Time `gorm:"type:timestamp;autoCreateTime;default:CURRENT_TIMESTAMP" json:"createdAt"`
	UpdatedAt time.Time `gorm:"type:timestamp;autoUpdateTime" json:"updated_at"`
}

// MarshalJSON 自定义 JSON 序列化
func (a AccountMap) MarshalJSON() ([]byte, error) {
	type Alias AccountMap // 创建别名避免递归调用
	
	return json.Marshal(&struct {
		*Alias
		CreatedAt string `json:"createdAt"`
		UpdatedAt string `json:"updated_at"`
	}{
		Alias:     (*Alias)(&a),
		CreatedAt: formatTime(a.CreatedAt),
		UpdatedAt: formatTime(a.UpdatedAt),
	})
}

// UnmarshalJSON 自定义 JSON 反序列化（如果需要从 JSON 创建对象）
func (a *AccountMap) UnmarshalJSON(data []byte) error {
	type Alias AccountMap
	aux := &struct {
		*Alias
		CreatedAt string `json:"createdAt"`
		UpdatedAt string `json:"updated_at"`
	}{
		Alias: (*Alias)(a),
	}
	
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}
	
	// 解析时间字符串
	var err error
	if aux.CreatedAt != "" {
		a.CreatedAt, err = parseTime(aux.CreatedAt)
		if err != nil {
			return err
		}
	}
	if aux.UpdatedAt != "" {
		a.UpdatedAt, err = parseTime(aux.UpdatedAt)
		if err != nil {
			return err
		}
	}
	
	return nil
}

// formatTime 格式化时间
func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02 15:04:05")
}

// parseTime 解析时间字符串
func parseTime(str string) (time.Time, error) {
	if str == "" {
		return time.Time{}, nil
	}
	return time.Parse("2006-01-02 15:04:05", str)
}

// LoadAccountMapFromDB 加载数据库映射库
func LoadAccountMapFromDB() error {
	err := db.Find(&mappings).Error
	if err != nil {
		log.Printf("无法加载账户映射: %v", err)
	}
	return err
}

func GetAccountMap() []AccountMap {
	return mappings
}

// CreateAccountMap 创建账户映射
func CreateAccountMap(accountMap AccountMap) error {
	err := db.Create(&accountMap).Error
	if err == nil {
		// 创建成功后重新加载缓存
		LoadAccountMapFromDB()
	}
	return err
}

// UpdateAccountMap 更新账户映射
func UpdateAccountMap(id uint64, mapp AccountMap) error {
	err := db.Model(&AccountMap{}).Where("id = ?", id).Updates(map[string]interface{}{
		"keyword": mapp.Keyword,
		"account": mapp.Account,
		"type":    mapp.Type,
	}).Error
	if err == nil {
		// 更新成功后重新加载缓存
		LoadAccountMapFromDB()
	}
	return err
}

// DeleteAccountMap 删除账户映射
func DeleteAccountMap(id uint64) error {
	err := db.Where("id = ?", id).Delete(&AccountMap{}).Error
	if err == nil {
		// 删除成功后重新加载缓存
		LoadAccountMapFromDB()
	}
	return err
}

// GetAllAccountMap 获取所有账户映射
func GetAllAccountMap() ([]AccountMap, error) {
	var mappings []AccountMap
	err := db.Find(&mappings).Error
	return mappings, err
}

// GetAccountByKeyword 根据关键词查找账户（缓存查询）
func GetAccountByKeyword(keyword string) (AccountMap, bool) {
	for _, mapping := range mappings {
		if mapping.Keyword == keyword {
			return mapping, true
		}
	}
	return AccountMap{}, false
}

// RefreshAccountMapCache 刷新缓存
func RefreshAccountMapCache() error {
	return LoadAccountMapFromDB()
}