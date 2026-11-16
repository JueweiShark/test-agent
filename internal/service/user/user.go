package service

import (
    "context"
    
    "github.com/go-kratos/kratos/v2/log"
    
    "golang.org/x/crypto/bcrypt"
    apiErr "wuwen.com/wwuc/api/errors"
    pb "wuwen.com/wwuc/api_gen/user/v1"
    "wuwen.com/wwuc/internal/biz/common"
    "wuwen.com/wwuc/internal/biz/user"
    scommon "wuwen.com/wwuc/internal/service/common"
)

type UserService struct {
    pb.UnimplementedUserServer
    userUC *user.UserUsecase
    tUC    *user.TokenUsecase
}

func NewUserService(userUc *user.UserUsecase, tUC *user.TokenUsecase) *UserService {
    return &UserService{
        userUC: userUc,
        tUC:    tUC,
    }
}

func (s *UserService) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserReply, error) {
    reqCtx := common.ReqCtxFromContext(ctx)
    log.Context(ctx).Infof("UserService: Create User ctx user:%d ", reqCtx.GetUserId())
    
    reply := &pb.CreateUserReply{
        Code:    200,
        Message: scommon.ReplyMessageSuccess,
    }
    curUserInfo, errU := s.userUC.UserInfo(ctx, common.ReqCtxFromContext(ctx).GetUserId(), false)
    if errU != nil {
        reply.Code = errU.Code
        reply.Message = errU.Error()
        return reply, nil
    }
    provider := user.KeyUserProviderWuwen
    appId := 0

    if curUserInfo.App==2 {
        //具身平台
        provider = user.KeyUserProviderEmbodiedAi
        appId = 2
    }
    if curUserInfo.App==3 {
        //具身合成
        provider = user.KeyUserProviderEmbodiedGen
        appId = 3
    }
    if req.Password ==""{
        req.Password = "Wuwen2026!"
    }
    // 如果没有传appId 则新增的用户自动和当前为同一appId
    if appId == 0 {
        appId = int(curUserInfo.App)
    }
    userInfo := user.User{
        Username: req.Username,
        Password: req.Password,
        Nickname: req.Nickname,
        Provider: provider,
        App:      int64(appId),
        Status:   1,
    }
    err := s.userUC.CreateOne(ctx, &userInfo, req.GetGroupId(), req.GetRoleIds())
    if err != nil {
        reply.Code = err.Code
        reply.Message = err.Error()
        return reply, nil
    }
    if userInfo.Id != nil {
        reply.Data = &pb.CreateUserReply_Data{UserId: *userInfo.Id}
    }
    return reply, nil
}

func (s *UserService) CreateInnerUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserReply, error) {
    reply := &pb.CreateUserReply{
        Code:    200,
        Message: scommon.ReplyMessageSuccess,
    }

    provider := user.KeyUserProviderWuwen
    appId := req.GetAppId()

    if appId==2 {
        //具身平台
        provider = user.KeyUserProviderEmbodiedAi
    }
    if appId==3 {
        //具身合成
        provider = user.KeyUserProviderEmbodiedGen
    }
    if req.Password ==""{
        req.Password = "Wuwen2026!"
    }

    userInfo := user.User{
        Username: req.Username,
        Password: req.Password,
        Nickname: req.Nickname,
        Provider: provider,
        App:      int64(appId),
        Status:   1,
    }
    err := s.userUC.CreateOne(ctx, &userInfo, req.GetGroupId(), req.GetRoleIds())
    if err != nil {
        reply.Code = err.Code
        reply.Message = err.Error()
        return reply, nil
    }
    if userInfo.Id != nil {
        reply.Data = &pb.CreateUserReply_Data{UserId: *userInfo.Id}
    }
    return reply, nil
}

