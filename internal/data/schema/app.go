package schema

import "time"

// App 产品表 (替代 app_registry)
type App struct {
    Id              *int64     `gorm:"column:id;primary_key;AUTO_INCREMENT" mapper:"id"`
    Name            string     `gorm:"column:name;not null;comment:'显示名'" mapper:"name"`             // 显示名 仿真平台
    Code            string     `gorm:"column:code;not null;comment:'app 编号'" mapper:"code"`            // 编号 simulation
    EntryUrl        string     `gorm:"column:entry_url;not null;comment:'入口URL'" mapper:"entryUrl"`  // 入口URL
    Icon            string     `gorm:"column:icon;not null;comment:'图标'" mapper:"icon"`              // 图标
    Sort            int32      `gorm:"column:sort;default:0;comment:'排序'" mapper:"sort"`              // 排序
    Status          int8       `gorm:"column:status;default:1;comment:'1=显示 0=隐藏'" mapper:"status"` // 状态
    Desc            string     `gorm:"column:desc;comment:'描述&简介'" mapper:"desc"`                   // 描述&简介
    Detail          string     `gorm:"column:detail;comment:'JSON扩展'" mapper:"detail"`               // JSON扩展
    Type            string     `gorm:"column:type;default:'';comment:'app类型: platform=平台, link=链接'" mapper:"type"`          // app 类型: platform=平台, link=链接
    CreatedBy       int64      `gorm:"column:created_by" mapper:"createdBy"`
    CreatedAt       *time.Time `gorm:"column:created_at;autoCreateTime" mapper:"createdAt"`
    UpdatedBy       int64      `gorm:"column:updated_by" mapper:"updatedBy"`
    UpdatedAt       *time.Time `gorm:"column:updated_at;autoUpdateTime" mapper:"updatedAt"`
    DeletedUniqueId int64      `gorm:"column:deleted_unique_id;default:0" mapper:"deleteUniqueId"`
    DeletedAt       *time.Time `gorm:"column:deleted_at" mapper:"deletedAt"`
}

func (a *App) TableName() string {
    return "app"
}
