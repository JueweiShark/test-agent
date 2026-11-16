package user

import (
    "context"
    e "errors"
    "fmt"
    "time"

    "github.com/go-kratos/kratos/v2/errors"
    "github.com/go-kratos/kratos/v2/log"
    "github.com/mohae/deepcopy"
    "go.uber.org/zap"
    "gorm.io/gorm"

    apiErr "wuwen.com/wwuc/api/errors"
    "wuwen.com/wwuc/internal/biz/common"
    "wuwen.com/wwuc/util/wuwen"
)

type roleSlice []*Role
type rolePermissionSlice []*RefRolePermission

var (
    managerRoleName = "系统管理员"
    memberRoleName  = "成员"
    defaultRoles    = roleSlice{
        {
            Name:     managerRoleName,
            GroupId:  0,
            CreateBy: 0,
            UpdateBy: 0,
            Creator:  "",
            System:   1,
        }, {
            Name:     memberRoleName,
            GroupId:  0,
            CreateBy: 0,
            UpdateBy: 0,
            Creator:  "",
        },
    }
    groupManagerPermissionCode           = []int64{11, 12, 13, 14, 110, 111, 112, 113, 114, 120, 130, 131, 132, 133, 140, 141, 1101, 1102, 1103, 1104, 1111, 1112, 1113, 1114, 1121, 1122, 1123, 1124, 1141, 1142, 1143, 1144, 1201, 1202, 1203, 1301, 1302, 1303, 1304, 1305, 1306, 1311, 1312, 1313}
    normalPermissionCode                 = []int64{11, 12, 13, 110, 111, 112, 113, 114, 120, 130, 131, 132, 133, 1101, 1102, 1103, 1104, 1111, 1112, 1113, 1114, 1121, 1122, 1123, 1124, 1141, 1142, 1143, 1144, 1201, 1202, 1203, 1301, 1302, 1303, 1304, 1305, 1306, 1311, 1312, 1313}
    groupManagerPermissionCodeEmbodiedAi = []int64{2010, 2020, 2021, 2030, 2031, 2032, 2033, 2040, 2041, 2042, 2043, 2050, 2051, 2052, 2053, 2060}
    normalPermissionCodeEmbodiedAi       = []int64{2010, 2020, 2021, 2030, 2031, 2032, 2033}
)

var GroupCode = []int64{
    2050,
    10800,
}

func (r roleSlice) updateGroupId(groupId int64) {
    for i, _ := range r {
        r[i].GroupId = groupId
    }
}

type GroupCondition struct {
    Name      *string
    AppId     int64
    Ids       []int64
    GroupType *int32
}
type FindGroupCondition struct {
    Name  string
    AppId int64
    Ids   []int64
}

type GroupManager struct {
    Id       int64  `gorm:"column:id"`
    GroupId  int64  `gorm:"column:group_id"`
    NickName string `gorm:"column:nickname"`
}
type Group struct {
    Id             *int64     `mapper:"id"`        // Unique identifier
    GroupName      string     `mapper:"groupName"` // group name
    GroupType      int8       `mapper:"groupType"` // 类型: 1-自有团队, 2-供应商
    Remarks        string     `mapper:"remarks"`   // 备注
    App            int64      `mapper:"app"`       // App ID
    Managers       []int64    `mapper:"managers"`  // the group managers
    ManagerNames   []string   // the group managers
    CreatedBy      int64      `mapper:"createdBy"`      // Creator ID
    CreatedAt      *time.Time `mapper:"createdAt"`      // Creation timestamp
    UpdatedBy      int64      `mapper:"updatedBy"`      // Updater ID
    UpdatedAt      *time.Time `mapper:"updatedAt"`      // Update timestamp
    DeleteUniqueId int64      `mapper:"deleteUniqueId"` // Deletion flag. 0 indicates not deleted, 1 indicates deleted
    DeletedAt      *time.Time `mapper:"deletedAt"`      // Deletion timestamp
    Roles          []*Role
    PermissionCode []int64
}
type Member struct {
    UserId    int64   `json:"userId,omitempty"`
    UserName  string  `json:"userName,omitempty"`
    NickName  string  `json:"nickName,omitempty"`
    Roles     []*Role `json:"roleNames,omitempty"`
    Provider  string  `json:"provider,omitempty"`
    CreatedBy string  `json:"createdBy,omitempty"`
    CreatedAt int64   `json:"createdAt,omitempty"`
    Status    int64   `json:"status,omitempty"`
}
type GroupRepo interface {
    Create(ctx context.Context, g *Group) error
    First(ctx context.Context, id int64) (*Group, error)
    Update(ctx context.Context, g *Group) error
    Delete(ctx context.Context, userId, groupId int64) error
    List(ctx context.Context, c GroupCondition, offset, limit int64) ([]*Group, int64, error)
    ListAll(ctx context.Context, c FindGroupCondition) ([]*Group, error)
    FindByManagers(ctx context.Context, id int64) ([]*Group, error)
    FindByIds(ctx context.Context, id []int64) ([]*Group, error)
}
type GroupUsecase struct {
    tm  common.Transaction
    log *log.Helper
    gr  GroupRepo
    ur  UserRepo
    rr  RoleRepo
    pr  PermissionRepo
    rug RefUserGroupRepo
    rur RefUserRoleRepo
    rrp RefRolePermissionRepo
}