func (s *UserService) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.UpdateUserReply, error) {
    reqCtx := common.ReqCtxFromContext(ctx)
    log.Context(ctx).Infof("UserService: Update User req:%v,req ctx user %d", req.GetId(), reqCtx.GetUserId())
    reply := &pb.UpdateUserReply{
        Code:    200,
        Message: scommon.ReplyMessageSuccess,
    }
    
    userId := req.GetId()
    //判断密码是否和之前一样
    if len(req.OldPassword) > 0 {
        userInfo, errU := s.userUC.UserInfo(ctx, req.GetId(), true)
        if errU != nil {
            reply.Code = errU.Code
            reply.Message = errU.Error()
            return reply, nil
        }
        //判断密码是否正确
        err2 := bcrypt.CompareHashAndPassword([]byte(userInfo.Password), []byte(req.OldPassword))
        if err2 != nil {
            reply.Code = apiErr.CodePWDErr
            reply.Message = "请输入正确的原密码"
            return reply, nil
        }
        //判断新密码是否和旧密码一样
        err3 := bcrypt.CompareHashAndPassword([]byte(userInfo.Password), []byte(req.Password))
        if err3 == nil {
            reply.Code = apiErr.CodePWDSameErr
            reply.Message = "新密码不可与原密码一致"
            return reply, nil
        }
    }
    userInfo := user.User{
        Id:        &userId,
        Password:  req.Password,
        Nickname:  req.Nickname,
        UpdatedBy: reqCtx.GetUserId(),
    }
    err := s.userUC.Update(ctx, &userInfo)
    if err != nil {
        reply.Code = err.Code
        reply.Message = err.Error()
    }
    return reply, nil
}
func (s *UserService) DeleteUser(ctx context.Context, req *pb.DeleteUserRequest) (*pb.DeleteUserReply, error) {
    reqCtx := common.ReqCtxFromContext(ctx)
    log.Context(ctx).Infof("UserService: DeleteUser req:%v,req ctx user %d", req.GetId(), reqCtx.GetUserId())
    reply := &pb.DeleteUserReply{
        Code:    200,
        Message: scommon.ReplyMessageSuccess,
    }
    //判断操作者和被删除的用户是否属于同一项目
    userInfoAdmin, userInfoAdminErr := s.userUC.UserInfo(ctx, reqCtx.GetUserId(), false)
    if userInfoAdminErr != nil {
        log.Errorf("UserInfo:%+v get err:%+v", reqCtx.GetUserId(), userInfoAdminErr)
        reply.Code = userInfoAdminErr.Code
        reply.Message = userInfoAdminErr.Error()
        return reply, nil
    }
    
    userInfoDel, userInfoDelErr := s.userUC.UserInfo(ctx, req.GetId(), false)
    if userInfoDelErr != nil {
        log.Errorf("UserInfo:%+v get err:%+v", req.GetId(), userInfoDelErr)
        reply.Code = userInfoDelErr.Code
        reply.Message = userInfoDelErr.Error()
        return reply, nil
    }
    if userInfoAdmin.App != userInfoDel.App {
        reply.Code = apiErr.UserDeleteFail.Code
        reply.Message = "用户不在同一项目中"
        return reply, nil
    }
    
    errDel := s.userUC.Delete(ctx, reqCtx.GetUserId(), req.GetId())
    if errDel != nil {
        reply.Code = errDel.Code
        reply.Message = errDel.Error()
    }
    return reply, nil
}

// ForbiddenUser 禁用用户
func (s *UserService) ForbiddenUser(ctx context.Context, req *pb.ForbiddenUserRequest) (*pb.ForbiddenUserReply, error) {
    reqCtx := common.ReqCtxFromContext(ctx)
    log.Context(ctx).Infof("UserService: ForbiddenUser req:%v,req ctx user %d", req.GetId(), reqCtx.GetUserId())
    reply := &pb.ForbiddenUserReply{
        Code:    200,
        Message: scommon.ReplyMessageSuccess,
    }
    err := s.userUC.Forbidden(ctx, req.GetId(), req.GetStatus())
    if err != nil {
        reply.Code = err.Code
        reply.Message = err.Error()
    }
    return reply, nil
}

