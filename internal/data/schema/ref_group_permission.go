package schema

import "time"

// RefGroupPermission 企业权限范围关系表 (父域: 企业能看到的全部菜单权限)
// 替代 ref_group_app, 企业与产品解耦后改为企业↔权限关联
type RefGroupPermission struct {
    Id              *int64     `gorm:"column:id;primary_key;AUTO_INCREMENT" mapper:"id"`
    GroupId         int64      `gorm:"column:group_id;not null;comment:'租户id'" mapper:"groupId"`         // 租户id
    PermissionId    int64      `gorm:"column:permission_id;not null;comment:'菜单权限id'" mapper:"permissionId"` // 菜单权限id
    CreatedBy       int64      `gorm:"column:created_by" mapper:"createdBy"`
    CreatedAt       *time.Time `gorm:"column:created_at;autoCreateTime" mapper:"createdAt"`
    UpdatedBy       int64      `gorm:"column:updated_by" mapper:"updatedBy"`
    UpdatedAt       *time.Time `gorm:"column:updated_at;autoUpdateTime" mapper:"updatedAt"`
    DeletedUniqueId int64      `gorm:"column:deleted_unique_id;default:0" mapper:"deleteUniqueId"`
    DeletedAt       *time.Time `gorm:"column:deleted_at" mapper:"deletedAt"`
}

func (r *RefGroupPermission) TableName() string {
    return "ref_group_permission"
}
