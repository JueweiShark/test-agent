package schema

import (
	"time"
)

type User struct {
	Id             *int64     `gorm:"column:id;primary_key;AUTO_INCREMENT;comment:'primary key'" mapper:"id"`
	Username       string     `gorm:"column:username;NOT NULL;comment:'login username'" mapper:"username"`
	Password       string     `gorm:"column:password;NOT NULL;comment:'encode password'" mapper:"password"`
	Nickname       string     `gorm:"column:nickname;comment:'user nickname'" mapper:"nickname"`
	CreatedAt      time.Time  `gorm:"column:created_at;NOT NULL;comment:'user created time';<-:create" mapper:"createdAt"`
	CreatedBy      int64      `gorm:"column:created_by;default:0;comment:'the person create '" mapper:"createdBy"`
	UpdatedAt      *time.Time `gorm:"column:updated_at;comment:'user updated time';autoUpdateTime" mapper:"updatedAt"`
	UpdatedBy      int64      `gorm:"column:updated_by;default:0;comment:'the person create '" mapper:"updatedBy"`
	LastLoginAt    *time.Time `gorm:"column:last_login_at;comment:'last login time'" mapper:"lastLoginAt"`
	DeleteUniqueId int64      `gorm:"column:deleted_unique_id;default:0;comment:'this user is deleted default 0'" mapper:"uniqueDeletedKey"`
	DeletedAt      *time.Time `gorm:"column:deleted_at;comment:'user delete time'" mapper:"deletedAt"`
	Provider       string     `gorm:"column:provider;comment:'user provider'" mapper:"provider"`
	App            int64      `gorm:"column:app;default:0;comment:'1:仿真 2:具身'" mapper:"app"`
	Status         int64      `gorm:"column:status;default:1;comment:'0:禁用 1:启用'" mapper:"status"`
	IsAll          int64      `gorm:"column:is_all;default:0;comment:'是否全平台0否 1是'" mapper:"isAll"`
	Remarks        string     `gorm:"column:remarks;NOT NULL;default:'';comment:'备注'" mapper:"remarks"`
}

func (u *User) TableName() string {
	return "user"
}