func (s *UserService) UserInfo(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserReply, error) {
    log.Context(ctx).Infof("UserService: UserInfo req:%v,req ctx user %d", req.GetId())
    reply := &pb.GetUserReply{
        Code:    200,
        Message: scommon.ReplyMessageSuccess,
    }
    checkMatch := true
    if req.GetSource() == "grpc" {
        checkMatch = false
    }
    userInfo, err := s.userUC.UserInfo(ctx, req.GetId(), checkMatch)
    if err != nil {
        reply.Code = err.Code
        reply.Message = err.Error()
        return reply, nil
    }
    var updateAt, lastLoginAt int64
    if userInfo.UpdatedAt != nil {
        updateAt = userInfo.UpdatedAt.UnixMilli()
    }
    if userInfo.LastLoginAt != nil {
        lastLoginAt = userInfo.LastLoginAt.UnixMilli()
    }
    var role pb.RoleInfo
    //获取权限信息
    roleInfo, err := s.userUC.UserRoleInfo(ctx, req.GetId())
    if roleInfo != nil {
        role.Id = *roleInfo.Id
        role.Name = roleInfo.Name
        role.PermissionCodes = roleInfo.PermissionCodes
    }
    
    reply.Data = &pb.UserInfo{
        Id:          *userInfo.Id,
        Username:    userInfo.Username,
        Nickname:    userInfo.Nickname,
        Status:      int32(userInfo.Status),
        CreatedAt:   userInfo.CreatedAt.UnixMilli(),
        UpdatedAt:   updateAt,
        LastLoginAt: lastLoginAt,
        RoleInfo:    &role,
    }
    return reply, nil
}
func (s *UserService) CurrentUserInfo(ctx context.Context, _ *pb.Empty) (*pb.GetUserReply, error) {
    reply := &pb.GetUserReply{
        Code:    200,
        Message: scommon.ReplyMessageSuccess,
    }
    
    userInfo, err := s.userUC.UserInfo(ctx, common.ReqCtxFromContext(ctx).GetUserId(), false)
    if err != nil {
        reply.Code = err.Code
        reply.Message = err.Error()
        return reply, nil
    }
    var updateAt, lastLoginAt int64
    if userInfo.UpdatedAt != nil {
        updateAt = userInfo.UpdatedAt.UnixMilli()
    }
    if userInfo.LastLoginAt != nil {
        lastLoginAt = userInfo.LastLoginAt.UnixMilli()
    }
    reply.Data = &pb.UserInfo{
        Id:          *userInfo.Id,
        Username:    userInfo.Username,
        Nickname:    userInfo.Nickname,
        CreatedAt:   userInfo.CreatedAt.UnixMilli(),
        UpdatedAt:   updateAt,
        LastLoginAt: lastLoginAt,
    }
    return reply, nil
}

