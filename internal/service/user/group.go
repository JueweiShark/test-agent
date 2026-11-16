package service

import (
    "context"
    "time"
    
    "github.com/go-kratos/kratos/v2/log"
    "github.com/golang/protobuf/proto"
    
    apiErr "wuwen.com/wwuc/api/errors"
    pb "wuwen.com/wwuc/api_gen/user/v1"
    "wuwen.com/wwuc/internal/biz/common"
    "wuwen.com/wwuc/internal/biz/user"
    scommon "wuwen.com/wwuc/internal/service/common"
)

type GroupService struct {
    pb.UnimplementedGroupServer
    uc *user.GroupUsecase
    userUC *user.UserUsecase
}

func NewGroupService(uc *user.GroupUsecase,userUc *user.UserUsecase, ) *GroupService {
    return &GroupService{
        uc: uc,
        userUC: userUc,
    }
}

func (s *GroupService) CreateGroup(ctx context.Context, req *pb.CreateGroupRequest) (*pb.CreateGroupReply, error) {
    reply := &pb.CreateGroupReply{
        Code:    200,
        Message: scommon.ReplyMessageSuccess,
    }
    //获取用户信息
    reqCtx := common.ReqCtxFromContext(ctx)
    userInfo, err := s.userUC.UserInfo(ctx, reqCtx.GetUserId(), true)
    if err != nil {
        log.Errorf("UserInfo uid:%+v  err:%+v",reqCtx.GetUserId(),err )
        reply.Code = err.Code
        reply.Message = err.Error()
        return reply, nil
    }
    id, err := s.uc.Create(ctx, reqCtx.GetUserId(), &user.Group{
        GroupName: req.GetName(),
        Managers:  req.GetManagers(),
        Remarks:  req.GetRemarks(),
        GroupType: int8(req.GetGroupType()),
        App: userInfo.App,
    })
    if err != nil {
        reply.Code = err.Code
        reply.Message = err.Error()
        return reply, nil
    }
    reply.Data = id
    return reply, nil
}
func (s *GroupService) UpdateGroup(ctx context.Context, req *pb.UpdateGroupRequest) (*pb.UpdateGroupReply, error) {
    reply := &pb.UpdateGroupReply{
        Code:    200,
        Message: scommon.ReplyMessageSuccess,
    }
    reqCtx := common.ReqCtxFromContext(ctx)
    groupId := req.GetId()
    err := s.uc.Update(ctx, reqCtx.GetUserId(), &user.Group{
        Id:        &groupId,
        GroupName: req.GetName(),
        Remarks:   req.GetRemarks(),
        Managers:  req.GetManagers(),
        GroupType: int8(req.GetGroupType()),
    })
    if err != nil {
        reply.Code = err.Code
        reply.Message = err.Error()
        return reply, nil
    }
    return reply, nil
}
func (s *GroupService) DeleteGroup(ctx context.Context, req *pb.DeleteGroupRequest) (*pb.DeleteGroupReply, error) {
    reply := &pb.DeleteGroupReply{
        Code:    200,
        Message: scommon.ReplyMessageSuccess,
    }
    reqCtx := common.ReqCtxFromContext(ctx)
    err := s.uc.Delete(ctx, reqCtx.GetUserId(), req.GetId())
    if err != nil {
        reply.Code = err.Code
        reply.Message = err.Error()
        return reply, nil
    }
    return reply, nil
}
func (s *GroupService) GetGroup(ctx context.Context, req *pb.GetGroupRequest) (*pb.GetGroupReply, error) {
    reply := &pb.GetGroupReply{
        Code:    200,
        Message: scommon.ReplyMessageSuccess,
    }
    group, err := s.uc.FindOne(ctx, req.GetId())
    if err != nil {
        reply.Code = err.Code
        reply.Message = err.Error()
        return reply, nil
    }
    //managers := s.combineGroupManager(ctx, group)
    reply.Data = &pb.GroupInfo{
        Id:   *group.Id,
        Name: group.GroupName,
        Remarks: group.Remarks,
        GroupType: int32(group.GroupType),
    }
    return reply, nil
}
func (s *GroupService) ListGroup(ctx context.Context, req *pb.ListGroupRequest) (*pb.ListGroupReply, error) {
    reply := &pb.ListGroupReply{
        Code:    200,
        Message: scommon.ReplyMessageSuccess,
    }
    //获取用户信息
    reqCtx := common.ReqCtxFromContext(ctx)
    userInfo, err := s.userUC.UserInfo(ctx, reqCtx.GetUserId(), true)
    if err != nil {
        log.Errorf("UserInfo uid:%+v  err:%+v",reqCtx.GetUserId(),err )
        reply.Code = err.Code
        reply.Message = err.Error()
        return reply, nil
    }
    offset, limit := common.GetPageOffset(req.GetPageNo(), req.GetPageSize())
    groups, total, err := s.uc.FindByPage(ctx,userInfo, user.GroupCondition{
        Name: req.Name,
        AppId: userInfo.App,
        GroupType: req.GroupType,
    }, offset, limit)
    if err != nil {
        reply.Code = err.Code
        reply.Message = err.Error()
        return reply, nil
    }
    infos := make([]*pb.GroupInfo, len(groups))
    for i, group := range groups {
        var t time.Time
        if group.CreatedAt != nil {
            t = *group.CreatedAt
        }
        infos[i] = &pb.GroupInfo{
            Id:        *group.Id,
            Name:      group.GroupName,
            Remarks:   group.Remarks,
            CreatedAt: t.UnixMilli(),
            Managers:  group.ManagerNames,
            GroupType: int32(group.GroupType),
        }
    }
    reply.Data = &pb.ListGroupReply_Data{
        Total:    total,
        PageSize: int64(limit),
        PageNo:   int64((offset / limit) + 1),
        List:     infos,
    }
    return reply, nil
}

