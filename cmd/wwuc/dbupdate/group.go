package dbupdate

import (
    "context"

    "github.com/go-kratos/kratos/v2/log"

    "wuwen.com/wwuc/internal/data/common"
    "wuwen.com/wwuc/internal/data/schema"
)

var group = NewGroup()

type Group struct {
    log *log.Helper
}

func NewGroup() *Group {
    return &Group{
        log: log.NewHelper(log.With(log.DefaultLogger, "module", "dbupdate.group")),
    }
}
func (uc *Group) MethodName() string {
    return "loadGroup"
}

func (uc *Group) Do(ctx context.Context, data *common.Data) bool {
    group := getGroupData()
    err := data.DB(ctx).Model(schema.Group{}).Create(group).Error
    if err != nil {
        uc.log.WithContext(ctx).Errorf("create user group :%v", err)
        return false
    }
    uc.log.WithContext(ctx).Infof("create user success,group name :%s", group.Name)
    err = data.DB(ctx).Model(schema.Group{}).Where("id = 1").UpdateColumn("id", 0).Error
    if err != nil {
        uc.log.WithContext(ctx).Errorf("update group id error:%v", err)
        return false
    }
    return true
}

func getGroupData() *schema.Group {

    return &schema.Group{
        System: 1,
        Name:   "SA-Group",
    }
}