func (s *UserService) ListUser(ctx context.Context, req *pb.ListUserRequest) (*pb.ListUserReply, error) {
    log.Context(ctx).Infof("UserService: ListUser")
    reply := &pb.ListUserReply{
        Code:    200,
        Message: scommon.ReplyMessageSuccess,
    }
    reqCtx := common.ReqCtxFromContext(ctx)
    userInfo, err := s.userUC.UserInfo(ctx, reqCtx.GetUserId(), true)
    if err != nil {
        log.Errorf("UserInfo:%+v get err:%+v", reqCtx.GetUserId(), err)
        if err != nil {
            reply.Code = err.Code
            reply.Message = err.Error()
            return reply, nil
        }
    }
    offset, limit := common.GetPageOffset(req.GetPageNo(), req.GetPageSize())
    userInfos, total, err := s.userUC.FindByPage(ctx, user.UserCondition{
        NickName: req.Nickname,
        Username: req.Username,
        Status:   req.Status,
        App:      userInfo.App,
    }, offset, limit)
    if err != nil {
        reply.Code = err.Code
        reply.Message = err.Error()
        return reply, nil
    }
    pbUserInfos := make([]*pb.UserInfo, len(userInfos))
    
    //获取用户的角色信息
    if userInfo.App >= 2 {
        uIds := make([]int64, 0)
        for _, v := range userInfos {
            uIds = append(uIds, *v.Id)
        }
        rolesMap, errR := s.userUC.GetRolesByUidsAndAppId(ctx, uIds, userInfo.App)
        if errR != nil {
            reply.Code = errR.Code
            reply.Message = errR.Error()
            return reply, nil
        }
        
        for i, userInfo := range userInfos {
            var roleInfo pb.RoleInfo
            if _, exists := rolesMap[*userInfo.Id]; exists {
                oneInfo := rolesMap[*userInfo.Id]
                roleInfo.Id = *oneInfo.Id
                roleInfo.Remarks = oneInfo.Remarks
                roleInfo.Name = oneInfo.Name
                roleInfo.PermissionCodes = oneInfo.PermissionCodes
                roleInfo.System = oneInfo.System
            }
            
            pbUserInfos[i] = &pb.UserInfo{
                Id:        *userInfo.Id,
                Username:  userInfo.Username,
                Nickname:  userInfo.Nickname,
                Status:    int32(userInfo.Status),
                RoleInfo:  &roleInfo,
                GroupInfo: s.combineGroup(ctx, userInfo.Groups),
                CreatedAt: userInfo.CreatedAt.UnixMilli(),
            }
        }
    } else {
        for i, userInfo := range userInfos {
            pbUserInfos[i] = &pb.UserInfo{
                Id:        *userInfo.Id,
                Username:  userInfo.Username,
                Nickname:  userInfo.Nickname,
                Status:    int32(userInfo.Status),
                GroupInfo: s.combineGroup(ctx, userInfo.Groups),
                CreatedAt: userInfo.CreatedAt.UnixMilli(),
            }
        }
    }
    reply.Data = &pb.ListUserReplyListData{
        Total:    total,
        PageSize: limit,
        PageNo:   (offset / limit) + 1,
        List:     pbUserInfos,
    }
    return reply, nil
}

//ListPermissionUser 用户列表：过滤掉权限含有2030和2050的用户
func (s *UserService) ListPermissionUser(ctx context.Context, req *pb.ListUserRequest) (*pb.ListUserReply, error) {
    log.Context(ctx).Infof("UserService: ListPermissionUser")
    reply := &pb.ListUserReply{
        Code:    200,
        Message: scommon.ReplyMessageSuccess,
    }
    reqCtx := common.ReqCtxFromContext(ctx)
    userInfo, err := s.userUC.UserInfo(ctx, reqCtx.GetUserId(), true)
    if err != nil {
        log.Errorf("UserInfo:%+v get err:%+v", reqCtx.GetUserId(), err)
        reply.Code = err.Code
        reply.Message = err.Error()
        return reply, nil
    }
    //获取权限含有2030和2050的用户
    excludeUserIds, errEx := s.userUC.GetExcludeUserIdsForPermission(ctx, userInfo, userInfo.PermissionCode)
    if errEx != nil {
        reply.Code = errEx.Code
        reply.Message = errEx.Error()
        return reply, nil
    }
    offset, limit := common.GetPageOffset(req.GetPageNo(), req.GetPageSize())
    userInfos, total, err := s.userUC.FindByPage(ctx, user.UserCondition{
        NickName:   req.Nickname,
        Username:   req.Username,
        Status:     req.Status,
        App:        userInfo.App,
        ExcludeIds: excludeUserIds,
    }, offset, limit)
    if err != nil {
        reply.Code = err.Code
        reply.Message = err.Error()
        return reply, nil
    }
    pbUserInfos := make([]*pb.UserInfo, len(userInfos))
    
    if userInfo.App >0 {
        uIds := make([]int64, 0)
        for _, v := range userInfos {
            uIds = append(uIds, *v.Id)
        }
        rolesMap, errR := s.userUC.GetRolesByUidsAndAppId(ctx, uIds, userInfo.App)
        if errR != nil {
            reply.Code = errR.Code
            reply.Message = errR.Error()
            return reply, nil
        }
        
        for i, userInfo := range userInfos {
            var roleInfo pb.RoleInfo
            if _, exists := rolesMap[*userInfo.Id]; exists {
                oneInfo := rolesMap[*userInfo.Id]
                roleInfo.Id = *oneInfo.Id
                roleInfo.Remarks = oneInfo.Remarks
                roleInfo.Name = oneInfo.Name
                roleInfo.PermissionCodes = oneInfo.PermissionCodes
                roleInfo.System = oneInfo.System
            }
            
            pbUserInfos[i] = &pb.UserInfo{
                Id:        *userInfo.Id,
                Username:  userInfo.Username,
                Nickname:  userInfo.Nickname,
                Status:    int32(userInfo.Status),
                RoleInfo:  &roleInfo,
                GroupInfo: s.combineGroup(ctx, userInfo.Groups),
                CreatedAt: userInfo.CreatedAt.UnixMilli(),
            }
        }
    } else {
        for i, userInfo := range userInfos {
            pbUserInfos[i] = &pb.UserInfo{
                Id:        *userInfo.Id,
                Username:  userInfo.Username,
                Nickname:  userInfo.Nickname,
                Status:    int32(userInfo.Status),
                GroupInfo: s.combineGroup(ctx, userInfo.Groups),
                CreatedAt: userInfo.CreatedAt.UnixMilli(),
            }
        }
    }
    reply.Data = &pb.ListUserReplyListData{
        Total:    total,
        PageSize: limit,
        PageNo:   (offset / limit) + 1,
        List:     pbUserInfos,
    }
    return reply, nil
}