func (s *GroupService) ListGroupAll(ctx context.Context, req *pb.ListGroupAllRequest) (*pb.ListGroupAllReply, error) {
    reply := &pb.ListGroupAllReply{
        Code:    200,
        Message: scommon.ReplyMessageSuccess,
    }
    // app 必填校验
    if req.App == nil || req.GetApp() == 0 {
        reply.Code = apiErr.CodeParamsInvalid
        reply.Message = "app is required"
        return reply, nil
    }
    offset, limit := common.GetPageOffset(req.GetPageNo(), req.GetPageSize())
    groups, total, err := s.uc.FindAllByPage(ctx, user.GroupCondition{
        Name:      req.Name,
        AppId:     int64(req.GetApp()),
        GroupType: req.GroupType,
    }, offset, limit)
    if err != nil {
        reply.Code = err.Code
        reply.Message = err.Error()
        return reply, nil
    }
    list := make([]*pb.ListGroupAllReply_Group, len(groups))
    for i, group := range groups {
        list[i] = &pb.ListGroupAllReply_Group{
            Id:        *group.Id,
            Name:      group.GroupName,
            Remarks:   group.Remarks,
            GroupType: int32(group.GroupType),
        }
    }
    reply.Data = &pb.ListGroupAllReply_Data{
        Total:    total,
        PageSize: int64(limit),
        PageNo:   int64((offset / limit) + 1),
        List:     list,
    }
    return reply, nil
}

func (s *GroupService) FindGroupList(ctx context.Context, req *pb.FindGroupListRequest) (*pb.FindGroupListReply, error) {
    reply := &pb.FindGroupListReply{
        Code:    200,
        Message: scommon.ReplyMessageSuccess,
    }
    groups, apiE := s.uc.FindList(ctx, user.FindGroupCondition{
        Name:  req.Name,
        Ids:   req.GetIds(),
        AppId: req.AppId,
    })
    if apiE != nil {
        reply.Code = apiE.Code
        reply.Message = apiE.Error()
        return reply, nil
    }

    infos := make([]*pb.GroupInfo, len(groups))
    for i, group := range groups {
        var t time.Time
        if group.CreatedAt != nil {
            t = *group.CreatedAt
        }
        infos[i] = &pb.GroupInfo{
            Id:        *group.Id,
            Name:      group.GroupName,
            Remarks:   group.Remarks,
            CreatedAt: t.UnixMilli(),
            Managers:  group.ManagerNames,
        }
    }
    reply.Data = infos
    return reply, nil
}

//func (s *GroupService) combineGroupManager(ctx context.Context, group *user.Group) []*pb.GroupInfo_Manager {
//    managers := make([]*pb.GroupInfo_Manager, len(group.ManagerInfos))
//    for j, info := range group.ManagerInfos {
//        if info.Id == nil {
//            log.Context(ctx).Warnf("manager id is nil skip")
//            continue
//        }
//        managers[j] = &pb.GroupInfo_Manager{
//            Id:       *info.Id,
//            Nickname: info.Nickname,
//        }
//    }
//    return managers
//}