func NewGroupUsecase(logger log.Logger, tm common.Transaction, ur UserRepo, gr GroupRepo, rr RoleRepo,
        rug RefUserGroupRepo, rur RefUserRoleRepo, rrp RefRolePermissionRepo, pr PermissionRepo) *GroupUsecase {
    return &GroupUsecase{
        log: log.NewHelper(logger),
        tm:  tm,
        ur:  ur,
        gr:  gr,
        rr:  rr,
        rug: rug,
        rur: rur,
        rrp: rrp,
        pr:  pr,
    }
}
func (uc *GroupUsecase) Create(ctx context.Context, userId int64, g *Group) (int64, *errors.Error) {

    //users, err := uc.ur.FindByIds(ctx, g.Managers)
    //if len(users) != len(g.Managers) || err != nil { // 如果管理员查询的结果和入参长度不一致，则用户的存在非法的情况，
    //    uc.log.WithContext(ctx).Errorf("user managers has one or more dont exist ori lenth :%d ,dest lenth:%d", len(g.Managers), len(users))
    //    return 0, apiErr.UserGroupCreateFail
    //}
    err := uc.tm.InTx(ctx, func(ctx context.Context) error {
        err := uc.gr.Create(ctx, g)
        if errors.Is(err, gorm.ErrDuplicatedKey) {
            uc.log.WithContext(ctx).Errorf("create group err:%v", err)
            return apiErr.GroupNameDuplicate
        } else if err != nil {
            uc.log.WithContext(ctx).Errorf("create group err:%v", err)
            return apiErr.UserGroupCreateFail
        }
        if g.Id == nil {
            uc.log.WithContext(ctx).Error("create group fail , group id is nil")
            return apiErr.UserGroupCreateFail
        }
        if g.App == 2 {
            return nil
        }
        roles := deepcopy.Copy(defaultRoles).(roleSlice)
        roles.updateGroupId(*g.Id)
        err = uc.rr.BatchCreate(ctx, roles)
        if err != nil {
            uc.log.WithContext(ctx).Error("create role fail")
            return apiErr.UserRoleCreateFail
        }
        for _, role := range roles {
            var codes []int64
            switch {
            case role.Name == managerRoleName:
                codes = groupManagerPermissionCode
                if g.App == 2 {
                    codes = groupManagerPermissionCodeEmbodiedAi
                }
            case role.Name == memberRoleName:
                codes = normalPermissionCode
                if g.App == 2 {
                    codes = normalPermissionCodeEmbodiedAi
                }
            }
            permissions, err := uc.pr.FindByCode(ctx, codes)
            if err != nil {
                uc.log.WithContext(ctx).Errorf("find role permissions fail %+v", zap.Error(err))
                return apiErr.UserPermissionFindFail
            }
            rolePermissions := make([]*RefRolePermission, len(permissions))
            filter := map[string]struct{}{}
            for i, permission := range permissions {
                key := fmt.Sprintf("%d#%d", *role.Id, permission.Id)
                if _, exist := filter[key]; exist {
                    continue
                }
                filter[key] = struct{}{}
                rolePermissions[i] = &RefRolePermission{
                    RoleId:   *role.Id,
                    PermId:   permission.Id,
                    CreateBy: userId,
                }
            }
            err = uc.rrp.CreateRolePermission(ctx, rolePermissions)
            if err != nil {
                uc.log.WithContext(ctx).Errorf("default role createFail")
                return apiErr.UserRoleCreateFail
            }
        }
        return nil
    })
    var apiE *errors.Error
    if e.As(err, &apiE) {
        return 0, apiE
    } else if err != nil {
        uc.log.WithContext(ctx).Errorf("create role failed, %+v", zap.Error(err))
        return 0, apiErr.UserGroupCreateFail.WithCause(err)
    }
    return *g.Id, nil
}

