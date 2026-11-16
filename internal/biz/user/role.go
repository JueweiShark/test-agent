package user

import (
    "context"
    e "errors"
    "time"

    "github.com/go-kratos/kratos/v2/errors"
    "github.com/go-kratos/kratos/v2/log"
    "go.uber.org/zap"
    "gorm.io/gorm"

    apiErr "wuwen.com/wwuc/api/errors"
    "wuwen.com/wwuc/internal/biz/common"
    "wuwen.com/wwuc/util/wuwen"
)

var (
    defaultCode = []int64{12, 13, 120, 130, 131, 132, 133, 1201, 1202, 1203, 1301, 1302, 1303, 1304, 1305, 1306}
)
var (
    defaultEmbodiedCode []int64
)

type Role struct {
    Id              *int64     `json:"Id" mapper:"id"`                // Unique Identifier
    Name            string     `json:"name" mapper:"name"`            // Role name
    Remarks         string     `json:"remarks" mapper:"remarks"`            // remarks
    GroupId         int64      `json:"group_Id" mapper:"groupId"`     // role in which group
    App             int64      `json:"app" mapper:"app"`     // role in which group
    CreateBy        int64      `json:"create_by" mapper:"createBy"`   // Creator Id
    CreateAt        *time.Time `json:"create_at" mapper:"createAt"`   // Creation timestamp
    UpdateBy        int64      `json:"update_by" mapper:"updateBy"`   // Updater Id
    UpdateAt        *time.Time `json:"update_at" mapper:"updateAt"`   // Update timestamp
    IsDelete        int64      `json:"is_delete" mapper:"isDelete"`   // Deletion flag. 0 indicates not deleted, 1 indicates deleted
    DeletedAt       *time.Time `json:"deleted_at" mapper:"deletedAt"` // Deletion timestamp
    System          int64      `mapper:"System"`
    PermissionCodes []int64
    Creator         string
}
type RoleRepo interface {
    Create(ctx context.Context, r *Role) error
    BatchCreate(ctx context.Context, r []*Role) error
    First(ctx context.Context, id int64) (*Role, error)
    Update(ctx context.Context, r *Role) error
    Delete(ctx context.Context, userId, roleId int64) error
    List(ctx context.Context, groupId int64, name string, offset, limit int) ([]*Role, int64, error)
    ListByAppId(ctx context.Context, appid int64, name string, offset, limit int) ([]*Role, int64, error)
    FindByIds(ctx context.Context, groupId int64, ids []int64) ([]*Role, error)
    FindByIdsAndAppId(ctx context.Context, appId int64, ids []int64) ([]*Role, error)
    IsAdmin(ctx context.Context, id int64) ([]*Role, error)
    //FindUserRoleInGroup(ctx context.Context, groupId int64, userId int64) ([]*Role, error)
}
type RoleUsecase struct {
    log *log.Helper
    tm  common.Transaction
    rr  RoleRepo
    ur  UserRepo
    pr  PermissionRepo
    rpr RefRolePermissionRepo
    rur RefUserRoleRepo
}