func (s *UserService) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginReply, error) {
    log.Context(ctx).Infof("UserService: user %s Login", req.GetUsername())
    reply := &pb.LoginReply{
        Code:    200,
        Message: scommon.ReplyMessageSuccess,
    }
    
    userInfo := user.User{
        Username: req.Username,
        Password: req.Password,
        App:      req.AppId,
    }
    lr, err := s.userUC.Login(ctx, &userInfo)
    if err != nil {
        reply.Code = err.Code
        reply.Message = err.Error()
        return reply, nil
    }
    var updateAt, lastLoginAt int64
    if lr.UpdatedAt != nil {
        updateAt = lr.UpdatedAt.UnixMilli()
    }
    if lr.LastLoginAt != nil {
        lastLoginAt = lr.LastLoginAt.UnixMilli()
    }
    
    var role pb.RoleInfo
    //获取权限信息
    roleInfo, err := s.userUC.UserRoleInfo(ctx, *lr.Id)
    if roleInfo != nil {
        role.Id = *roleInfo.Id
        role.Name = roleInfo.Name
        role.PermissionCodes = roleInfo.PermissionCodes
    }
    
    reply.Data = &pb.LoginReplyLoginData{
        Token: lr.Token,
        User: &pb.UserInfo{
            Id:          *lr.Id,
            Username:    lr.Username,
            Nickname:    lr.Nickname,
            AvatarId:    "",
            CreatedAt:   lr.CreatedAt.UnixMilli(),
            UpdatedAt:   updateAt,
            LastLoginAt: lastLoginAt,
            RoleInfo:    &role,
            Status:      int32(lr.Status),
        },
    }
    return reply, nil
}