func (uc *GroupUsecase) Update(ctx context.Context, userId int64, r *Group) *errors.Error {
    if r.Id == nil {
        uc.log.WithContext(ctx).Errorf("update group fail , group id is nil")
        return apiErr.UserGroupFindFail
    }
    _, err := uc.gr.First(ctx, *r.Id)

    if err != nil {
        uc.log.WithContext(ctx).Errorf("update group fail , group not exist, group id:%d err:%v", *r.Id, err)
        return apiErr.UserGroupFindFail
    }

    err = uc.gr.Update(ctx, r)
    if err != nil {
        uc.log.WithContext(ctx).Errorf("update group fail ,%d, err:%v", *r.Id, err)
        return apiErr.UserGroupUpdateFail
    }
    return nil
}

func (uc *GroupUsecase) Delete(ctx context.Context, userId, groupId int64) *errors.Error {
    refUserGroups, err := uc.rug.FindByGroupId(ctx, groupId)
    if err != nil {
        uc.log.WithContext(ctx).Errorf("group bind info query fail, cant delete")
        return apiErr.GroupBindInfo
    }
    if len(refUserGroups) != 0 {
        uc.log.WithContext(ctx).Errorf("group has member, cant delete")
        return apiErr.GroupHasMember
    }
    err = uc.gr.Delete(ctx, userId, groupId)
    if err != nil {
        uc.log.WithContext(ctx).Errorf("delete role failed, roleId: %d, err: %s", groupId, err.Error())
        return apiErr.UserGroupDeleteFail
    }
    return nil
}

func (uc *GroupUsecase) FindOne(ctx context.Context, groupId int64) (*Group, *errors.Error) {
    group, err := uc.gr.First(ctx, groupId)
    if err != nil {
        uc.log.WithContext(ctx).Errorf("find role failed, roleId: %d, err: %s", groupId, err.Error())
        return nil, apiErr.UserGroupFindFail
    }
    reqCtx := common.ReqCtxFromContext(ctx)
    userRoles, err := uc.rur.FindByUserId(ctx, groupId, reqCtx.GetUserId())
    if err != nil {
        uc.log.WithContext(ctx).Errorf("find user roles fail ,user:%d,group:%d,err:%+v", reqCtx.GetUserId(), groupId, err)
        return nil, apiErr.UserRoleFindFail
    }
    roleIds := wuwen.SliceMapper(userRoles, func(t *RefUserRole) int64 {
        if t == nil {
            return 0
        }
        return t.RoleId
    })
    roles, err := uc.rr.FindByIds(ctx, groupId, roleIds)
    if err != nil {
        uc.log.WithContext(ctx).Errorf("find user roles fail ,user:%d,group:%d,role ids %+v,err:%+v",
            reqCtx.GetUserId(), groupId, roleIds, err)
        return nil, apiErr.UserRoleFindFail
    }
    group.Roles = roles
    return group, nil
}

func (uc *GroupUsecase) managerInfoMap(ctx context.Context, groups ...*Group) (map[int64][]*User, *errors.Error) {
    m := make(map[int64][]*User)
    for _, group := range groups {
        managers, err := uc.ur.FindByIds(ctx, group.Managers)
        if err != nil {
            uc.log.WithContext(ctx).Errorf("find member by id  failed, %s", err.Error())
            return nil, apiErr.GroupMemberListFail
        }

        m[*group.Id] = managers
    }

    return m, nil
}