func NewRoleUsecase(logger log.Logger, tm common.Transaction,
    ur UserRepo, rr RoleRepo, pr PermissionRepo,
    rur RefUserRoleRepo, rpr RefRolePermissionRepo,
) *RoleUsecase {
    return &RoleUsecase{
        log: log.NewHelper(logger),
        tm:  tm,
        rr:  rr,
        pr:  pr,
        ur:  ur,
        rpr: rpr,
        rur: rur,
    }
}
func (uc *RoleUsecase) Create(ctx context.Context, userId int64, r *Role) (int64, *errors.Error) {
    var roleId int64
    if len(r.PermissionCodes) == 0 {
        uc.log.WithContext(ctx).Error("permission code is empty")
       // return 0, apiErr.UserPermissionFindFail
    }
    pcs := make([]int64,0)
    if r.GroupId == 2{
        pcs = append(r.PermissionCodes, defaultEmbodiedCode...)
    }else{
        pcs = append(r.PermissionCodes, defaultCode...)
    }
    permissions, err := uc.findParent(ctx, pcs)
    if err != nil {
        uc.log.WithContext(ctx).Errorf("find role permissions fail %+v", zap.Error(err))
        return 0, apiErr.UserPermissionFindFail
    }
    err = uc.tm.InTx(ctx, func(ctx context.Context) error {
        err := uc.rr.Create(ctx, r) //插入role表
        if errors.Is(err, gorm.ErrDuplicatedKey) {
            uc.log.WithContext(ctx).Errorf("name duplicate")
            return apiErr.UserRoleDuplicate
        } else if err != nil {
            uc.log.WithContext(ctx).Errorf("create role failed, %+v", zap.Error(err))
            return apiErr.UserRoleCreateFail
        }
        roleId = *r.Id
        set := map[int64]struct{}{}
        rolePermissions := make([]*RefRolePermission, 0, len(permissions))
        for _, permission := range permissions {
            if _, exist := set[permission.PermissionCode]; exist {
                continue
            }
            set[permission.PermissionCode] = struct{}{}
            rolePermissions = append(rolePermissions, &RefRolePermission{
                RoleId:   roleId,
                PermId:   permission.Id,
                App:      r.App,
                CreateBy: userId,
            })
        }
        err = uc.rpr.CreateRolePermission(ctx, rolePermissions)
        if err != nil {
            uc.log.WithContext(ctx).Errorf("create role permission fail, roleId: %d, err: %s", roleId, err.Error())
            return apiErr.UserRoleCreateFail
        }
        return nil
    })
    var apiE *errors.Error
    if e.As(err, &apiE) {
        return roleId, apiE
    } else if err != nil {
        uc.log.WithContext(ctx).Errorf("create role failed, %+v", zap.Error(err))
        return 0, apiErr.UserRoleCreateFail.WithCause(err)
    }
    return roleId, nil
}

func (uc *RoleUsecase) Update(ctx context.Context, userId int64, r *Role) *errors.Error {
    var roleId int64
    if r.Id == nil {
        uc.log.WithContext(ctx).Errorf("update role failed, role Id is nil")
        return apiErr.UserRoleFindFail
    }
    roleId = *r.Id
    err := uc.tm.InTx(ctx, func(ctx context.Context) error {
        role, err := uc.rr.First(ctx, roleId)
        if err != nil {
            uc.log.WithContext(ctx).Errorf("find role failed, %+v", zap.Error(err))
            return apiErr.UserRoleFindFail
        }
        err = uc.rr.Update(ctx, r)

        if errors.Is(err, gorm.ErrDuplicatedKey) {
            uc.log.WithContext(ctx).Errorf("name duplicate")
            return apiErr.UserRoleDuplicate
        } else if err != nil {
            uc.log.WithContext(ctx).Errorf("update role failed, %+v", zap.Error(err))
            return apiErr.UserRoleUpdateFail
        }
        pcs := make([]int64,0)
        if r.GroupId == 2{
            pcs = append(r.PermissionCodes, defaultEmbodiedCode...)
        }else{
            pcs = append(r.PermissionCodes, defaultCode...)
        }
        if len(r.PermissionCodes)>0 && !uc.comparePermissionChange(pcs, role.PermissionCodes) {
            return nil
        }
        err = uc.rpr.DeleteByRoleId(ctx, userId, roleId)
        if err != nil {
            uc.log.WithContext(ctx).Errorf("delete role permissions fail %+v", zap.Error(err))
            return apiErr.UserRoleUpdateFail
        }
        uc.log.WithContext(ctx).Infof(" role %d bind permissions has change ,need update", roleId)
        permissions, err := uc.pr.FindByCode(ctx, pcs)
        if err != nil {
            uc.log.WithContext(ctx).Errorf("find role permissions fail %+v", zap.Error(err))
            return apiErr.UserPermissionFindFail
        }
        set := map[int64]struct{}{}
        rolePermissions := make([]*RefRolePermission, 0, len(permissions))
        for _, permission := range permissions {
            if _, exist := set[permission.PermissionCode]; exist {
                continue
            }
            set[permission.PermissionCode] = struct{}{}
            rolePermissions = append(rolePermissions, &RefRolePermission{
                RoleId:   roleId,
                PermId:   permission.Id,
                CreateBy: userId,
            })
        }
        err = uc.rpr.CreateRolePermission(ctx, rolePermissions)
        if err != nil {
            uc.log.WithContext(ctx).Errorf("create role permission fail, roleId: %d, err: %s", roleId, err.Error())
            return apiErr.UserRoleBindFail
        }
        return nil
    })
    if apiE, ok := err.(*errors.Error); ok {
        return apiE
    } else if err != nil {
        uc.log.WithContext(ctx).Errorf("create role failed, %+v", zap.Error(err))
        return apiErr.UserRoleCreateFail
    }
    return nil
}