// Uid2Token 根据 uid 生成登录 token（免密静默登录）
func (s *UserService) Uid2Token(ctx context.Context, req *pb.Uid2TokenRequest) (*pb.LoginReply, error) {
    log.Context(ctx).Infof("UserService: Uid2Token userId:%d appId:%d", req.GetUserId(), req.GetAppId())
    reply := &pb.LoginReply{
        Code:    200,
        Message: scommon.ReplyMessageSuccess,
    }

    lr, err := s.userUC.Uid2Token(ctx, req.GetUserId(), req.GetAppId())
    if err != nil {
        reply.Code = err.Code
        reply.Message = err.Error()
        return reply, nil
    }
    var updateAt, lastLoginAt int64
    if lr.UpdatedAt != nil {
        updateAt = lr.UpdatedAt.UnixMilli()
    }
    if lr.LastLoginAt != nil {
        lastLoginAt = lr.LastLoginAt.UnixMilli()
    }

    var role pb.RoleInfo
    //获取权限信息
    roleInfo, err := s.userUC.UserRoleInfo(ctx, *lr.Id)
    if roleInfo != nil {
        role.Id = *roleInfo.Id
        role.Name = roleInfo.Name
        role.PermissionCodes = roleInfo.PermissionCodes
    }

    reply.Data = &pb.LoginReplyLoginData{
        Token: lr.Token,
        User: &pb.UserInfo{
            Id:          *lr.Id,
            Username:    lr.Username,
            Nickname:    lr.Nickname,
            AvatarId:    "",
            CreatedAt:   lr.CreatedAt.UnixMilli(),
            UpdatedAt:   updateAt,
            LastLoginAt: lastLoginAt,
            RoleInfo:    &role,
            Status:      int32(lr.Status),
        },
    }
    return reply, nil
}
func (s *UserService) CheckUserExist(ctx context.Context, userId int64) bool {
    return s.userUC.CheckUserExist(ctx, userId)
}
func (s *UserService) Logout(ctx context.Context, _ *pb.LogoutRequest) (*pb.LogoutReply, error) {
    log.Context(ctx).Infof("UserService:user logout")
    return &pb.LogoutReply{
        Code:    200,
        Message: scommon.ReplyMessageSuccess,
    }, nil
    
}

func (s *UserService) combineMemberRole(ctx context.Context, rs []*user.Role) []*pb.RoleInfo {
    roles := make([]*pb.RoleInfo, len(rs))
    for j, role := range rs {
        if role.Id == nil {
            log.Context(ctx).Infof("role id is nil skip")
            continue
        }
        roles[j] = &pb.RoleInfo{
            Id:              *role.Id,
            Name:            role.Name,
            System:          role.System,
            PermissionCodes: role.PermissionCodes,
        }
    }
    return roles
}
func (s *UserService) combineGroup(ctx context.Context, groups []*user.Group) []*pb.GroupInfo {
    roles := make([]*pb.GroupInfo, 0)
    for _, group := range groups {
        if group == nil || group.Id == nil {
            log.Context(ctx).Infof("role id is nil skip")
            continue
        }
        
        roles = append(roles, &pb.GroupInfo{
            Id:       *group.Id,
            Name:     group.GroupName,
            Role:     s.combineMemberRole(ctx, group.Roles),
            Managers: group.ManagerNames,
        })
    }
    return roles
}

func (s *UserService) CheckPermission(ctx context.Context, path, httpMethod string) error {
    reqCtx := common.ReqCtxFromContext(ctx)
    err := s.userUC.CheckPermission(ctx, reqCtx.GetUserId(), path, httpMethod)
    if err != nil {
        log.Errorf("permission check fail err:%+v", err)
    }
    return err
}
func (s *UserService) CheckTokenExist(ctx context.Context, token string) bool {
    success := s.tUC.FindToken(ctx, token)
    if !success {
        log.Error("token find fail")
    }
    return success
}

func (s *UserService) UpdateInfo(ctx context.Context, req *pb.UpdateInfoRequest) (*pb.UpdateInfoReply, error) {
    reqCtx := common.ReqCtxFromContext(ctx)
    log.Context(ctx).Infof("UserService: UpdateInfo req:%v,req ctx user %d", req.GetId(), reqCtx.GetUserId())
    reply := &pb.UpdateInfoReply{
        Code:    200,
        Message: scommon.ReplyMessageSuccess,
    }
    userId := req.GetId()
    userInfo := user.User{
        Id:        &userId,
        Nickname:  req.Nickname,
        UpdatedBy: reqCtx.GetUserId(),
    }
    err := s.userUC.UpdateInfo(ctx, &userInfo, req.GetRoleId())
    if err != nil {
        reply.Code = err.Code
        reply.Message = err.Error()
    }
    return reply, nil
}