func (uc *GroupUsecase) FindByPage(ctx context.Context, curUserInfo *User, c GroupCondition, offset, limit int64) ([]*Group, int64, *errors.Error) {
    adminRoles, err := uc.rr.IsAdmin(ctx, common.ReqCtxFromContext(ctx).GetUserId())
    if err != nil {
        uc.log.WithContext(ctx).Errorf("list role failed, %s", err.Error())
        return nil, 0, apiErr.UserGroupFindFail
    }
    var isAdmin bool
    for _, role := range adminRoles {
        isAdmin = isAdmin || (role != nil && role.Id != nil && *role.Id == 1)
    }
    var list []*Group
    var total int64
    //判断用户是否拥有团队管理权限
    hasGroupRole := false
    if curUserInfo.PermissionCode != nil {
        for _, v := range curUserInfo.PermissionCode {
            for _, code := range GroupCode {
                if code == v {
                    hasGroupRole = true
                }
            }
        }
    }
    if !isAdmin && !hasGroupRole {
        rugs, err := uc.rug.FindByUser(ctx, common.ReqCtxFromContext(ctx).GetUserId())
        if err != nil {
            uc.log.WithContext(ctx).Errorf("list role failed, %s", err.Error())
            return nil, 0, apiErr.UserGroupFindFail
        }
        var gids []int64
        for _, rug := range rugs {
            gids = append(gids, rug.GroupId)
        }
        if len(gids) == 0 {
            return []*Group{}, 0, nil
        }
        c.Ids = gids
    }

    list, total, err = uc.gr.List(ctx, c, offset, limit)
    if err != nil {
        uc.log.WithContext(ctx).Errorf("list role failed, %s", err.Error())
        return nil, 0, apiErr.UserGroupFindFail
    }
    gIDS := wuwen.SliceMapper(list, func(t *Group) int64 {
        if t == nil || t.Id == nil {
            return 0
        }
        return *t.Id
    })

    manager, err := uc.ur.FindGroupManager(ctx, gIDS)
    if err != nil {
        uc.log.WithContext(ctx).Errorf("list role failed, %s", err.Error())
        return nil, 0, apiErr.UserGroupFindFail
    }
    managerMap := wuwen.SliceMapper2MapWithGroup(manager, func(t *GroupManager) (int64, string) {
        if t == nil {
            return 0, ""
        }
        return t.GroupId, t.NickName
    })
    for i, group := range list {
        if group.Id == nil {
            continue
        }
        list[i].ManagerNames = managerMap[*group.Id]
    }
    return list, total, nil
}

// FindAllByPage 分页查询所有团队（不做权限过滤，管理员视角）
func (uc *GroupUsecase) FindAllByPage(ctx context.Context, c GroupCondition, offset, limit int64) ([]*Group, int64, *errors.Error) {
	list, total, err := uc.gr.List(ctx, c, offset, limit)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("list group all failed, %s", err.Error())
		return nil, 0, apiErr.UserGroupFindFail
	}
	return list, total, nil
}

func (uc *GroupUsecase) FindList(ctx context.Context, c FindGroupCondition) ([]*Group, *errors.Error) {

    list, err := uc.gr.ListAll(ctx, c)
    if err != nil {
        uc.log.WithContext(ctx).Errorf("list group failed, %s", err.Error())
        return nil, apiErr.UserGroupFindFail
    }

    gIDS := wuwen.SliceMapper(list, func(t *Group) int64 {
        if t == nil || t.Id == nil {
            return 0
        }
        return *t.Id
    })
    manager, err := uc.ur.FindGroupManager(ctx, gIDS)
    if err != nil {
        uc.log.WithContext(ctx).Errorf("list group failed, %s", err.Error())
        return nil, apiErr.UserGroupFindFail
    }
    managerMap := wuwen.SliceMapper2MapWithGroup(manager, func(t *GroupManager) (int64, string) {
        if t == nil {
            return 0, ""
        }
        return t.GroupId, t.NickName
    })
    for _, group := range list {
        if group == nil || group.Id == nil {
            continue
        }
        group.ManagerNames = managerMap[*group.Id]
    }
    return list, nil
}