func (uc *RoleUsecase) comparePermissionChange(old, new []int64) bool {
    if len(old) != len(new) {
        return true
    }
    oldMap := wuwen.SliceMapper2Map(old, func(t int64) (int64, struct{}) {
        return t, struct{}{}
    })
    var change bool
    for _, Id := range new {
        _, exist := oldMap[Id]
        change = change || !exist
    }
    return false
}

func (uc *RoleUsecase) Delete(ctx context.Context, userId, roleId int64) *errors.Error {
    refUserRole, err := uc.rur.FindByRole(ctx, roleId)
    if err != nil {
        uc.log.WithContext(ctx).Errorf("role bind info query fail, cant delete")
        return apiErr.UserRoleBindInfo
    }
    if len(refUserRole) != 0 {
        uc.log.WithContext(ctx).Errorf("role is used, cant delete")
        return apiErr.UserRoleInUsed
    }
    err = uc.rr.Delete(ctx, userId, roleId)
    if err != nil {
        uc.log.WithContext(ctx).Errorf("delete role failed, roleId: %d, err: %s", roleId, err.Error())
        return apiErr.UserRoleDeleteFail
    }
    return nil
}

func (uc *RoleUsecase) FindOne(ctx context.Context, id int64) (*Role, *errors.Error) {
    role, err := uc.rr.First(ctx, id)
    if err != nil {
        uc.log.WithContext(ctx).Errorf("find role failed, roleId: %d, err: %s", id, err.Error())
        return nil, apiErr.UserRoleFindFail
    }
    refRolePermissions, err := uc.rpr.GetRolePermissionByRoleId(ctx, []int64{*role.Id})
    if err != nil {
        uc.log.WithContext(ctx).Errorf("find role permissions failed, roleId: %d, err: %s", *role.Id, err.Error())
        return nil, apiErr.UserRoleFindFail
    }
    pids := wuwen.SliceMapper(refRolePermissions, func(item *RefRolePermission) int64 {
        return item.PermId
    })
    ps, err := uc.pr.FindByIds(ctx, pids)
    if err != nil {
        uc.log.WithContext(ctx).Errorf("find  permissions fail %+v", zap.Error(err))
        return nil, apiErr.UserRoleFindFail
    }
    codes := wuwen.SliceMapper(ps, func(item *Permission) int64 {
        return item.PermissionCode
    })
    role.PermissionCodes = codes
    return role, nil
}