func (s *UserService) UpdateUserRole(ctx context.Context, req *pb.UpdateUserRoleRequest) (*pb.UpdateUserReply, error) {
    log.Context(ctx).Infof("UserService: UpdateUserRole")
    reply := &pb.UpdateUserReply{
        Code:    200,
        Message: scommon.ReplyMessageSuccess,
    }
    err := s.userUC.UpdateRole(ctx, req.GetId(), req.GetGroupId(), req.GetRoleIds())
    if err != nil {
        reply.Code = err.GetCode()
        reply.Message = err.GetMessage()
    }
    return reply, nil
}
func (s *UserService) UserGroupInfo(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserGroupInfoReply, error) {
    log.Context(ctx).Infof("UserService: UpdateUserRole")
    reply := &pb.GetUserGroupInfoReply{
        Code:    200,
        Message: scommon.ReplyMessageSuccess,
    }
    groups, err := s.userUC.GroupInfo(ctx, req.GetId())
    if err != nil {
        reply.Code = err.GetCode()
        reply.Message = err.GetMessage()
        return reply, nil
    }
    reply.Data = s.combineGroup(ctx, groups)
    
    return reply, nil
}
func (s *UserService) CheckToken(ctx context.Context, req *pb.CheckTokenRequest) (*pb.CheckTokenReply, error) {
    return &pb.CheckTokenReply{
        Result: s.CheckTokenExist(ctx, req.GetToken()),
    }, nil
}

func (s *UserService) BatchFindUserUserFindRequest(ctx context.Context, req *pb.UserFindRequest) (*pb.UserFindReply, error) {
    reply := &pb.UserFindReply{
        Code:    200,
        Message: scommon.ReplyMessageSuccess,
    }
    users, e := s.userUC.FindByIds(ctx, req.GetIds())
    if e != nil {
        reply.Code = e.GetCode()
        reply.Message = e.GetReason()
        return reply, nil
    }
    
    data := make([]*pb.UserInfo, 0, len(users))
    for _, u := range users {
        data = append(data, &pb.UserInfo{
            Id:       *u.Id,
            Nickname: u.Nickname,
            Username: u.Username,
        })
    }
    reply.Data = data
    return reply, nil
}

// FindUserGroupRoleRequest
func (s *UserService) FindUserGroupRoleRequest(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserGroupInfoReply, error) {
    log.Context(ctx).Infof("UserService: FindUserGroupRoleRequest")
    reply := &pb.GetUserGroupInfoReply{
        Code:    200,
        Message: scommon.ReplyMessageSuccess,
    }
    groups, err := s.userUC.GroupInfo(ctx, req.GetId())
    if err != nil {
        reply.Code = err.GetCode()
        reply.Message = err.GetMessage()
        return reply, nil
    }
    reply.Data = s.combineGroup(ctx, groups)
    return reply, nil
}

func (s *UserService) FindUserByCondRequest(ctx context.Context, req *pb.FindUserCondRequest) (*pb.UserFindReply, error) {
    log.Context(ctx).Infof("UserService: FindUserByCondRequest")
    reply := &pb.UserFindReply{
        Code:    200,
        Message: scommon.ReplyMessageSuccess,
    }
    //获取用户信息
    appId := int64(0)
    if req.GetAppId() > 0 {
        appId = req.GetAppId()
    }
    users, e := s.userUC.FindByCond(ctx, &user.UserSearchCondition{
        Username: req.GetUsername(),
        NickName: req.GetNickname(),
        Ids:      req.GetIds(),
        AppId:    appId,
    })
    if e != nil {
        reply.Code = e.GetCode()
        reply.Message = e.GetReason()
        return reply, nil
    }
    
    data := make([]*pb.UserInfo, 0, len(users))
    for _, u := range users {
        data = append(data, &pb.UserInfo{
            Id:       *u.Id,
            Nickname: u.Nickname,
            Username: u.Username,
        })
    }
    reply.Data = data
    return reply, nil
}
