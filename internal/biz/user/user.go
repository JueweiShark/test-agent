package user

import (
    "context"
    e "errors"
    "regexp"
    "time"
    
    "github.com/go-kratos/kratos/v2/errors"
    "github.com/go-kratos/kratos/v2/log"
    "golang.org/x/crypto/bcrypt"
    "gorm.io/gorm"
    
    apiErr "wuwen.com/wwuc/api/errors"
    "wuwen.com/wwuc/internal/biz/common"
    "wuwen.com/wwuc/util/wuwen"
)

const (
    KeyUserProviderWuwen      = "wuwen"
    KeyUserProviderEmbodiedAi = "embodiedAi"
    KeyUserProviderEmbodiedGen = "embodiedGen"
    
    roleAdminId    = 1
    roleOrgAdminId = 2
    roleNormalId   = 3
    roleDefaultId  = 4
    systemGroupId  = 1
)

var ExcludeCode = []int64{
    2030,
    10700,
    10800,
}

type LoginRep struct {
    User
    Token string `json:"token"`
}

type UserCondition struct {
    NickName   *string
    Username   *string
    Status     *int32
    Ids        []int64
    ExcludeIds []int64
    App        int64 `json:"app" mapper:"app"`
}

type UserSearchCondition struct {
    NickName string
    Username string
    Ids      []int64
    AppId    int64
}
type User struct {
    Id               *int64     `json:"id"  mapper:"id"`                              // primary key
    Username         string     `json:"username" mapper:"username"`                   // login username
    Password         string     `json:"password" mapper:"password"`                   // encode password
    Nickname         string     `json:"nickname" mapper:"nickname"`                   // user nickname
    CreatedAt        time.Time  `json:"created_at" mapper:"createdAt"`                // user created time
    CreatedBy        int64      `json:"created_by" mapper:"createdBy"`                // the person create
    UpdatedAt        *time.Time `json:"updated_at" mapper:"updatedAt"`                // user updated time
    UpdatedBy        int64      `json:"updated_by" mapper:"updatedBy"`                // the person create
    LastLoginAt      *time.Time `json:"last_login_at" mapper:"lastLoginAt"`           // last login time
    UniqueDeletedKey int64      `json:"unique_deleted_key" mapper:"uniqueDeletedKey"` // this user is deleted default 0
    DeletedAt        *time.Time `json:"deleted_at" mapper:"deletedAt"`                // user delete time
    Provider         string     `json:"provider" mapper:"provider"`                   // user provider
    PermissionCode   []int64
    Role             []*Role
    Groups           []*Group
    App              int64 `json:"app" mapper:"app"`       // 所属项目 1:仿真 2:具身
    Status           int64 `json:"status" mapper:"status"` // 状态 0:禁用 1:启用
}
type UserUsecase struct {
    tm      common.Transaction
    log     *log.Helper
    ur      UserRepo
    rr      RoleRepo
    gr      GroupRepo
    pr      PermissionRepo
    rur     RefUserRoleRepo
    rug     RefUserGroupRepo
    tokenUc *TokenUsecase
    rpUc    *RefRolePermissionUsecase
}

