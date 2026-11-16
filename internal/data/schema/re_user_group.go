package schema

import (
    "time"
)

// RefUserGroup 用户-企业关系表 schema.
// 字段类型与 biz/v2.RefUserGroup 对齐 (指针语义), 便于 common.ConvertData / mapper.MapperSlice 双向转换.
// 与 role.go / group.go / user.go 等 v2 schema 风格保持一致.
type RefUserGroup struct {
    Id              *int64     `gorm:"column:id;primary_key;AUTO_INCREMENT;comment:'Unique identifier'" mapper:"id"`
    UserId          int64      `gorm:"column:user_id;NOT NULL;comment:'User ID'" mapper:"userId"`
    GroupId         int64      `gorm:"column:group_id;NOT NULL;comment:'Group ID'" mapper:"groupId"`
    Status          int64      `gorm:"column:status;default:1;NOT NULL;comment:'0:禁用 1:启用'" mapper:"status"`
    Remarks         string     `gorm:"column:remarks;comment:'备注信息'" mapper:"remarks"`
    CreatedTime     *time.Time `gorm:"column:created_time;default:CURRENT_TIMESTAMP;NOT NULL;comment:'Creation timestamp';autoCreateTime" mapper:"createdTime"`
    CreatedBy       int64      `gorm:"column:created_by;NOT NULL;comment:'Creator ID'" mapper:"createdBy"`
    UpdatedTime     *time.Time `gorm:"column:updated_time;default:CURRENT_TIMESTAMP;NOT NULL;comment:'Update timestamp';autoUpdateTime" mapper:"updatedTime"`
    UpdatedBy       *int64     `gorm:"column:updated_by;comment:'Updater ID'" mapper:"updatedBy"`
    DeletedUniqueId int64      `gorm:"column:deleted_unique_id;default:0;NOT NULL;comment:'Deletion flag. 0 indicates not deleted, 1 indicates deleted'" mapper:"isDeleted"`
    DeletedAt       *time.Time `gorm:"column:deleted_at;default:null;comment:'delete timestamp'" mapper:"deletedTime"`
}

func (r *RefUserGroup) TableName() string {
    return "ref_user_group"
}
