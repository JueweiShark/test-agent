package dbupdate

import (
    "context"

    "github.com/go-kratos/kratos/v2/log"

    "wuwen.com/wwuc/internal/data/common"
    "wuwen.com/wwuc/internal/data/schema"
)

var role = NewRole()

type Role struct {
    log *log.Helper
}

func NewRole() *Role {
    return &Role{
        log: log.NewHelper(log.With(log.DefaultLogger, "module", "dbupdate.role")),
    }
}
func (uc *Role) MethodName() string {
    return "loadRole"
}

func (uc *Role) Do(ctx context.Context, data *common.Data) bool {
    role := getRoleData()
    err := data.DB(ctx).Model(schema.Role{}).Create(role).Error
    if err != nil {
        uc.log.WithContext(ctx).Errorf("create role error:%v", err)
        return false
    }
    uc.log.WithContext(ctx).Infof("create user success,role name :%s", role.Name)
    return true
}

func getRoleData() *schema.Role {

    var id int64 = 1
    return &schema.Role{
        Id:        &id,
        Name:      "SA",
        GroupId:   0,
        CreatedBy: 0,
        System:    1,
    }
}
