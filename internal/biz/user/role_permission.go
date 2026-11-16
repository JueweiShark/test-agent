package user

import (
    "context"
    
    "github.com/go-kratos/kratos/v2/errors"
    "github.com/go-kratos/kratos/v2/log"
    
    apiErr "wuwen.com/wwuc/api/errors"
    "wuwen.com/wwuc/internal/biz/common"
    "wuwen.com/wwuc/util/wuwen"
)

type RefRolePermission struct {
    Id         *int64 `mapper:"id"`
    RoleId     int64  `mapper:"roleId"`
    PermId     int64  `mapper:"permId"`
    App        int64  `mapper:"app"`
    PermCode   int64  `mapper:"permCode"`
    CreateBy   int64  `mapper:"createBy"`
    Permission *Permission
}
type RefRolePermissionRepo interface {
    CreateRolePermission(context.Context, []*RefRolePermission) error
    GetRolePermissionByRoleId(context.Context, []int64) ([]*RefRolePermission, error)
    GetRolePermissionByAppId(context.Context, int64) ([]*Permission, error)
    DeleteByRoleId(ctx context.Context, id, roleId int64) error
}

type RefRolePermissionUsecase struct {
    log *log.Helper
    tm  common.Transaction
    pr  PermissionRepo
    rpr RefRolePermissionRepo
}

func NewRefRolePermissionUsecase(logger log.Logger, tm common.Transaction,
    rpr RefRolePermissionRepo, pr PermissionRepo) *RefRolePermissionUsecase {
    return &RefRolePermissionUsecase{
        log: log.NewHelper(logger),
        tm:  tm,
        rpr: rpr,
        pr:  pr,
    }
}
func (uc *RefRolePermissionUsecase) FindRolePermissionInfo(ctx context.Context, roleIds []int64) (map[int64][]int64, *errors.Error) {
    rolePermissions, err := uc.rpr.GetRolePermissionByRoleId(ctx, roleIds)
    if err != nil {
        uc.log.WithContext(ctx).Errorf("find role permissions by ids fail, roleIds: %+v", roleIds)
        return nil, apiErr.UserPermissionFindFail
    }
    roleIdSet := wuwen.NewLinkedHashSet[int64]()
    for _, rolePermission := range rolePermissions {
        if rolePermission.Id == nil {
            uc.log.WithContext(ctx).Warnf("role permission id is nil. roleIds: %+v", roleIds)
            continue
        }
        roleIdSet.Add(rolePermission.PermId)
    }

    permissionIds := roleIdSet.GetOrderedElements()
    ps, err := uc.pr.FindByIds(ctx, permissionIds)
    if err != nil {
        uc.log.WithContext(ctx).Errorf("find  permissions by ids fail, permission ids: %+v", permissionIds)
        return nil, apiErr.UserPermissionFindFail
    }
    pMap := wuwen.SliceMapper2Map(ps, func(item *Permission) (int64, *Permission) {
        if item == nil {
            return 0, nil
        }
        return item.Id, item
    })
    var isAdmin bool
    for i, permission := range rolePermissions {
        isAdmin = isAdmin || permission.RoleId == 1
        if permission.Id == nil {
            uc.log.WithContext(ctx).Warnf("role permission id is nil. roleIds: %+v", roleIds)
            continue
        }
        p, exist := pMap[permission.PermId]
        if !exist {
            uc.log.WithContext(ctx).Warnf("role permission is nil. roleIds: %+v", roleIds)
            continue
        }
        rolePermissions[i].PermCode = p.PermissionCode
    }
    if isAdmin {
        rolePermissions = append(rolePermissions, &RefRolePermission{
            Id:       nil,
            RoleId:   1,
            PermId:   -1,
            PermCode: -1,
        })
    }
    return wuwen.SliceMapper2MapWithGroup(rolePermissions, func(item *RefRolePermission) (int64, int64) {
        if item == nil {
            return 0, 0
        }
        return item.RoleId, item.PermCode
    }), nil
}


func (uc *RefRolePermissionUsecase) FindEmbodiedAidRolePermissionInfo(ctx context.Context, roleIds []int64,appId int64) ([]int64, *errors.Error) {
    permissions, err := uc.rpr.GetRolePermissionByAppId(ctx, appId)
    if err != nil {
        uc.log.WithContext(ctx).Errorf("find role permissions by ids fail, roleIds: %+v", roleIds)
        return nil, apiErr.UserPermissionFindFail
    }
    res :=make([]int64,0)
    if len(permissions)>0{
        for _,v:=range permissions{
            res = append(res,v.PermissionCode)
        }
    }
    return res,nil
}

func (uc *RefRolePermissionUsecase) GetRolePermission(ctx context.Context, roleId int64) (map[int64][]int64, *errors.Error) {
    roleIds :=[]int64{roleId}
    rolePermissions, err := uc.rpr.GetRolePermissionByRoleId(ctx, roleIds)
    if err != nil {
        uc.log.WithContext(ctx).Errorf("find role permissions by ids fail, roleIds: %+v", roleIds)
        return nil, apiErr.UserPermissionFindFail
    }
    roleIdSet := wuwen.NewLinkedHashSet[int64]()
    for _, rolePermission := range rolePermissions {
        if rolePermission.Id == nil {
            uc.log.WithContext(ctx).Warnf("role permission id is nil. roleIds: %+v", roleIds)
            continue
        }
        roleIdSet.Add(rolePermission.PermId)
    }
    
    permissionIds := roleIdSet.GetOrderedElements()
    ps, err := uc.pr.FindByIds(ctx, permissionIds)
    if err != nil {
        uc.log.WithContext(ctx).Errorf("find  permissions by ids fail, permission ids: %+v", permissionIds)
        return nil, apiErr.UserPermissionFindFail
    }
    pMap := wuwen.SliceMapper2Map(ps, func(item *Permission) (int64, *Permission) {
        if item == nil {
            return 0, nil
        }
        return item.Id, item
    })
    var isAdmin bool
    for i, permission := range rolePermissions {
        isAdmin = isAdmin || permission.RoleId == 1
        if permission.Id == nil {
            uc.log.WithContext(ctx).Warnf("role permission id is nil. roleIds: %+v", roleIds)
            continue
        }
        p, exist := pMap[permission.PermId]
        if !exist {
            uc.log.WithContext(ctx).Warnf("role permission is nil. roleIds: %+v", roleIds)
            continue
        }
        rolePermissions[i].PermCode = p.PermissionCode
    }
    if isAdmin {
        rolePermissions = append(rolePermissions, &RefRolePermission{
            Id:       nil,
            RoleId:   1,
            PermId:   -1,
            PermCode: -1,
        })
    }
    return wuwen.SliceMapper2MapWithGroup(rolePermissions, func(item *RefRolePermission) (int64, int64) {
        if item == nil {
            return 0, 0
        }
        return item.RoleId, item.PermCode
    }), nil
}