// NewUserUsecase new a user usecase.
func NewUserUsecase(logger log.Logger, tm common.Transaction, ur UserRepo,
    rur RefUserRoleRepo, rr RoleRepo, gr GroupRepo, rug RefUserGroupRepo,
    tokenUc *TokenUsecase, rpUc *RefRolePermissionUsecase, pr PermissionRepo) *UserUsecase {
    return &UserUsecase{
        log:     log.NewHelper(logger),
        tm:      tm,
        ur:      ur,
        rr:      rr,
        gr:      gr,
        rur:     rur,
        rug:     rug,
        tokenUc: tokenUc,
        rpUc:    rpUc,
        pr:      pr,
    }
}
func (uc *UserUsecase) CreateOne(ctx context.Context, u *User, groupId int64, roleIds []int64) *errors.Error {
    //reqCtx := common.ReqCtxFromContext(ctx)
    //createdBy := reqCtx.GetUserId()
    apiE := uc.generatePwd(ctx, u)
    if apiE != nil {
        return apiE
    }
    log.Info("password generate ok")
    //userRoles := make([]*RefUserRole, len(roleIds), 0)
    //userId := *u.Id
    //if groupId > 0 {
    //    _, err := uc.gr.First(ctx, groupId)
    //    if err != nil {
    //        uc.log.WithContext(ctx).Errorf("groupId is zero")
    //        return apiErr.UserGroupFindFail
    //    }
    //    if len(roleIds) > 0 {
    //        _, err = uc.rr.FindByIds(ctx, groupId, roleIds)
    //        if err != nil {
    //            uc.log.WithContext(ctx).Errorf("find role failed, %v", err)
    //            return apiErr.UserRoleFindFail
    //        }
    //        for _, roleId := range roleIds {
    //            userRoles = append(userRoles, &RefUserRole{
    //                UserId:   userId,
    //                RoleId:   roleId,
    //                CreateBy: createdBy,
    //                GroupId:  groupId,
    //            })
    //        }
    //    }
    //
    //}
    
    err := uc.tm.InTx(ctx, func(ctx context.Context) error {
        err := uc.ur.Create(ctx, u)
        if errors.Is(err, gorm.ErrDuplicatedKey) {
            uc.log.WithContext(ctx).Errorf("create user failed,user name dulicate %v", err)
            return apiErr.UserDuplicate
        } else if err != nil {
            uc.log.WithContext(ctx).Errorf("create user failed, %v", err)
            return apiErr.UserRegisterFail
        }
        if u.Id == nil {
            return apiErr.UserRegisterFail
        }
        //userId := *u.Id
        //if groupId > 0 {
        //    err = uc.rug.BatchCreate(ctx, []*RefUserGroup{{
        //        UserId:    userId,
        //        GroupId:   groupId,
        //        CreatedBy: createdBy,
        //    }})
        //    if len(userRoles) > 0 {
        //        err = uc.rur.BatchCreate(ctx, userRoles)
        //        if err != nil {
        //            uc.log.WithContext(ctx).Errorf("bind user role  failed, %v", err)
        //            return apiErr.UserRoleCreateFail
        //        }
        //    }
        //}
        //插入角色信息
        if len(roleIds) > 0 && u.App >= 2 {
            userRoles := make([]*RefUserRole, 0)
            for _, roleId := range roleIds {
                var roleInfo RefUserRole
                roleInfo.RoleId = roleId
                roleInfo.UserId = *u.Id
                userRoles = append(userRoles, &roleInfo)
            }
            err = uc.rur.BatchCreate(ctx, userRoles)
            if err != nil {
                uc.log.WithContext(ctx).Errorf("bind user role  failed, %v", err)
                return apiErr.UserRoleCreateFail
            }
        }
        
        if err != nil {
            uc.log.WithContext(ctx).Errorf("bind user group failed, %v", err)
            return apiErr.GroupAddMemberFail
        }
        return nil
    })
    
    if err != nil {
        return err.(*errors.Error)
    }
    return nil
}

func (uc *UserUsecase) generatePwd(ctx context.Context, u *User) *errors.Error {
    bytes, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
    if err != nil {
        uc.log.WithContext(ctx).Error("bcrypt generate password failed", u.Username, u.Password)
        return apiErr.UserRegisterFail
    }
    err = bcrypt.CompareHashAndPassword(bytes, []byte(u.Password))
    if err != nil {
        uc.log.WithContext(ctx).Info("password failed", u.Password)
        return apiErr.UserRegisterFail
    }
    u.Password = string(bytes)
    return nil
}

func (uc *UserUsecase) Update(ctx context.Context, u *User) *errors.Error {
    var needUpdate bool
    pwdChanged := false
    if len(u.Password) > 0 {
        needUpdate = true
        pwdChanged = true
        if !uc.checkPwd(ctx, u.Password) {
            uc.log.WithContext(ctx).Errorf("password check failed, %v", u.Password)
            return apiErr.ParamsInvalid
        }
        apiE := uc.generatePwd(ctx, u)
        if apiE != nil {
            return apiE
        }
    }
    if len(u.Nickname) > 0 {
        needUpdate = true
    }
    if needUpdate {
        err := uc.ur.Update(ctx, u)
        if err != nil {
            uc.log.WithContext(ctx).Errorf("create user failed, %v", err)
            return apiErr.UserUpdateFail
        }
        //修改密码后删除该用户全部token，强制重新登录
        if pwdChanged && u.Id != nil {
            if err := uc.tokenUc.DeleteUserTokens(ctx, *u.Id); err != nil {
                uc.log.WithContext(ctx).Errorf("delete user tokens failed, userId:%v, err:%v", *u.Id, err)
            }
        }
    }
    return nil
}

func (uc *UserUsecase) compareRoleChange(old, new []int64) bool {
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
    return change
}
func (uc *UserUsecase) Delete(ctx context.Context, userId, delUserId int64) *errors.Error {
    
    err := uc.tm.InTx(ctx, func(ctx context.Context) error {
        //删除用户和权限的关联表的数据
        err := uc.rur.BatchDelete(ctx, userId, delUserId, 0)
        if err != nil {
            uc.log.WithContext(ctx).Errorf("delete user role ref failed, %v", err)
            return apiErr.UserDeleteFail
        }
        //删除用户和分组的关联表的数据
        err = uc.rug.BatchDeleteByIds(ctx, userId, delUserId)
        if err != nil {
            uc.log.WithContext(ctx).Errorf("delete user group ref failed, %v", err)
            return apiErr.UserDeleteFail
        }
        //删除用户表的数据
        err = uc.ur.Delete(ctx, userId, delUserId)
        if err != nil {
            uc.log.WithContext(ctx).Errorf("delete user failed, %v", err)
            return apiErr.UserDeleteFail
        }
        return nil
    })
    var apiE *errors.Error
    if e.As(err, &apiE) {
        return apiE
    } else if err != nil {
        return apiErr.UserDeleteFail
    }
    return nil
}

