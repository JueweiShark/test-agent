package v2

import (
	"context"
	"strings"

	"github.com/go-kratos/kratos/v2/log"

	pb "wuwen.com/wwuc/api_gen/user/v2"
	"wuwen.com/wwuc/internal/biz/common"
	bizV2 "wuwen.com/wwuc/internal/biz/v2"
)

// UserV2Service 超管-用户管理 (手动创建,不登记手机号,不绑企业)
type UserV2Service struct {
	pb.UnimplementedUserV2Server
	userUC *bizV2.UserUsecase
	log    *log.Helper
}

// NewUserV2Service .
func NewUserV2Service(uc *bizV2.UserUsecase, logger log.Logger) *UserV2Service {
	return &UserV2Service{
		userUC: uc,
		log:    log.NewHelper(logger),
	}
}

// CreateUser 超管创建系统用户
func (s *UserV2Service) CreateUser(ctx context.Context, req *pb.CreateUserV2Request) (*pb.CreateUserV2Reply, error) {
	userId := common.ReqCtxFromContext(ctx).GetUserId()
	reply := &pb.CreateUserV2Reply{
		Code:    200,
		Message: "",
	}
	
	// 处理状态: 默认 1 (启用)
	status := int64(req.GetStatus())
	if status == 0 {
		status = 1
	}

	u := &bizV2.User{
		Username: strings.Trim(req.GetUsername(), " "),
		Nickname: strings.Trim(req.GetNickname(), " "),
		Status:    status,
		CreatedBy: userId,
		Remarks:   req.GetRemarks(),
	}
	plainPwd, err := s.userUC.CreateUser(ctx, u)
	if err != nil {
		s.log.WithContext(ctx).Errorf("CreateUser failed, username=%s operator=%d err=%v",
			req.GetUsername(), userId, err)
		reply.Code = err.GetCode()
		reply.Message = err.GetReason()
		return reply, nil
	}
	reply.Data = &pb.CreateUserV2Reply_Data{UserInfo: toUserInfo(u), Password: plainPwd}
	return reply, nil
}

// ResetPassword 超管重置用户密码
func (s *UserV2Service) ResetPassword(ctx context.Context, req *pb.ResetPasswordRequest) (*pb.ResetPasswordReply, error) {
	userId := common.ReqCtxFromContext(ctx).GetUserId()
	reply := &pb.ResetPasswordReply{
		Code:    200,
		Message: "",
	}
	plainPwd, err := s.userUC.ResetPassword(ctx, req.GetUserId())
	if err != nil {
		s.log.WithContext(ctx).Errorf("ResetPassword failed, userId=%d operator=%d err=%v",
			req.GetUserId(), userId, err)
		reply.Code = err.GetCode()
		reply.Message = err.GetReason()
		return reply, nil
	}
	reply.Data = &pb.ResetPasswordReply_Data{Password: plainPwd}
	return reply, nil
}

// UpdatePassword 超管指定新密码修改用户密码 (不回显明文)
func (s *UserV2Service) UpdatePassword(ctx context.Context, req *pb.UpdatePasswordRequest) (*pb.UpdatePasswordReply, error) {
	operatorId := common.ReqCtxFromContext(ctx).GetUserId()
	reply := &pb.UpdatePasswordReply{
		Code:    200,
		Message: "",
	}
	if err := s.userUC.UpdatePassword(ctx, req.GetUserId(), req.GetPassword()); err != nil {
		s.log.WithContext(ctx).Errorf("UpdatePassword failed, userId=%d operator=%d err=%v",
			req.GetUserId(), operatorId, err)
		reply.Code = err.GetCode()
		reply.Message = err.GetReason()
		return reply, nil
	}
	return reply, nil
}

// ListUsers 用户名单 (分页)
func (s *UserV2Service) ListUsers(ctx context.Context, req *pb.ListUsersV2Request) (*pb.ListUsersV2Reply, error) {
	reply := &pb.ListUsersV2Reply{
		Code:    200,
		Message: "",
	}
	offset, limit := common.GetPageOffset(int64(req.GetPage()), int64(req.GetPageSize()))
	cond := bizV2.UserCondition{}
	if req.GetUsername() != "" {
		username := req.GetUsername()
		cond.Username = &username
	}
	if req.GetNickname() != "" {
		nickname := req.GetNickname()
		cond.Nickname = &nickname
	}
	if req.GetStatus() > 0 {
		status := req.GetStatus()
		cond.Status = &status
	}
	users, total, err := s.userUC.FindByPage(ctx, cond, int(offset), int(limit))
	if err != nil {
		s.log.WithContext(ctx).Errorf("ListUsers failed, err=%v", err)
		reply.Code = err.GetCode()
		reply.Message = err.GetReason()
		return reply, nil
	}
	pbUsers := make([]*pb.UserDetailInfo, 0, len(users))
	for _, u := range users {
		pbUsers = append(pbUsers, toUserDetailInfo(u))
	}
	reply.Data = &pb.ListUsersV2Reply_Data{List: pbUsers, Total: total}
	return reply, nil
}

// GetUser 查用户详情
func (s *UserV2Service) GetUser(ctx context.Context, req *pb.GetUserV2Request) (*pb.GetUserV2Reply, error) {
	reply := &pb.GetUserV2Reply{
		Code:    200,
		Message: "",
	}
	u, err := s.userUC.FindById(ctx, req.GetUserId())
	if err != nil {
		s.log.WithContext(ctx).Errorf("GetUser failed, userId=%d err=%v", req.GetUserId(), err)
		reply.Code = err.GetCode()
		reply.Message = err.GetReason()
		return reply, nil
	}
	reply.Data = &pb.GetUserV2Reply_Data{User: toUserDetailInfo(u)}
	return reply, nil
}

// UpdateUser 更新用户信息 (昵称/显隐状态/备注)
func (s *UserV2Service) UpdateUser(ctx context.Context, req *pb.UpdateUserV2Request) (*pb.UpdateUserV2Reply, error) {
	operatorId := common.ReqCtxFromContext(ctx).GetUserId()
	reply := &pb.UpdateUserV2Reply{
		Code:    200,
		Message: "",
	}
	if err := s.userUC.UpdateInfo(ctx, req.GetUserId(), operatorId, req.GetNickname(), int64(req.GetStatus()), req.GetRemarks()); err != nil {
		s.log.WithContext(ctx).Errorf("UpdateUser failed, userId=%d operator=%d err=%v",
			req.GetUserId(), operatorId, err)
		reply.Code = err.GetCode()
		reply.Message = err.GetReason()
		return reply, nil
	}
	return reply, nil
}

// toUserInfo 将 biz User 转换为 proto UserInfo (仅基本信息: id/username/nickname)
func toUserInfo(u *bizV2.User) *pb.UserInfo {
	if u == nil || u.Id == nil {
		return nil
	}
	return &pb.UserInfo{
		UserId:   *u.Id,
		Username: u.Username,
		Nickname: u.Nickname,
	}
}