func (uc *GroupUsecase) MemberListV2(ctx context.Context, userId int64, nickName string,
        groupId, pageNo, pageSize int64) (members []*Member, total int64, err *errors.Error) {
    /*
       result, err := uc.caClient.UserList(ctx, &caPb.UserListRequest{
           GroupId:     groupId,
           MemberName:  nickName,
           CurrentPage: pageNo,
           PageSize:    pageSize,
       })

       if err != nil || result.GetCode() != 0 || result.GetData() == nil {
           uc.log.WithContext(ctx).Errorf("call changan for user member list fail, %s", err)
           return nil, 0, apiErr.GroupMemberListFail
       }
       var memberIds []int64
       for _, member := range result.GetData().GetResultList() {
           num := member.GetMemberNo()
           memberIds = append(memberIds, num)
       }
       refUserRoles, err := uc.rur.FindByUserIds(ctx, memberIds, groupId)
       if err != nil {
           uc.log.WithContext(ctx).Errorf("find member role by uer ids  failed, %s", err)
           return nil, 0, apiErr.UserNotExist
       }
       roleSet := wuwen.NewLinkedHashSet[int64]()
       // 添加团队管理员角色id
       roleSet.Add(roleOrgAdminId, roleDefaultId)
       for _, item := range refUserRoles {
           roleSet.Add(item.RoleId)
       }
       refUserRoleMap := wuwen.SliceMapper2MapWithGroup(refUserRoles, func(item *RefUserRole) (int64, int64) {
           if item == nil {
               return 0, 0
           }
           return item.UserId, item.RoleId
       })
       roles, err := uc.rr.FindByIds(ctx, roleSet.GetOrderedElements())
       if err != nil {
           uc.log.WithContext(ctx).Errorf("find role by id  failed, %s", err.Error())
           return nil, 0, apiErr.UserRoleFindFail
       }
       roleMap := wuwen.SliceMapper2Map(roles, func(item *Role) (int64, *Role) {
           return *item.Id, item
       })
       members := make([]*Member, len(result.GetData().GetResultList()))
       defaultRole, ok := roleMap[roleDefaultId]
       for i, member := range result.GetData().GetResultList() {
           usrId := member.GetMemberNo()
           roleIds := refUserRoleMap[usrId]
           memberRoles := make([]*Role, len(roleIds))
           if member.MemberRole == caPb.MemberRole_ROLE_ADMIN {
               memberRoles = append(memberRoles, &Role{
                   Name: "团队管理员",
               })
           } else {
               for j, roleId := range roleIds {
                   if role, exist := roleMap[roleId]; exist {
                       memberRoles[j] = role
                   } else {
                       memberRoles[j] = roleMap[roleDefaultId]
                   }
               }
           }
           if len(memberRoles) == 0 && ok {
               memberRoles = append(memberRoles, defaultRole)
           }

           parse, _ := time.Parse("2006-01-02 15:04:05", member.GetCreateTime())

           members[i] = &Member{
               UserId:    usrId,
               UserName:  strconv.FormatInt(usrId, 10),
               NickName:  member.GetMemberName(),
               Roles:     memberRoles,
               Provider:  "",
               CreatedBy: member.GetCreateUser(),
               CreatedAt: parse.UnixMilli(),
           }
       }
    */
    return
}
func (uc *GroupUsecase) MemberList(ctx context.Context, userId int64, c UserCondition,
        groupId, offset, limit int64) ([]*Member, int64, *errors.Error) {
    managerRefUserRoles, err := uc.rur.FindByUserId(ctx, userId, groupId)
    if err != nil {
        uc.log.WithContext(ctx).Errorf("created user not found by userId: %v", userId)
        return nil, 0, apiErr.UserPermissionMissFail
    }
    var isAdmin bool
    for _, role := range managerRefUserRoles {
        isAdmin = isAdmin || role.RoleId == roleAdminId
    }
    if isAdmin && groupId == 0 {
        groupId = systemGroupId
    }
    if groupId == 0 {
        uc.log.WithContext(ctx).Errorf("groupId is zero")
        return nil, 0, apiErr.UserGroupFindFail
    }

    //apiE := uc.checkIsManager(ctx, userId, groupId)
    //if apiE != nil {
    //    uc.log.WithContext(ctx).Errorf("check group member failed, %s", apiE.Error())
    //    return nil, 0, apiE
    //}

    refUserGroups, total, err := uc.rug.FindMembersByGroup(ctx, c, groupId, offset, limit)
    if err != nil {
        uc.log.WithContext(ctx).Errorf("find group member failed, %s", err.Error())
        return nil, 0, apiErr.GroupMemberListFail
    }

    userJoinTimeMap := make(map[int64]int64)
    memberIds := wuwen.SliceMapper(refUserGroups, func(item *RefUserGroup) int64 {
        if item == nil {
            return 0
        }
        userJoinTimeMap[item.UserId] = item.CreatedTime.UnixMilli()
        return item.UserId
    })
    members, err := uc.ur.FindByIds(ctx, memberIds)
    if err != nil {
        uc.log.WithContext(ctx).Errorf("find member by id  failed, %s", err.Error())
        return nil, 0, apiErr.GroupMemberListFail
    }
    createdUserSet := wuwen.NewLinkedHashSet[int64]()
    for _, item := range members {
        createdUserSet.Add(item.CreatedBy)
    }
    createdUsers, err := uc.ur.FindByIds(ctx, createdUserSet.GetOrderedElements())
    if err != nil {
        uc.log.WithContext(ctx).Errorf("find member by id  failed, %s", err.Error())
        return nil, 0, apiErr.GroupMemberListFail
    }
    providerMap := wuwen.SliceMapper2Map(createdUsers, func(item *User) (int64, string) {
        if item == nil {
            return 0, ""
        }
        return *item.Id, item.Nickname
    })
    refUserRoles, err := uc.rur.FindByUserIds(ctx, memberIds, groupId)
    if err != nil {
        uc.log.WithContext(ctx).Errorf("find member role by uer ids  failed, %s", err.Error())
        return nil, 0, apiErr.UserNotExist
    }
    roleSet := wuwen.NewLinkedHashSet[int64]()
    for _, item := range refUserRoles {
        roleSet.Add(item.RoleId)
    }
    refUserRoleMap := wuwen.SliceMapper2MapWithGroup(refUserRoles, func(item *RefUserRole) (int64, int64) {
        if item == nil {
            return 0, 0
        }
        return item.UserId, item.RoleId
    })
    roles, err := uc.rr.FindByIds(ctx, groupId, roleSet.GetOrderedElements())
    if err != nil {
        uc.log.WithContext(ctx).Errorf("find role by id  failed, %s", err.Error())
        return nil, 0, apiErr.UserRoleFindFail
    }
    roleMap := wuwen.SliceMapper2Map(roles, func(item *Role) (int64, *Role) {
        return *item.Id, item
    })

    memberInfos := make([]*Member, len(members))
    for i, user := range members {
        id := *user.Id
        roleIds := refUserRoleMap[id]
        memberRoles := make([]*Role, len(roleIds))
        for j, roleId := range roleIds {
            memberRoles[j] = roleMap[roleId]
        }
        memberInfos[i] = &Member{
            UserId:    id,
            UserName:  user.Username,
            NickName:  user.Nickname,
            Status:    user.Status,
            Roles:     memberRoles,
            Provider:  user.Provider,
            CreatedBy: providerMap[user.CreatedBy],
            CreatedAt: userJoinTimeMap[id],
        }
    }
    return memberInfos, total, nil
}

