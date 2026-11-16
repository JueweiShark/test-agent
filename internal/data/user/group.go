package user

import (
    "context"
    "fmt"
    "time"

    "github.com/go-kratos/kratos/v2/log"
    "gorm.io/gorm"

    "wuwen.com/wwuc/internal/biz/user"
    "wuwen.com/wwuc/internal/data/common"
    "wuwen.com/wwuc/internal/data/schema"
    "wuwen.com/wwuc/util/wuwen/mapper"
)

type groupRepo struct {
    data  *common.Data
    log   *log.Helper
    model schema.Group
}

func (r *groupRepo) FindByIds(ctx context.Context, id []int64) ([]*user.Group, error) {
    pos := new([]*schema.Group)
    db := r.data.DB(ctx).Model(r.model).
        Where("deleted_unique_id = 0 and id in (?)", id).
        Where("system = 1")
    err := db.Find(pos).Error
    if err != nil {
        r.log.WithContext(ctx).Errorf("find group error:%v", err)
        return nil, err
    }
    bos := new([]*user.Group)
    err = mapper.MapperSlice(pos, bos)
    if err != nil {
        r.log.WithContext(ctx).Errorf("mapper group error:%v", err)
        return nil, err
    }
    return *bos, nil
}

func (r *groupRepo) FindByManagers(ctx context.Context, id int64) ([]*user.Group, error) {
    pos := new([]*schema.Group)
    err := r.data.DB(ctx).
        Raw("SELECT * FROM `group` WHERE deleted_unique_id = 0 AND JSON_CONTAINS(managers, ?)",
            fmt.Sprintf("[%d]", id)).
        Scan(pos).Error
    if err != nil {
        r.log.WithContext(ctx).Errorf("find group error:%v", err)
        return nil, err
    }
    bos := new([]*user.Group)
    err = mapper.MapperSlice(pos, bos)
    if err != nil {
        r.log.WithContext(ctx).Errorf("mapper group error:%v", err)
        return nil, err
    }
    return *bos, nil
}

func (r *groupRepo) List(ctx context.Context, c user.GroupCondition, offset, limit int64) ([]*user.Group, int64, error) {
    pos := new([]*schema.Group)
    db := r.data.DB(ctx).Model(r.model).
        Where("deleted_unique_id = 0").
        Where("system = 1").
        Where("app = ?",c.AppId).
        Order("id desc")
    if c.Name != nil {
        db = common.QueryWithEscapedLike(db, "name", *c.Name)
    }
    if c.GroupType != nil {
        db = db.Where("group_type = ?", *c.GroupType)
    }
    if len(c.Ids) != 0 {
        db = db.Where("id IN (?)", c.Ids)
    }
    var total int64
    db.Count(&total)
    err := db.Offset(int(offset)).Limit(int(limit)).
        Find(pos).Error
    if err != nil {
        r.log.WithContext(ctx).Errorf("find group error:%v", err)
        return nil, 0, err
    }
    bos := new([]*user.Group)
    err = mapper.MapperSlice(pos, bos)
    if err != nil {
        r.log.WithContext(ctx).Errorf("mapper group error:%v", err)
        return nil, 0, err
    }
    return *bos, total, nil
}

func (r *groupRepo) ListAll(ctx context.Context, c user.FindGroupCondition) ([]*user.Group, error) {
    pos := new([]*schema.Group)
    db := r.data.DB(ctx).Model(r.model).
        Where("deleted_unique_id = 0").
        Where("system = 1").
        Where("app = ?", c.AppId).
        Order("id desc")
    if c.Name != "" {
        db = common.QueryWithEscapedLike(db, "name", c.Name)
    }
    if len(c.Ids) != 0 {
        db = db.Where("id IN (?)", c.Ids)
    }
    err := db.Find(pos).Error
    if err != nil {
        r.log.WithContext(ctx).Errorf("find group error:%v", err)
        return nil, err
    }
    bos := new([]*user.Group)
    err = mapper.MapperSlice(pos, bos)
    if err != nil {
        r.log.WithContext(ctx).Errorf("mapper group error:%v", err)
        return nil, err
    }
    return *bos, nil
}

func (r *groupRepo) Create(ctx context.Context, g *user.Group) error {
    sGroup := common.ConvertData[user.Group, schema.Group](ctx, g)
    err := r.data.DB(ctx).Model(r.model).Create(sGroup).Error
    if err != nil {
        r.log.WithContext(ctx).Errorf("create group error:%v", err)
        return err
    }
    g.Id = sGroup.Id
    return nil
}

func (r *groupRepo) First(ctx context.Context, id int64) (*user.Group, error) {
    group := new(schema.Group)
    err := r.data.DB(ctx).Model(r.model).
        Where("deleted_unique_id = 0").
        Where("id = ?", id).
        First(group).Error
    if err != nil {
        r.log.WithContext(ctx).Errorf("find group error:%v", err)
        return nil, err
    }
    biz := common.ConvertData[schema.Group, user.Group](ctx, group)
    return biz, nil
}

func (r *groupRepo) Update(ctx context.Context, g *user.Group) error {
    sGroup := common.ConvertData[user.Group, schema.Group](ctx, g)
    err := r.data.DB(ctx).Model(r.model).
        Where("deleted_unique_id = 0").
        Where("id = ?", sGroup.Id).Updates(sGroup).Error
    if err != nil {
        r.log.WithContext(ctx).Errorf("update group error:%v", err)
        return err
    }
    return nil
}

func (r *groupRepo) Delete(ctx context.Context, userId, groupId int64) error {
    err := r.data.DB(ctx).Model(r.model).Where("deleted_unique_id = 0 AND id = ?  ", groupId).
        Updates(map[string]interface{}{
            "updated_by":        userId,
            "deleted_at":        time.Now(),
            "deleted_unique_id": gorm.Expr("id"),
        }).Error
    if err != nil {
        r.log.WithContext(ctx).Errorf("delete group error:%v", err)
        return err
    }
    return nil
}

// NewGroupRepo .
func NewGroupRepo(data *common.Data, logger log.Logger) user.GroupRepo {
    return &groupRepo{
        data:  data,
        log:   log.NewHelper(logger),
        model: schema.Group{},
    }
}
