package schema

import (
    "time"
)

type Permission struct {
    Id                 int64      `gorm:"column:id;primary_key;AUTO_INCREMENT;comment:'Unique identifier'" mapper:"id"`
    Domain             string     `mapper:"domain" gorm:"column:domain;not null"`
    Code               int64      `gorm:"column:code;default:0;NOT NULL;comment:'permission_id in fe'" mapper:"permissionCode"`
    Type               uint8      `gorm:"column:type;NOT NULL;default:0;comment:'Resource type'" mapper:"type"` // 0 be 1fe 2non
    MenuType           *uint8     `gorm:"column:menu_type;default:null;comment:'Menu type 1=dir 2=menu 3=button, only valid when type=1'" mapper:"menuType"` // 菜单类型 1=目录 2=菜单 3=按钮, 仅 type=1 时有效
    RoutePath          string     `gorm:"column:route_path;default:'';comment:'Frontend route path'" mapper:"routePath"`     // 前端路由路径
    Icon               string     `gorm:"column:icon;default:'';comment:'Menu icon'" mapper:"icon"`                       // 菜单图标
    Sort               int32      `gorm:"column:sort;default:0;comment:'Sort order'" mapper:"sort"`                       // 排序
    Hidden             *uint8     `gorm:"column:hidden;NOT NULL;default:1;comment:'1显示 0隐藏'" mapper:"hidden"`      // 1显示 0隐藏
    App                int64      `gorm:"column:app;default:0;comment:'App ID 1=sim 2=embodied 3=embodiedGen'" mapper:"app"` // 区分平台
    Name               string     `gorm:"column:name;NOT NULL;comment:'Resource name'" mapper:"name"`
    Path               string     `gorm:"column:path;NOT NULL;comment:'Resource URL'" mapper:"path"`
    Act                *string    `gorm:"column:act;default:null" mapper:"act"`
    Description        *string    `gorm:"column:description;comment:'description for permission'" mapper:"description"`
    DependenceChildren Int64Array `gorm:"column:dependence_children;comment:'dependence_children'" mapper:"dependenceChildren"`
    DependenceParents  int64      `gorm:"column:dependence_parents;comment:'dependence_parents'" mapper:"dependenceParents"`
    CreateBy           int64      `gorm:"column:create_by;comment:'Creator ID'" mapper:"createBy"`
    CreateTime         *time.Time `gorm:"column:create_time;default:CURRENT_TIMESTAMP;NOT NULL;comment:'Creation timestamp'" mapper:"createTime"`
    UpdateBy           int64      `gorm:"column:update_by;comment:'Updater ID'" mapper:"updateBy"`
    UpdateTime         *time.Time `gorm:"column:update_time;default:CURRENT_TIMESTAMP;NOT NULL;comment:'Update timestamp'" mapper:"updateTime"`
    DeletedUniqueId    int64      `gorm:"column:deleted_unique_id;default:0;NOT NULL;comment:'Deletion flag. 0 indicates not deleted, 1 indicates deleted'" mapper:"deletedId"`
    DeleteTime         *time.Time `gorm:"column:delete_time;comment:'Deletion timestamp'" mapper:"deleteTime"`
}

func (p *Permission) TableName() string {
    return "permission"
}
