package schema

import "time"

type Group struct {
    Id              *int64     `gorm:"id" mapper:"id"`                               // Unique identifier
    System          int8       `gorm:"system;default:1" mapper:"system"`           // system 0 is system .1 is user create
    Name            string     `gorm:"name" mapper:"groupName"`                      // group name
    GroupCode       string     `gorm:"column:group_code;default:''" mapper:"groupCode"`                       // 统一社会信用代码/企业代码
    GroupCreatedAt  *time.Time `gorm:"column:group_created_at" mapper:"groupCreatedAt"`                       // 企业创建时间
    Status          int8       `gorm:"column:status;default:1" mapper:"status"`                              // 企业状态 1=正常 0=停用
    GroupType       int8       `gorm:"group_type;default:1" mapper:"groupType"`       // 类型: 1-自有团队, 2-供应商
    Remarks         string     `gorm:"remarks" mapper:"remarks"`                     // remarks
    Managers        Int64Array `gorm:"managers" mapper:"managers"`                   // the group managers
    App             int64      `gorm:"app" mapper:"app"`                             // App ID (老逻辑保留,新逻辑不读写)
    CreatedBy       int64      `gorm:"created_by" mapper:"createdBy"`                // Creator ID
    CreatedAt       *time.Time `gorm:"created_at;autoCreateTime" mapper:"createdAt"` // Creation timestamp
    UpdatedBy       int64      `gorm:"updated_by" mapper:"updatedBy"`                // Updater ID
    UpdatedAt       *time.Time `gorm:"updated_at;autoUpdateTime" mapper:"updatedAt"` // Update timestamp
    DeletedUniqueId int64      `gorm:"deleted_unique_id" mapper:"deleteUniqueId"`    // Deletion flag. 0 indicates not deleted, 1 indicates deleted
    DeletedAt       *time.Time `gorm:"deleted_at" mapper:"deletedAt"`                // Deletion timestamp
}

func (u *Group) TableName() string {
    return "group"
}
