package schema

import "time"

type RefRolePermission struct {
    Id           *int64     `gorm:"id" mapper:"id"`
    RoleId       int64      `gorm:"role_id" mapper:"roleId"`
    PermissionId int64      `gorm:"permission_id" mapper:"permId"`
    CreatedBy    int64      `gorm:"created_by" mapper:"createBy"`
    CreatedAt    *time.Time `gorm:"created_at;autoCreateTime" mapper:"createAt"` // Creation timestamp
    UpdatedBy    int64      `gorm:"updated_by" mapper:"updateBy"`                // Updater ID
    UpdatedAt    *time.Time `gorm:"updated_at;autoUpdateTime" mapper:"updateAt"` // Update timestamp
    isDeleted    int64      `gorm:"is_deleted" mapper:"isDelete"`                // Deletion flag. 0 indicates not deleted, 1 indicates deleted
    DeletedAt    *time.Time `gorm:"deleted_at" mapper:"deletedAt"`               // Deletion timestamp
}

func (t RefRolePermission) TableName() string {
    return "ref_role_permission"
}