func (uc *UserUsecase) Forbidden(ctx context.Context, userId, status int64) *errors.Error {
    err := uc.ur.Forbidden(ctx, userId, status)
    if err != nil {
        uc.log.WithContext(ctx).Errorf("Forbidden user failed, %v", err)
        return apiErr.UserForbiddenFail
    }
    return nil
}

func (uc *UserUsecase) UserInfo(ctx context.Context, id int64, checkMatch bool) (*User, *errors.Error) {
    reqCtx := common.ReqCtxFromContext(ctx)
    if reqCtx.GetUserId() != id && checkMatch {
        uc.log.WithContext(ctx).Errorf("token sub id not match input user id")
        return nil, apiErr.TokenNotMatch
    }
    userInfo, err := uc.ur.FindOne(ctx, id)
    if err != nil {
        uc.log.WithContext(ctx).Errorf("delete user failed, %v", err)
        return nil, apiErr.UserFindFail
    }
    //获取用户的权限信息
    if userInfo.App > 0 {
        refRoles, errR := uc.rur.FindByUserIdsAndAppId(ctx, []int64{id}, userInfo.App)
        if errR != nil {
            uc.log.WithContext(ctx).Errorf("find user roles failed, %v", errR)
            return nil, apiErr.UserFindFail
        }
        
        if len(refRoles) > 0 {
            roleIds := make([]int64, 0)
            for _, refRole := range refRoles {
                roleIds = append(roleIds, refRole.RoleId)
            }
            
            roles, err := uc.rr.FindByIds(ctx, 2, roleIds)
            if err != nil {
                uc.log.WithContext(ctx).Errorf("find roles by ids failed, %v", err)
                return nil, apiErr.UserFindFail
            }
            
            userInfo.Role = roles
        }
    }
    //获取当前用户的权限信息
    if len(userInfo.Role) > 0 {
        roleIds := make([]int64, 0)
        for _, v := range userInfo.Role {
            roleIds = append(roleIds, *v.Id)
        }
        
        rpMap, errP0 := uc.rpUc.FindRolePermissionInfo(ctx, roleIds)
        if errP0 != nil {
            uc.log.WithContext(ctx).Errorf("userInfo:%+v FindRolePermissionInfo failed err:%v", userInfo.Role, errP0)
            return nil, apiErr.UserFindFail
        }
        permissionCode := make([]int64, 0)
        for _, roleId := range roleIds {
            if _, exist := rpMap[roleId]; exist {
                permissionCode = append(permissionCode, rpMap[roleId]...)
            }
        }
        userInfo.PermissionCode = permissionCode
    }
    return userInfo, nil
}

func (uc *UserUsecase) UserRoleInfo(ctx context.Context, uid int64) (*Role, *errors.Error) {
    
    //roleList,_, err := uc.rr.RolePermission(ctx,)
    //
    //if err != nil {
    //    uc.log.WithContext(ctx).Errorf("roleList failed, %v", err)
    //    return nil, apiErr.UserRoleFindFail
    //}
    
    //获取用户信息
    userInfo, errU := uc.ur.FindOne(ctx, uid)
    if errU != nil {
        uc.log.WithContext(ctx).Errorf("uid:%+v roles userInfo failed err:%v", uid, errU)
        return nil, apiErr.UserFindFail
    }

    //通过uid获取roleId
    roles, errF := uc.rur.FindByUserId(ctx, uid, 2)
    if errF != nil {
        uc.log.WithContext(ctx).Errorf("uid:%+v roles FindByUserId failed err:%v", uid, errF)
        return nil, apiErr.UserFindFail
    }

    if userInfo.App >= 2 {
        roles2, _ := uc.rur.FindByUserIdsAndAppId(ctx, []int64{uid}, userInfo.App)
        // 如果不是超级管理员
        if len(roles) == 0 || roles[0].RoleId != 1 {
            roles = roles2
        }
    }
    if len(roles) == 0 {
        log.Errorf("uid:%+v role is nil", uid)
        return nil, apiErr.UserFindFail
    }
    roleInfo := roles[0]
    roleId := roleInfo.RoleId
    if roleId == 1 {
        adminRole, errA := uc.rpUc.FindEmbodiedAidRolePermissionInfo(ctx, []int64{roleId}, userInfo.App)
        if errA != nil {
            uc.log.WithContext(ctx).Errorf("uid:%+v roleId:%+v FindEmbodiedAidRolePermissionInfo failed err:%v", uid, roleId, errA)
            return nil, apiErr.UserFindFail
        }
		roleRow, _ := uc.rr.First(ctx, roleId)
		roleName := ""
		if roleRow != nil {
			roleName = roleRow.Name
		}
		var roleOne Role
        roleOne.Id = &roleId
        roleOne.Name = roleName
        roleOne.PermissionCodes = adminRole
        return &roleOne, nil
    }
    rpMap, errP := uc.rpUc.FindRolePermissionInfo(ctx, []int64{roleId})
    if errP != nil {
        uc.log.WithContext(ctx).Errorf("uid:%+v roleId:%+v FindRolePermissionInfo failed err:%v", uid, roleId, errP)
        return nil, apiErr.UserFindFail
    }
    if _, exist := rpMap[roleId]; exist {

		roleRow, _ := uc.rr.First(ctx, roleId)
		roleName := ""
		if roleRow != nil {
			roleName = roleRow.Name
		}

        var roleOne Role
        roleOne.Id = &roleId
        roleOne.Name = roleName
        roleOne.PermissionCodes = rpMap[roleId]
        return &roleOne, nil
    }
    return nil, nil
}

