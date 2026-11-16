package schema

import "time"

// SystemConfig 系统配置表 (key-value)
type SystemConfig struct {
    Id              *int64     `gorm:"id" mapper:"id"`
    ConfigKey       string     `gorm:"column:config_key;not null;comment:'配置key'" mapper:"configKey"`      // 配置key
    ConfigValue     string     `gorm:"column:config_value;comment:'配置value'" mapper:"configValue"`          // 配置value
    Description     string     `gorm:"column:description;comment:'描述'" mapper:"description"`               // 描述
    CreatedBy       int64      `gorm:"created_by" mapper:"createdBy"`
    CreatedAt       *time.Time `gorm:"created_at;autoCreateTime" mapper:"createdAt"`
    UpdatedBy       int64      `gorm:"updated_by" mapper:"updatedBy"`
    UpdatedAt       *time.Time `gorm:"updated_at;autoUpdateTime" mapper:"updatedAt"`
    DeletedUniqueId int64      `gorm:"deleted_unique_id" mapper:"deleteUniqueId"`
    DeletedAt       *time.Time `gorm:"deleted_at" mapper:"deletedAt"`
}

func (s *SystemConfig) TableName() string {
    return "system_config"
}
