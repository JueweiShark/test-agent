package service

import (
    "context"

    pb "wuwen.com/wwuc/api_gen/user/v1"
    "wuwen.com/wwuc/internal/biz/user"
    scommon "wuwen.com/wwuc/internal/service/common"

    uPB "wuwen.com/wwuc/api_gen/user/v1"
)

type PermissionService struct {
    pb.UnimplementedPermissionServer
    uc *user.PermissionUsecase
}

func NewPermissionService(uc *user.PermissionUsecase) *PermissionService {
    return &PermissionService{
        uc: uc,
    }
}
func (s *PermissionService) DeletePermission(ctx context.Context, request *uPB.DeletePermissionRequest) (*uPB.DeletePermissionReply, error) {
    //TODO implement me
    panic("implement me")
}

func (s *PermissionService) UpdatePermission(ctx context.Context, request *uPB.UpdatePermissionRequest) (*uPB.UpdatePermissionReply, error) {
    //TODO implement me
    panic("implement me")
}

func (s *PermissionService) ListPermission(ctx context.Context, req *pb.ListPermissionRequest) (*pb.ListPermissionReply, error) {
    reply := &pb.ListPermissionReply{
        Code:    200,
        Message: scommon.ReplyMessageSuccess,
    }
    list, err := s.uc.List(ctx)
    if err != nil {
        reply.Code = 500
        reply.Message = err.Error()
    }
    permissions := make([]*pb.PermissionInfo, len(list))
    for i, per := range list {
        permissions[i] = &pb.PermissionInfo{
            Id:    per.Id,
            Code:  per.PermissionCode,
            Name:  "",
            Route: "",
            Path:  "",
        }

    }
    reply.Data = permissions
    return reply, nil
}

func (s *PermissionService) CreatePermission(ctx context.Context, req *pb.CreatePermissionRequest) (*pb.CreatePermissionReply, error) {
    reply := &pb.CreatePermissionReply{
        Code:    200,
        Message: scommon.ReplyMessageSuccess,
    }
    var permissions []*user.Permission
    if len(req.GetPath()) > 0 {
        permissions = append(permissions, &user.Permission{
            PermissionCode: req.GetCode(),
            Name:           req.GetName(),
            Path:           req.GetPath(),
            Type:           0,
            Act:            &req.Method,
        })
    }
    if len(req.GetRoute()) > 0 {
        permissions = append(permissions, &user.Permission{
            PermissionCode: req.GetCode(),
            Name:           req.GetName(),
            Path:           req.GetRoute(),
            Type:           1,
            Act:            &req.Method,
        })
    }
    if len(req.GetRoute()) == 0 && len(req.GetPath()) == 0 {
        permissions = append(permissions, &user.Permission{
            PermissionCode: req.GetCode(),
            Name:           req.GetName(),
            Path:           req.GetRoute(),
            Type:           2,
            Act:            &req.Method,
        })
    }
    err := s.uc.Create(ctx, permissions)
    if err != nil {
        reply.Code = err.GetCode()
        reply.Message = err.GetReason()
    }
    return reply, nil
}