func (uc *UserUsecase) GetRolesByUids(ctx context.Context, uids []int64) (map[int64]*Role, *errors.Error) {
    res := make(map[int64]*Role)
    //通过uid获取roleId
    roles, errF := uc.rur.FindByUserIds(ctx, uids, 2)
    if errF != nil {
        uc.log.WithContext(ctx).Errorf("uid:%+v roles GetRolesByUids failed err:%v", uids, errF)
        return nil, apiErr.UserFindFail
    }
    if len(roles) == 0 {
        log.Errorf("uids:%+v role is nil", uids)
        //return nil, apiErr.UserFindFail
    }
    roleIds := make([]int64, 0)
    for _, v := range roles {
        roleIds = append(roleIds, v.RoleId)
    }
    rpMap, errP := uc.rpUc.FindRolePermissionInfo(ctx, roleIds)
    if errP != nil {
        uc.log.WithContext(ctx).Errorf("uids:%+v roleIds:%+v FindRolePermissionInfo failed err:%v", uids, roleIds, errP)
        return nil, apiErr.UserFindFail
    }
    //获取角色的名字
    roleList, errL := uc.rr.FindByIds(ctx, 2, roleIds)
    if errL != nil {
        uc.log.WithContext(ctx).Errorf("uids:%+v roleIds:%+v roles FindByIds failed err:%v", uids, roleIds, errL)
        return nil, apiErr.UserFindFail
    }

    for _, v := range roles {
        roleId := v.RoleId
        for _, v2 := range roleList {
            if roleId == *v2.Id {
                roleName := ""
                roleRemarks := ""
                roleName = v2.Name
                roleRemarks = v2.Remarks
                var roleOne Role
                roleOne.Id = &roleId
                roleOne.Name = roleName
                roleOne.Remarks = roleRemarks
                roleOne.PermissionCodes = rpMap[roleId]
                res[v.UserId] = &roleOne
            }
        }

    }
    return res, nil
}

func (uc *UserUsecase) GetRolesByUidsAndAppId(ctx context.Context, uids []int64, appId int64) (map[int64]*Role, *errors.Error) {
    res := make(map[int64]*Role)
    //通过uid和appId获取roleId
    roles, errF := uc.rur.FindByUserIdsAndAppId(ctx, uids, appId)
    if errF != nil {
        uc.log.WithContext(ctx).Errorf("uid:%+v appId:%+v roles GetRolesByUidsAndAppId failed err:%v", uids, appId, errF)
        return nil, apiErr.UserFindFail
    }
    if len(roles) == 0 {
        uc.log.WithContext(ctx).Infof("uids:%+v appId:%+v role is nil", uids, appId)
    }
    roleIds := make([]int64, 0)
    for _, v := range roles {
        roleIds = append(roleIds, v.RoleId)
    }
    rpMap, errP := uc.rpUc.FindRolePermissionInfo(ctx, roleIds)
    if errP != nil {
        uc.log.WithContext(ctx).Errorf("uids:%+v appId:%+v roleIds:%+v FindRolePermissionInfo failed err:%v", uids, appId, roleIds, errP)
        return nil, apiErr.UserFindFail
    }
    //获取角色的名字
    roleList, errL := uc.rr.FindByIdsAndAppId(ctx, appId, roleIds)
    if errL != nil {
        uc.log.WithContext(ctx).Errorf("uids:%+v appId:%+v roleIds:%+v roles FindByIdsAndAppId failed err:%v", uids, appId, roleIds, errL)
        return nil, apiErr.UserFindFail
    }
    for _, v := range roles {
        roleId := v.RoleId
        for _, v2 := range roleList {
            if roleId == *v2.Id {
                var roleOne Role
                roleOne.Id = &roleId
                roleOne.Name = v2.Name
                roleOne.Remarks = v2.Remarks
                roleOne.PermissionCodes = rpMap[roleId]
                res[v.UserId] = &roleOne
            }
        }
    }
    return res, nil
}

