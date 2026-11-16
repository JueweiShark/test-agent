package dbupdate

import (
    "context"

    "github.com/go-kratos/kratos/v2/log"
    "go.uber.org/zap"

    "wuwen.com/wwuc/internal/data/common"
    "wuwen.com/wwuc/internal/data/schema"
)

var refs = NewRelation()

type Relation struct {
    log *log.Helper
}

func NewRelation() *Relation {
    return &Relation{
        log: log.NewHelper(log.With(log.DefaultLogger, "module", "dbupdate.relation")),
    }
}

func (uc *Relation) MethodName() string {
    return "loadRelation"
}
func (uc *Relation) Do(ctx context.Context, data *common.Data) bool {
    userGroup := getRefUserGroup()
    userRole := getRefUserRole()
    rolePermission := uc.getRefRolePermission(data)

    if userGroup == nil || userRole == nil || rolePermission == nil {
        uc.log.WithContext(ctx).Errorf("get ref data fail")
        return false
    }
    err := data.DB(ctx).Model(schema.RefUserGroup{}).Create(&userGroup).Error
    if err != nil {
        uc.log.WithContext(ctx).Errorf("create user group error:%v", err)
        return false
    }
    err = data.DB(ctx).Model(schema.RefUserRole{}).Create(&userRole).Error
    if err != nil {
        uc.log.WithContext(ctx).Errorf("create user role error:%v", err)
        return false
    }
    err = data.DB(ctx).Model(schema.RefRolePermission{}).Create(&rolePermission).Error
    if err != nil {
        uc.log.WithContext(ctx).Errorf("create role permission error:%v", err)
        return false
    }
    return true
}
func getRefUserRole() *schema.RefUserRole {
    return &schema.RefUserRole{
        UserId:  1,
        RoleId:  1,
        GroupId: 0,
    }
}
func getRefUserGroup() *schema.RefUserGroup {
    return &schema.RefUserGroup{
        UserId:  1,
        GroupId: 0,
    }
}
func (uc *Relation) getRefRolePermission(data *common.Data) []*schema.RefRolePermission {
    ctx, cancelFunc := context.WithCancel(context.Background())
    defer cancelFunc()

    permissions := new([]*schema.Permission)
    err := data.DB(ctx).Model(schema.Permission{}).Find(permissions, "deleted_unique_id = 0").Error
    if err != nil {
        uc.log.WithContext(ctx).Errorf("find role permissions fail %+v", zap.Error(err))
        return nil
    }
    rolePermissions := make([]*schema.RefRolePermission, len(*permissions))
    for i, permission := range *permissions {
        rolePermissions[i] = &schema.RefRolePermission{
            RoleId:       1,
            PermissionId: permission.Id,
        }
    }
    return rolePermissions
}
