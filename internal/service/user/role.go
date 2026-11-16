package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	pb "wuwen.com/wwuc/api_gen/user/v1"
	"wuwen.com/wwuc/internal/biz/common"
	"wuwen.com/wwuc/internal/biz/user"
	scommon "wuwen.com/wwuc/internal/service/common"
)

type RoleService struct {
    pb.UnimplementedRoleServer
    roleUc *user.RoleUsecase
    userUC *user.UserUsecase
}

func NewRoleService(roleUc *user.RoleUsecase,userUc *user.UserUsecase) *RoleService {
    return &RoleService{
        roleUc: roleUc,
        userUC: userUc,
    }
}

func (s *RoleService) CreateRole(ctx context.Context, req *pb.CreateRoleRequest) (*pb.CreateRoleReply, error) {
    reply := &pb.CreateRoleReply{
        Code:    200,
        Message: scommon.ReplyMessageSuccess,
    }
    reqCtx := common.ReqCtxFromContext(ctx)
    //获取用户信息
    userInfo, err := s.userUC.UserInfo(ctx, reqCtx.GetUserId(), true)
    if err != nil {
        log.Errorf("UserInfo uid:%+v  err:%+v",reqCtx.GetUserId(),err )
        reply.Code = err.Code
        reply.Message = err.Error()
        return reply, nil
    }
    groupId := req.GetGroupId()
    if userInfo.App==2{
        groupId = 2
    }
    roleID, err := s.roleUc.Create(ctx, reqCtx.GetUserId(), &user.Role{
        Name:            req.GetName(),
        GroupId:         groupId,
        PermissionCodes: req.GetPermissionCodes(),
        CreateBy:        reqCtx.GetUserId(),
        App:            userInfo.App,
        Remarks:        req.GetRemarks(),
    })
    if err != nil {
        reply.Code = err.Code
        reply.Message = err.Error()
        return reply, nil
    }
    reply.Data = roleID
    return reply, nil
}
func (s *RoleService) UpdateRole(ctx context.Context, req *pb.UpdateRoleRequest) (*pb.UpdateRoleReply, error) {
    reply := &pb.UpdateRoleReply{
        Code:    200,
        Message: scommon.ReplyMessageSuccess,
    }
    reqCtx := common.ReqCtxFromContext(ctx)
    //获取用户信息
    userInfo, errU := s.userUC.UserInfo(ctx, reqCtx.GetUserId(), true)
    if errU != nil {
        log.Errorf("UserInfo uid:%+v  err:%+v",reqCtx.GetUserId(),errU )
        reply.Code = errU.Code
        reply.Message = errU.Error()
        return reply, nil
    }
    groupId := req.GetGroupId()
    if userInfo.App==2{
        groupId = 2
    }
    id := req.GetId()
    err := s.roleUc.Update(ctx, reqCtx.GetUserId(), &user.Role{
        Id:              &id,
        Name:            req.GetName(),
        GroupId:         groupId,
        PermissionCodes: req.GetPermissionCodes(),
        Remarks:         req.GetRemarks(),
    })
    if err != nil {
        reply.Code = err.Code
        reply.Message = err.Error()
        return reply, nil
    }
    return reply, nil
}
func (s *RoleService) DeleteRole(ctx context.Context, req *pb.DeleteRoleRequest) (*pb.DeleteRoleReply, error) {
    reply := &pb.DeleteRoleReply{
        Code:    200,
        Message: scommon.ReplyMessageSuccess,
    }
    reqCtx := common.ReqCtxFromContext(ctx)
    err := s.roleUc.Delete(ctx, reqCtx.GetUserId(), req.GetId())
    if err != nil {
        reply.Code = err.Code
        reply.Message = err.Error()
        return reply, nil
    }
    return reply, nil
}
func (s *RoleService) GetRole(ctx context.Context, req *pb.GetRoleRequest) (*pb.GetRoleReply, error) {
    reply := &pb.GetRoleReply{
        Code:    200,
        Message: scommon.ReplyMessageSuccess,
    }
    role, err := s.roleUc.FindOne(ctx, req.GetId())
    if err != nil {
        reply.Code = err.Code
        reply.Message = err.Error()
        return reply, nil
    }
    roleId := *role.Id
    reply.Data = &pb.RoleInfo{
        Id:              roleId,
        Name:            role.Name,
        PermissionCodes: role.PermissionCodes,
    }
    return reply, nil
}
func (s *RoleService) ListRole(ctx context.Context, req *pb.ListRoleRequest) (*pb.ListRoleReply, error) {
    reply := &pb.ListRoleReply{
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
    groupId := req.GetGroupId()
    // 优先使用请求参数中的 appId，没有则使用当前用户的 appId
    appId := userInfo.App
    roles, err := s.roleUc.List(ctx, groupId, appId, req.Name)
    if err != nil {
        reply.Code = err.Code
        reply.Message = err.Error()
        return reply, nil
    }
    pbRoles := make([]*pb.RoleInfo, len(roles))
    for i, role := range roles {
        roleId := *role.Id
        pbRoles[i] = &pb.RoleInfo{
            Id:              roleId,
            Name:            role.Name,
            Remarks:         role.Remarks,
            Creator:         role.Creator,
            PermissionCodes: role.PermissionCodes,
            CreatedAt:       role.CreateAt.UnixMilli(),
            System:          role.System,
        }
    }
    reply.Data = pbRoles
    return reply, nil
}
