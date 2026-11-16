package schema

import "time"

type RefUserRole struct {
    Id              *int64     `gorm:"id" mapper:"id"`
    UserId          int64      `gorm:"user_id" mapper:"userId"`
    RoleId          int64      `gorm:"role_id" mapper:"roleId"`
    GroupId         int64      `mapper:"groupId" gorm:"column:group_id"`
    CreatedBy       int64      `gorm:"created_by" mapper:"createBy"`
    CreatedAt       *time.Time `gorm:"created_at;autoCreateTime" mapper:"createAt"` // Creation timestamp
    UpdatedBy       int64      `gorm:"updated_by" mapper:"updateBy"`                // Updater ID
    UpdatedAt       *time.Time `gorm:"updated_at;autoUpdateTime" mapper:"updateAt"` // Update timestamp
    DeletedUniqueId int64      `gorm:"deleted_unique_id" mapper:"isDelete"`         // Deletion flag. 0 indicates not deleted, 1 indicates deleted
    DeletedAt       *time.Time `gorm:"deleted_at" mapper:"deletedAt"`               // Deletion timestamp
}

func (t RefUserRole) TableName() string {
    return "ref_user_role"
}
