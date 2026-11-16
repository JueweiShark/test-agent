package user

import (
    "context"
    "fmt"
    "testing"

    "github.com/go-kratos/kratos/v2/log"

    "wuwen.com/wwuc/internal/conf"
    "wuwen.com/wwuc/internal/data/common"
    "wuwen.com/wwuc/internal/data/schema"
    "wuwen.com/wwuc/util/wuwen/mapper"
)

type permissionRepo struct {
    data  *common.Data
    model schema.Permission
    log   *log.Helper
}

func (r *permissionRepo) Create(ctx context.Context, p []*Permission) error {
    //TODO implement me
    panic("implement me")
}

func (r *permissionRepo) Update(ctx context.Context, p *Permission) error {
    //TODO implement me
    panic("implement me")
}

func (r *permissionRepo) Delete(ctx context.Context, userId int, id int64) error {
    //TODO implement me
    panic("implement me")
}

func (r *permissionRepo) FindByIds(ctx context.Context, ids []int64) ([]*Permission, error) {
    //TODO implement me
    panic("implement me")
}

func (r *permissionRepo) FindByPage(ctx context.Context, offset int64, limit int64) ([]*Permission, error) {
    //TODO implement me
    panic("implement me")
}

func (r *permissionRepo) FindByUser(ctx context.Context, id int64) ([]*Permission, error) {
    //TODO implement me
    panic("implement me")
}

func (r *permissionRepo) FindByCode(ctx context.Context, codes []int64) ([]*Permission, error) {
    pos := new([]*schema.Permission)
    err := r.data.DB(ctx).Model(r.model).
        Find(pos, "deleted_unique_id = 0 and code IN (?)  ", codes).Error
    if err != nil {
        r.log.WithContext(ctx).Errorf("find permission error:%v", err)
        return nil, err
    }
    bos := new([]*Permission)
    err = mapper.MapperSlice(pos, bos)
    if err != nil {
        r.log.WithContext(ctx).Errorf("mapper permission error:%v", err)
        return nil, err
    }
    return *bos, nil
}

// NewPermissionRepo .
func NewPermissionRepo(data *common.Data, logger log.Logger) PermissionRepo {
    return &permissionRepo{
        data:  data,
        log:   log.NewHelper(logger),
        model: schema.Permission{},
    }
}

func TestFindParent(t *testing.T) {
    db := common.NewDB(&conf.Bootstrap{
        Data: &conf.Data{
            Database: &conf.Data_Database{
                Driver:         "mysql",
                Source:         "root:ImOxO8Lz@tcp(192.168.2.49:8191)/wwuc?charset=utf8mb4&parseTime=True&loc=Local",
                MaxConnThreads: 100,
                MaxLifeMinute:  30,
            },
            Redis: nil,
        },
    }, log.DefaultLogger)
    data, _, _ := common.NewData(db, nil, log.DefaultLogger)
    ctx := context.Background()
    usecase := NewRoleUsecase(log.DefaultLogger, nil, nil, nil, NewPermissionRepo(data, log.DefaultLogger), nil, nil)
    parent, _ := usecase.findParent(ctx, []int64{1122})
    for _, permission := range parent {
        fmt.Print(permission.PermissionCode, "\t")
    }
    parent, _ = usecase.findParent(ctx, []int64{112})
    for _, permission := range parent {
        fmt.Print(permission.PermissionCode, "\t")
    }
    parent, _ = usecase.findParent(ctx, []int64{11})
    for _, permission := range parent {
        fmt.Print(permission.PermissionCode, "\t")
    }
    parent, _ = usecase.findParent(ctx, []int64{999})
    for _, permission := range parent {
        fmt.Print(permission.PermissionCode, "\t")
    }

}