func (s *GroupService) MemberList(ctx context.Context, req *pb.MemberListRequest) (*pb.MemberListReply, error) {
    reply := &pb.MemberListReply{
        Code:    200,
        Message: scommon.ReplyMessageSuccess,
    }
    offset, limit := common.GetPageOffset(req.GetPageNo(), req.GetPageSize())
    reqCtx := common.ReqCtxFromContext(ctx)
    curUserInfo, errC := s.userUC.UserInfo(ctx, reqCtx.GetUserId(), true)
    if errC !=nil{
        log.Errorf("UserInfo:%+v get err:%+v", reqCtx.GetUserId(),errC)
        reply.Code = errC.Code
        reply.Message = errC.Error()
        return reply, nil
    }
    members, total, err := s.uc.MemberList(ctx, reqCtx.GetUserId(), user.UserCondition{
        NickName: req.Nickname,
        Username: req.Username,
    }, req.GetGroupId(),
        offset, limit)
    if err != nil {
        reply.Code = err.Code
        reply.Message = err.Error()
        return reply, nil
    }
    infos := make([]*pb.MemberInfo, len(members))
    if curUserInfo.App >= 2 {
        //如果是具身，则获取角色信息
        uIds := make([]int64,0)
        for _, v := range members {
            uIds = append(uIds,v.UserId)
        }
        rolesMap, errR := s.userUC.GetRolesByUidsAndAppId(ctx, uIds, curUserInfo.App)
        if errR != nil {
            reply.Code = errR.Code
            reply.Message = errR.Error()
            return reply, nil
        }

        for mK, member := range members {
                var roleInfo pb.RoleInfo
                if _, exists := rolesMap[member.UserId]; exists {
                    oneInfo := rolesMap[member.UserId]
                    roleInfo.Id = *oneInfo.Id
                    roleInfo.Remarks = oneInfo.Remarks
                    roleInfo.Name = oneInfo.Name
                    roleInfo.PermissionCodes = oneInfo.PermissionCodes
                    roleInfo.System = oneInfo.System
                }
                infos[mK] = &pb.MemberInfo{
                    Id:        proto.Int64(member.UserId),
                    Username:  proto.String(member.UserName),
                    Nickname:  member.NickName,
                    RoleInfo:  &roleInfo,
                    Provider:  member.Provider,
                    CreatedBy: member.CreatedBy,
                    CreatedAt: member.CreatedAt,
                    Status: int32(member.Status),
                }
        }
    }else {
        for i, member := range members {
            roles := s.combineMemberRole(ctx, member)
            infos[i] = &pb.MemberInfo{
                Id:        proto.Int64(member.UserId),
                Username:  proto.String(member.UserName),
                Nickname:  member.NickName,
                Role:      roles,
                Provider:  member.Provider,
                CreatedBy: member.CreatedBy,
                CreatedAt: member.CreatedAt,
            }
            
        }
    }
    reply.Data = &pb.MemberListReply_Data{
        Total:    total,
        PageSize: req.GetPageSize(),
        PageNo:   req.GetPageNo(),
        List:     infos,
    }
    return reply, nil
}

func (s *GroupService) AddMember(ctx context.Context, req *pb.AddMemberRequest) (*pb.AddMemberReply, error) {
    reply := &pb.AddMemberReply{
        Code:    200,
        Message: scommon.ReplyMessageSuccess,
    }
    members := make([]*user.Member, len(req.GetMembers()))
    for i, info := range req.GetMembers() {
        member := user.Member{}
        if info.Id != nil {
            member.UserId = info.GetId()
        } else if info.Username != nil {
            member.UserName = info.GetUsername()
        } else {
            reply.Code = apiErr.CodeParamsInvalid
            reply.Message = "member info invalid"
            return reply, nil
        }
        members[i] = &member
    }
    num, err := s.uc.AddMember(ctx, req.GetGroupId(), members)
    if err != nil {
        reply.Code = err.GetCode()
        reply.Message = err.GetReason()
        return reply, nil
    }
    reply.Data = num
    return reply, nil
}
func (s *GroupService) RemoveMember(ctx context.Context, req *pb.RemoveMemberRequest) (*pb.AddMemberReply, error) {
    reply := &pb.AddMemberReply{
        Code:    200,
        Message: scommon.ReplyMessageSuccess,
    }

    err := s.uc.DelMember(ctx, req.GetGroupId(), req.GetMembers())
    if err != nil {
        reply.Code = err.GetCode()
        reply.Message = err.GetReason()
        return reply, nil
    }
    return reply, nil
}
func (s *GroupService) combineMemberRole(ctx context.Context, member *user.Member) []*pb.MemberInfo_Role {
    roles := make([]*pb.MemberInfo_Role, len(member.Roles))
    for j, role := range member.Roles {
        if role == nil || role.Id == nil {
            log.Context(ctx).Infof("role id is nil skip")
            continue
        }
        roles[j] = &pb.MemberInfo_Role{
            Id:   *role.Id,
            Name: role.Name,
        }
    }
    return roles
}