func (uc *UserUsecase) GroupInfo(ctx context.Context, id int64) ([]*Group, *errors.Error) {
    refUserGroup, err := uc.rug.FindByUser(ctx, id)
    if err != nil {
        uc.log.WithContext(ctx).Errorf("find user group failed, %v", err)
        return nil, apiErr.UserFindFail
    }
    groups := make([]*Group, len(refUserGroup))
    gids := make([]int64, len(refUserGroup))
    roles := make([]*Role, 0)
    
    for i, ref := range refUserGroup {
        groupId := ref.GroupId
        gids = append(gids, groupId)
        refUserRoles, err := uc.rur.FindByUserId(ctx, id, groupId)
        if err != nil {
            uc.log.WithContext(ctx).Errorf("find user failed, %v", err)
            return nil, apiErr.UserPermissionFindFail
        }
        groupInfo, err := uc.gr.First(ctx, groupId)
        if err != nil {
            uc.log.WithContext(ctx).Errorf("find user failed, %v", err)
            return nil, apiErr.UserGroupFindFail
        }
        set := wuwen.NewLinkedHashSet[int64]()
        for _, rur := range refUserRoles {
            set.Add(rur.RoleId)
        }
        //roleIds := set.GetOrderedElements()
        //if len(roleIds) == 0 {
        //    continue
        //}
        rs, err := uc.rr.FindByIds(ctx, groupId, set.GetOrderedElements())
        if err != nil {
            uc.log.WithContext(ctx).Errorf("find user failed, %v", err)
            return nil, apiErr.UserPermissionFindFail
        }
        groupInfo.Roles = rs
        groups[i] = groupInfo
        roles = append(roles, rs...)
    }
    roleIdSet := wuwen.NewLinkedHashSet[int64]()
    for _, role := range roles {
        if role == nil {
            continue
        }
        roleIdSet.Add(*role.Id)
    }
    rolePerMap, apiE := uc.rpUc.FindRolePermissionInfo(ctx, roleIdSet.GetOrderedElements())
    if apiE != nil {
        uc.log.WithContext(ctx).Errorf("find role permission info failed, %v", apiE)
        return nil, apiE
    }
    
    manager, err := uc.ur.FindManager(ctx, gids)
    if err != nil {
        uc.log.WithContext(ctx).Errorf("list role failed, %s", err.Error())
        return nil, apiErr.UserGroupFindFail
    }
    managerMap := wuwen.SliceMapper2MapWithGroup(manager, func(t *GroupManager) (int64, string) {
        if t == nil {
            return 0, ""
        }
        return t.GroupId, t.NickName
    })
    for i, group := range groups {
        if group == nil {
            continue
        }
        groups[i].ManagerNames = managerMap[*group.Id]
        //permissionCode := make([]int64, 0)
        for j, role := range group.Roles {
            if role == nil || role.Id == nil {
                uc.log.WithContext(ctx).Warnf("role  is nil ,skip")
                continue
            }
            //permissionCode = append(permissionCode, rolePerMap[*role.Id]...)
            groups[i].Roles[j].PermissionCodes = rolePerMap[*role.Id]
        }
        //if len(permissionCode) == 0 {
        //    permissionCode = append(permissionCode, -999)
        //}
        
    }
    
    return groups, nil
}
func (uc *UserUsecase) inflateGroupInfo(ctx context.Context, id int64, isOrgAdmin bool, userInfo *User) *errors.Error {
    if isOrgAdmin {
        groupInfo, err := uc.gr.FindByManagers(ctx, id)
        if err != nil {
            uc.log.WithContext(ctx).Errorf("find group failed, %v", err)
            return apiErr.UserGroupFindFail
        }
        if len(groupInfo) == 0 {
            return apiErr.UserGroupFindFail
        }
        userInfo.Groups = groupInfo
    } else {
        refUserGroups, err := uc.rug.FindByUser(ctx, id)
        if err != nil {
            return apiErr.UserGroupFindFail
        }
        groups := make([]*Group, len(refUserGroups))
        for i, ref := range refUserGroups {
            groups[i] = &Group{
                Id: &ref.GroupId,
            }
        }
        userInfo.Groups = groups
    }
    return nil
}