func (uc *RoleUsecase) List(ctx context.Context, groupId int64, appId int64, name string) ([]*Role, *errors.Error) {
    var list []*Role
    var err error

    // 兼容旧的用户角色 和 具身用户角色
    if appId >= 2 {
        list, _, err = uc.rr.ListByAppId(ctx, appId, name, -1, -1)
    } else {
        // 否则使用 groupId 查询
        list, _, err = uc.rr.List(ctx, groupId, name, -1, -1)
    }
    if err != nil {
        uc.log.WithContext(ctx).Errorf("list role failed, %s", err.Error())
        return nil, apiErr.UserRoleFindFail
    }
    set := wuwen.NewLinkedHashSet[int64]()
    for _, role := range list {
        set.Add(role.CreateBy)
    }
    creatorIds := set.GetOrderedElements()
    if len(creatorIds) == 0 {
        return list, nil
    }
    roleIds := wuwen.SliceMapper(list, func(t *Role) int64 {
        if t == nil {
            return 0
        }
        return *t.Id
    })
    refRolePermissions, err := uc.rpr.GetRolePermissionByRoleId(ctx, roleIds)
    if err != nil {
        uc.log.WithContext(ctx).Errorf("find role permissions failed, roleId: %d, err: %s", roleIds, err.Error())
        return nil, apiErr.UserRoleFindFail
    }
    pids := wuwen.SliceMapper(refRolePermissions, func(item *RefRolePermission) int64 {
        return item.PermId
    })
    ps, err := uc.pr.FindByIds(ctx, pids)
    if err != nil {
        uc.log.WithContext(ctx).Errorf("find  permissions fail %+v", zap.Error(err))
        return nil, apiErr.UserRoleFindFail
    }
    permissionMap := wuwen.SliceMapper2Map(ps, func(item *Permission) (int64, int64) {
        return item.Id, item.PermissionCode
    })
    rpMap := wuwen.SliceMapper2MapWithGroup(refRolePermissions, func(item *RefRolePermission) (int64, int64) {
        return item.RoleId, item.PermId
    })
    for i, role := range list {

        pids := rpMap[*role.Id]
        for _, pid := range pids {
            list[i].PermissionCodes = append(list[i].PermissionCodes, permissionMap[pid])
        }

    }
    return list, nil
}
func (uc *RoleUsecase) RolePermission(ctx context.Context,groupId, roleId int64) ([]int64, *errors.Error) {
    if roleId == 0 {
        return nil, nil
    }
    list,  err := uc.rr.FindByIds(ctx, groupId, []int64{roleId})
    if err != nil {
        uc.log.WithContext(ctx).Errorf("list role failed, %s", err.Error())
        return nil, apiErr.UserRoleFindFail
    }
    set := wuwen.NewLinkedHashSet[int64]()
    for _, role := range list {
        set.Add(role.CreateBy)
    }
    creatorIds := set.GetOrderedElements()
    if len(creatorIds) == 0 {
        return nil, nil
    }
    roleIds := wuwen.SliceMapper(list, func(t *Role) int64 {
        if t == nil {
            return 0
        }
        return *t.Id
    })
    refRolePermissions, err := uc.rpr.GetRolePermissionByRoleId(ctx, roleIds)
    if err != nil {
        uc.log.WithContext(ctx).Errorf("find role permissions failed, roleId: %d, err: %s", roleIds, err.Error())
        return nil, apiErr.UserRoleFindFail
    }
    pids := wuwen.SliceMapper(refRolePermissions, func(item *RefRolePermission) int64 {
        return item.PermId
    })
    ps, err := uc.pr.FindByIds(ctx, pids)
    if err != nil {
        uc.log.WithContext(ctx).Errorf("find  permissions fail %+v", zap.Error(err))
        return nil, apiErr.UserRoleFindFail
    }
    permissionMap := wuwen.SliceMapper2Map(ps, func(item *Permission) (int64, int64) {
        return item.Id, item.PermissionCode
    })
    rpMap := wuwen.SliceMapper2MapWithGroup(refRolePermissions, func(item *RefRolePermission) (int64, int64) {
        return item.RoleId, item.PermId
    })
    for i, role := range list {
        
        pids := rpMap[*role.Id]
        for _, pid := range pids {
            list[i].PermissionCodes = append(list[i].PermissionCodes, permissionMap[pid])
        }
        
    }
    if len(list)>0{
        return list[0].PermissionCodes,nil
    }
    return nil, nil
}

func (uc *RoleUsecase) findParent(ctx context.Context, code []int64) ([]*Permission, error) {
    permissions, err := uc.pr.FindByCode(ctx, code)
    if err != nil {
        uc.log.WithContext(ctx).Errorf("find permision by code fail %+v", err)
        return nil, err
    }
    if len(permissions) == 0 {
        uc.log.WithContext(ctx).Info("permission is empty, end")
        return nil, nil
    }
    var parentCode []int64
    for _, permission := range permissions {
        parentCode = append(parentCode, permission.Parents)
    }
    cp, err := uc.findParent(ctx, parentCode)
    if err != nil {
        uc.log.WithContext(ctx).Errorf("find permision by code fail %+v", err)
        return permissions, nil
    }

    permissions = append(permissions, cp...)

    return permissions, nil
}