func (uc *GroupUsecase) checkIsManager(ctx context.Context, userId int64, groupId int64) *errors.Error {
    first, err := uc.gr.First(ctx, groupId)
    if err != nil {
        uc.log.WithContext(ctx).Errorf("group not exist: %d, err: %s", groupId, err.Error())
        return apiErr.UserGroupFindFail
    }
    managers := first.Managers
    managerMap := wuwen.SliceMapper2Map(managers, func(item int64) (int64, struct{}) {
        return item, struct{}{}
    })
    if _, exists := managerMap[userId]; !exists {
        uc.log.WithContext(ctx).Errorf("user %d is not a manager of this group: %d", userId, groupId)
        return apiErr.UserPermissionMissFail
    }
    return nil
}

func (uc *GroupUsecase) AddMember(ctx context.Context, groupId int64, members []*Member) (int64, *errors.Error) {

    reqCtx := common.ReqCtxFromContext(ctx)
    names := make([]string, 0)
    userIds := make([]int64, 0)
    for _, m := range members {
        if m.UserId == 0 {
            names = append(names, m.UserName)
        } else {
            userIds = append(userIds, m.UserId)
        }
    }
    user, err := uc.ur.FindByUserName(ctx, names)
    if err != nil {
        uc.log.WithContext(ctx).Errorf("find user by name fail,name:%+v,err:%+v", names, err)
        return 0, apiErr.UserFindFail
    }
    var successCount int64
    for _, u := range user {
        if u.Id == nil {
            uc.log.WithContext(ctx).Warnf("user id is nil %+v", u)
            continue
        }
        userIds = append(userIds, *u.Id)
    }
    for _, userId := range userIds {
        err := uc.rug.BatchCreate(ctx, []*RefUserGroup{{
            UserId:    userId,
            GroupId:   groupId,
            CreatedBy: reqCtx.GetUserId(),
        }})
        if err == nil {
            successCount++
        }
    }

    return successCount, nil
}

func (uc *GroupUsecase) DelMember(ctx context.Context, groupId int64, members []int64) *errors.Error {
    reqCtx := common.ReqCtxFromContext(ctx)
    err := uc.rug.RemoveUser(ctx, reqCtx.GetUserId(), groupId, members)

    if err != nil {
        uc.log.WithContext(ctx).Errorf("member Del fail,%+v", err)
        return apiErr.GroupDelMemberFail
    }
    return nil
}