func (uc *UserUsecase) FindByPage(ctx context.Context, c UserCondition, offset, limit int64) (u []*User, total int64, aErr *errors.Error) {
    var err error
    u, total, err = uc.ur.FindByPage(ctx, c, int(offset), int(limit))
    if err != nil {
        uc.log.Errorf("List dataset err:%+v", err)
        aErr = apiErr.UserFindFail
        return
    }
    userIds := wuwen.SliceMapper(u, func(t *User) int64 {
        if t == nil || t.Id == nil {
            return -1
        }
        return *t.Id
    })
    refUserGroups, err := uc.rug.FindByUserIds(ctx, userIds)
    if err != nil {
        uc.log.WithContext(ctx).Errorf("find user ref group fail")
        aErr = apiErr.UserGroupFindFail
    }
    var groupIds []int64
    userGroupMap := make(map[int64]map[int64]struct{})
    for _, group := range refUserGroups {
        if group == nil {
            continue
        }
        if _, exist := userGroupMap[group.UserId]; exist {
            userGroupMap[group.UserId][group.GroupId] = struct{}{}
        } else {
            userGroupMap[group.UserId] = map[int64]struct{}{
                group.GroupId: {},
            }
        }
        groupIds = append(groupIds, group.GroupId)
    }
    groups, err := uc.gr.FindByIds(ctx, groupIds)
    if err != nil {
        uc.log.WithContext(ctx).Errorf("find user  group info fail")
        aErr = apiErr.UserGroupFindFail
    }
    groupMap := wuwen.SliceMapper2Map(groups, func(t *Group) (int64, *Group) {
        if t == nil || t.Id == nil {
            return -1, nil
        }
        return *t.Id, t
    })
    
    for i, user := range u {
        if user.Id == nil {
            continue
        }
        m := userGroupMap[*user.Id]
        gs := make([]*Group, 0, len(m))
        for gid, _ := range m {
            if g, exist := groupMap[gid]; exist {
                gs = append(gs, g)
            }
        }
        u[i].Groups = gs
    }
    return
}
func (uc *UserUsecase) Login(ctx context.Context, userInfo *User) (lr *LoginRep, aErr *errors.Error) {
    user, err := uc.ur.FindOneByUserName(ctx, userInfo.Username, userInfo.App)
    if err != nil {
        uc.log.WithContext(ctx).Errorf("find user failed, %v", err)
        return nil, apiErr.UserNotExist
    }
    if user.Status == 0 {
        return nil, apiErr.UseIsForbidden
    }
    err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(userInfo.Password))
    if err != nil {
        uc.log.WithContext(ctx).Errorf("compare password failed, %v", err)
        return nil, apiErr.UserPWDNotMatch
    }
    token, err := uc.tokenUc.GenerateToken(ctx, *user.Id, tokenGenerateTypeGateWay, nil)
    if err != nil {
        uc.log.WithContext(ctx).Errorf("generate token failed, %v", err)
        return nil, apiErr.UserTokenGenerateFail
    }
    now := time.Now()
    user.LastLoginAt = &now
    err = uc.ur.Update(ctx, user)
    if err != nil {
        uc.log.WithContext(ctx).Errorf("update user failed, %v", err)
        return nil, apiErr.UserLoginFail
    }
    
    lr = &LoginRep{
        Token: token,
        User:  *user,
    }
    return
}

// Uid2Token 根据 uid 生成登录 token（免密静默登录，语义同 Login）
func (uc *UserUsecase) Uid2Token(ctx context.Context, userId, appId int64) (lr *LoginRep, aErr *errors.Error) {
    user, err := uc.ur.FindOne(ctx, userId)
    if err != nil {
        uc.log.WithContext(ctx).Errorf("uid2token find user failed, %v", err)
        return nil, apiErr.UserNotExist
    }
    if user.Status == 0 {
        return nil, apiErr.UseIsForbidden
    }
    // app 校验：请求指定的应用与用户所属应用不一致视为不存在
    if appId > 0 && user.App != appId {
        uc.log.WithContext(ctx).Errorf("uid2token app not match, user app:%d, req app:%d", user.App, appId)
        return nil, apiErr.UserNotExist
    }
    token, err := uc.tokenUc.GenerateToken(ctx, *user.Id, tokenGenerateTypeGateWay, nil)
    if err != nil {
        uc.log.WithContext(ctx).Errorf("uid2token generate token failed, %v", err)
        return nil, apiErr.UserTokenGenerateFail
    }
    now := time.Now()
    user.LastLoginAt = &now
    err = uc.ur.Update(ctx, user)
    if err != nil {
        uc.log.WithContext(ctx).Errorf("uid2token update user failed, %v", err)
        return nil, apiErr.UserLoginFail
    }

    lr = &LoginRep{
        Token: token,
        User:  *user,
    }
    return
}

func (uc *UserUsecase) checkPwd(ctx context.Context, password string) bool {
    pattern := `^[A-Za-z\d!@#%&]{6,20}$`
    regex, err := regexp.Compile(pattern)
    if err != nil {
        uc.log.WithContext(ctx).Errorf("compile pattern failed, %v", err)
        return false
    }
    return regex.MatchString(password)
    
}
func (uc *UserUsecase) CheckUserExist(ctx context.Context, uId int64) bool {
    
    _, err := uc.ur.FindOne(ctx, uId)
    if err != nil {
        return false
    }
    return true
}

