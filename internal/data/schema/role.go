package schema

import "time"

type Role struct {
    Id              *int64     `gorm:"id" mapper:"id"`                              // Unique identifier
    Name            string     `gorm:"name" mapper:"name"`                          // Role name
    Code            string     `gorm:"column:code;default:''" mapper:"code"`        // 角色编码 role+id, 如 role00003, 创建后自动生成只读
    Remarks         string     `json:"remarks" mapper:"remarks"`            // remarks
    GroupId         int64      `gorm:"group_id" mapper:"groupId"`                   // role in which group
    App             int64      `gorm:"app" mapper:"app"`                   // app_id
    CreatedBy       int64      `gorm:"created_by" mapper:"createBy"`                // Creator ID
    CreatedAt       *time.Time `gorm:"created_at;autoCreateTime" mapper:"createAt"` // Creation timestamp
    UpdatedBy       int64      `gorm:"updated_by" mapper:"updateBy"`                // Updater ID
    UpdatedAt       *time.Time `gorm:"updated_at;autoUpdateTime" mapper:"updateAt"` // Update timestamp
    DeletedUniqueId int64      `gorm:"deleted_unique_id" mapper:"isDelete"`         // Deletion flag. 0 indicates not deleted, 1 indicates deleted
    DeletedAt       *time.Time `gorm:"deleted_at" mapper:"deletedAt"`               // Deletion timestamp
    System          int64      `gorm:"column:system" mapper:"System"`               // 1 cant edit
}

func (u *Role) TableName() string {
    return "role"
}