func (uc *UserUsecase) CheckPermission(ctx context.Context, userId int64, path string, method string) error {
    //enforce, err := uc.enforcer.Enforce(sub, path, method)
    //if err != nil {
    //    uc.log.WithContext(ctx).Errorf("%s ,path %s ,method %s ,permission check fail", sub, path, method)
    //    return err
    //}
    //if !enforce {
    //    uc.log.WithContext(ctx).Errorf("%s ,path %s ,method %s , dont have permission", sub, path, method)
    //    return e.New("dont have permission")
    //}
    permissions, err := uc.pr.FindByUser(ctx, userId)
    if err != nil {
        uc.log.WithContext(ctx).Errorf("find user permision fail %+v", err)
        return err
    }
    
    matched := false
    for _, p := range permissions {
        if p == nil {
            continue
        }
        pathRe, err := regexp.Compile(p.Path)
        if err != nil {
            uc.log.WithContext(ctx).Errorf("Invalid regex pattern '%+v': %v", *p, err)
            continue
        }
        var methodRe *regexp.Regexp
        if p.Act != nil {
            methodRe, err = regexp.Compile(*p.Act)
            if err != nil {
                uc.log.WithContext(ctx).Errorf("Invalid regex pattern '%+v': %v", *p, err)
                continue
            }
        } else {
            methodRe, _ = regexp.Compile("\\*")
        }
        
        if pathRe.MatchString(path) && p.Act != nil && methodRe.MatchString(method) {
            uc.log.WithContext(ctx).Infof("path '%s' matches pattern '%+v'", path, *p)
            matched = true
            break
        }
    }
    
    if matched {
        return nil
    } else {
        return e.New("dont have permission")
    }
    
}

func (uc *UserUsecase) UpdateInfo(ctx context.Context, u *User, roleId int64) *errors.Error {
    reqCtx := common.ReqCtxFromContext(ctx)
    err := uc.tm.InTx(ctx, func(ctx context.Context) error {
        // 更新昵称
        if len(u.Nickname) > 0 {
            err := uc.ur.Update(ctx, u)
            if err != nil {
                uc.log.WithContext(ctx).Errorf("update user info failed, %v", err)
                return apiErr.UserUpdateFail
            }
        }
        // 更新角色
        if roleId > 0 {
            // 先删除该用户的所有角色绑定（不需要groupId判断）
            err := uc.rur.BatchDeleteByUserId(ctx, reqCtx.GetUserId(), *u.Id)
            if err != nil {
                uc.log.WithContext(ctx).Errorf("delete ref user role failed, %v", err)
                return apiErr.UserRoleUnBindFail
            }
            // 重新绑定新角色
            userRole := &RefUserRole{
                UserId:   *u.Id,
                RoleId:   roleId,
                CreateBy: reqCtx.GetUserId(),
            }
            err = uc.rur.Create(ctx, userRole)
            if err != nil {
                uc.log.WithContext(ctx).Errorf("bind user role failed, %v", err)
                return apiErr.UserRoleCreateFail
            }
        }
        return nil
    })
    var apiE *errors.Error
    if e.As(err, &apiE) {
        return apiE
    } else if err != nil {
        return apiErr.UserUpdateFail.WithCause(err)
    }
    return nil
}

func (uc *UserUsecase) UpdateRole(ctx context.Context, userId, groupId int64, roleIds []int64) *errors.Error {
    reqCtx := common.ReqCtxFromContext(ctx)
    refuserRoles, err := uc.rur.FindByUserId(ctx, userId, groupId)
    if err != nil {
        uc.log.WithContext(ctx).Errorf("user role  not found by userId: %v", userId)
        return apiErr.UserRoleBindInfo
    }
    oldRoleId := make([]int64, len(refuserRoles))
    //refIds := make([]int64, len(refuserRoles))
    var isAdmin bool
    
    err = uc.tm.InTx(ctx, func(ctx context.Context) error {
        
        if !uc.compareRoleChange(oldRoleId, roleIds) || isAdmin { // 如果角色未变更
            return nil
        }
        err = uc.rur.BatchDelete(ctx, reqCtx.GetUserId(), userId, groupId)
        if err != nil {
            uc.log.WithContext(ctx).Errorf("delete ref user role failed, %v", err)
            return apiErr.UserRoleUnBindFail
        }
        userRoles := make([]*RefUserRole, len(roleIds))
        for i, roleId := range roleIds {
            userRoles[i] = &RefUserRole{
                UserId:   userId,
                RoleId:   roleId,
                CreateBy: reqCtx.GetUserId(),
                GroupId:  groupId,
            }
        }
        err = uc.rur.BatchCreate(ctx, userRoles)
        if err != nil {
            uc.log.WithContext(ctx).Errorf("bind user role  failed, %v", err)
            return apiErr.UserRoleCreateFail
        }
        return nil
    })
    var apiE *errors.Error
    if e.As(err, &apiE) {
        return apiE
    } else if err != nil {
        return apiErr.UserUpdateFail.WithCause(err)
    }
    return nil
}

func (uc *UserUsecase) FindByIds(ctx context.Context, ids []int64) ([]*User, *errors.Error) {
    users, err := uc.ur.FindByIds(ctx, ids)
    if err != nil {
        uc.log.WithContext(ctx).Errorf("find user by ids fail ,ids:%+v,err: %+v", ids, err)
        return nil, apiErr.UserFindFail
    }
    return users, nil
}

func (uc *UserUsecase) FindByCond(ctx context.Context, condition *UserSearchCondition) ([]*User, *errors.Error) {
    users, err := uc.ur.FindByCond(ctx, condition)
    if err != nil {
        uc.log.WithContext(ctx).Errorf("find user by condition fail ,condition:%+v,err: %+v", condition, err)
        return nil, apiErr.UserFindFail
    }
    return users, nil
}

//GetExcludeUserIdsForPermission 获取权限含有2030和2050的用户
func (uc *UserUsecase) GetExcludeUserIdsForPermission(ctx context.Context, user *User, permissionCodes []int64) ([]int64, *errors.Error) {
    if *user.Id == 0 {
        return nil, nil
    }
    currentUserId := *user.Id
    currentRole, err := uc.UserRoleInfo(ctx, currentUserId)
    if err != nil {
        uc.log.WithContext(ctx).Errorf("get current user role failed, userId: %d, err: %v", currentUserId, err)
        return nil, err
    }
    //如果是超管，则不需要考虑这个
    if currentRole != nil && currentRole.Id != nil && *currentRole.Id == roleAdminId {
        return []int64{}, nil
    }
    
    excludeSet := wuwen.NewLinkedHashSet[int64]()
    
    if currentRole != nil && currentRole.Id != nil {
        sameRoleUsers, errF := uc.rur.FindByRole(ctx, *currentRole.Id)
        if errF != nil {
            uc.log.WithContext(ctx).Errorf("find users by role failed, roleId: %d, err: %v", *currentRole.Id, errF)
            return nil, apiErr.UserFindFail
        }
        for _, ref := range sameRoleUsers {
            excludeSet.Add(ref.UserId)
        }
    }
    
    adminRoleUsers, errA := uc.rur.FindByRole(ctx, roleAdminId)
    if errA != nil {
        uc.log.WithContext(ctx).Errorf("find admin role users failed, err: %v", errA)
        return nil, apiErr.UserFindFail
    }
    for _, ref := range adminRoleUsers {
        excludeSet.Add(ref.UserId)
    }
    
    // 检查permissionCodes中是否包含2030或2050
    containsSpecialPermission := false
    targetPermissionCodes := map[int64]struct{}{}
    for _, code := range permissionCodes {
        for _, ec := range ExcludeCode {
            if code == ec {
                targetPermissionCodes[ec] = struct{}{}
                containsSpecialPermission = true
            }
        }
    }
    
    if containsSpecialPermission {
        // 获取所有的role IDs
        allRoles, _, err := uc.rr.ListByAppId(ctx, user.App, "", -1, -1) // 假设groupId为2是具身项目，offset和limit为-1表示获取所有
        if err != nil {
            uc.log.WithContext(ctx).Errorf("list all roles failed, err: %v", err)
            return nil, apiErr.UserRoleFindFail
        }
        
        if len(allRoles) > 0 {
            roleIds := make([]int64, len(allRoles))
            for i, role := range allRoles {
                roleIds[i] = *role.Id
            }
            
            // 获取所有角色的权限信息
            rolePermissionMap, err := uc.rpUc.FindRolePermissionInfo(ctx, roleIds)
            if err != nil {
                uc.log.WithContext(ctx).Errorf("find role permission info failed, roleIds: %v, err: %v", roleIds, err)
                return nil, apiErr.UserPermissionFindFail
            }
            
            // 过滤出包含2030的role IDss
            targetRoleIds := wuwen.NewLinkedHashSet[int64]()
            for roleId, permCodes := range rolePermissionMap {
                for _, code := range permCodes {
                    if _, exist := targetPermissionCodes[code]; exist {
                        targetRoleIds.Add(roleId)
                        break
                    }
                }
            }
            
            // 根据这些role IDs找到对应的所有user IDs
            if targetRoleIds.Size() > 0 {
                for _, roleId := range targetRoleIds.GetOrderedElements() {
                    roleUsers, err := uc.rur.FindByRole(ctx, roleId)
                    if err != nil {
                        uc.log.WithContext(ctx).Errorf("find users by role failed, roleId: %d, err: %v", roleId, err)
                        return nil, apiErr.UserFindFail
                    }
                    for _, ref := range roleUsers {
                        excludeSet.Add(ref.UserId)
                    }
                }
            }
        }
    }
    
    return excludeSet.GetOrderedElements(), nil
}

type UserRepo interface {
    Create(ctx context.Context, e *User) error
    Update(ctx context.Context, e *User) error
    Forbidden(ctx context.Context, id, status int64) error
    FindOne(ctx context.Context, id int64) (*User, error)
    FindByPage(ctx context.Context, c UserCondition, offset, limit int) ([]*User, int64, error)
    Delete(ctx context.Context, userId, id int64) error
    FindOneByUserName(ctx context.Context, userName string, app int64) (*User, error)
    FindByIds(ctx context.Context, ids []int64) ([]*User, error)
    FindByUserName(ctx context.Context, userName []string) ([]*User, error)
    FindManager(ctx context.Context, gids []int64) ([]*GroupManager, error)
    FindGroupManager(ctx context.Context, ids []int64) ([]*GroupManager, error)
    FindByCond(ctx context.Context, condition *UserSearchCondition) ([]*User, error)
}
