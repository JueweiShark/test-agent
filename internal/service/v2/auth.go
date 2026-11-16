package v2

import (
	"context"
	"github.com/go-kratos/kratos/v2/log"

	pb "wuwen.com/wwuc/api_gen/user/v2"
	"wuwen.com/wwuc/internal/biz/common"
	bizV2 "wuwen.com/wwuc/internal/biz/v2"
)

// AuthV2Service 登录与总览
type AuthV2Service struct {
	pb.UnimplementedAuthV2Server
	authUC *bizV2.AuthUsecase
	log    *log.Helper
}

// NewAuthV2Service .
func NewAuthV2Service(authUC *bizV2.AuthUsecase, logger log.Logger) *AuthV2Service {
	return &AuthV2Service{
		authUC: authUC,
		log:    log.NewHelper(logger),
	}
}

// LoginV2 登录 (白名单接口, 无需 token)
func (s *AuthV2Service) LoginV2(ctx context.Context, req *pb.LoginV2Request) (*pb.LoginV2Reply, error) {
	reply := &pb.LoginV2Reply{
		Code:    200,
		Message: "",
	}
	result, err := s.authUC.LoginV2(ctx, req.GetUsername(), req.GetPassword())
	if err != nil {
		s.log.WithContext(ctx).Errorf("LoginV2 failed, username=%s err=%v", req.GetUsername(), err)
		reply.Code = err.GetCode()
		reply.Message = err.GetReason()
		return reply, nil
	}
	return s.toLoginReply(result), nil
}

// toLoginReply 将登录结果装配为响应 (Groups/Permissions 始终为非 nil 切片)
func (s *AuthV2Service) toLoginReply(result *bizV2.LoginResult) *pb.LoginV2Reply {
	data := &pb.LoginV2Reply_Data{
		Token:        result.Token,
		ExpireTime:   result.ExpireTime,
		IsSuperAdmin: result.IsSuperAdmin,
	}
	if result.User != nil && result.User.Id != nil {
		data.UserInfo = &pb.UserInfo{
			UserId:   *result.User.Id,
			Username: result.User.Username,
			Nickname: result.User.Nickname,
		}
	}
	data.Groups = make([]*pb.GroupInfo, 0, len(result.Groups))
	for _, g := range result.Groups {
		data.Groups = append(data.Groups, toGroupInfo(g))
	}
	data.Permissions = make([]*pb.PermissionInfo, 0, len(result.Permissions))
	for _, p := range result.Permissions {
		data.Permissions = append(data.Permissions, toPermissionInfo(p))
	}
	return &pb.LoginV2Reply{Code: 200, Message: "", Data: data}
}

// ListMyApps 总览查当前企业下我能进的平台 (需 token, userId 从 ctx 取, groupId 从请求取)
// type 非空时按 app 类型过滤 (platform=平台, link=链接)
func (s *AuthV2Service) ListMyApps(ctx context.Context, req *pb.ListMyAppsRequest) (*pb.ListMyAppsReply, error) {
	userId := common.ReqCtxFromContext(ctx).GetUserId()
	reply := &pb.ListMyAppsReply{
		Code:    200,
		Message: "",
	}
	isSuper := s.authUC.IsSuperAdminByUserId(ctx, userId)
	apps, err := s.authUC.ListMyApps(ctx, userId, req.GetGroupId(), isSuper, req.GetType())
	if err != nil {
		s.log.WithContext(ctx).Errorf("ListMyApps failed, userId=%d groupId=%d type=%s err=%v", userId, req.GetGroupId(), req.GetType(), err)
		reply.Code = err.GetCode()
		reply.Message = err.GetReason()
		return reply, nil
	}
	data := &pb.ListMyAppsReply_Data{Apps: make([]*pb.AppInfo, 0, len(apps))}
	for _, a := range apps {
		data.Apps = append(data.Apps, toAppInfo(a))
	}
	reply.Data = data
	return reply, nil
}

// CheckUserGroupAccess 校验用户租户归属 (gRPC, 供下游/网关)
func (s *AuthV2Service) CheckUserGroupAccess(ctx context.Context, req *pb.CheckUserGroupAccessRequest) (*pb.CheckUserGroupAccessReply, error) {
	reply := &pb.CheckUserGroupAccessReply{
		Code:    200,
		Message: "",
	}
	has, err := s.authUC.CheckUserGroupAccess(ctx, req.GetUserId(), req.GetGroupId())
	if err != nil {
		s.log.WithContext(ctx).Errorf("CheckUserGroupAccess failed, userId=%d groupId=%d err=%v",
			req.GetUserId(), req.GetGroupId(), err)
		reply.Code = err.GetCode()
		reply.Message = err.GetReason()
		return reply, nil
	}
	reply.Data = &pb.CheckUserGroupAccessReply_Data{HasAccess: has}
	return reply, nil
}

// IssueGroupToken 换发组 token (数据隔离改造: groupId 入 Token)
// 请求携带第一步 LoginV2 返回的基础 token + 前端选择的 groupId;
// 服务端解析基础 token 得到权威 userId, 校验组绑定后签发携带 groupId claim 的新 token。
// 超管传 groupId=0 签发全局 token; 切换分组即重复调用本接口。
func (s *AuthV2Service) IssueGroupToken(ctx context.Context, req *pb.IssueGroupTokenRequest) (*pb.IssueGroupTokenReply, error) {
	reply := &pb.IssueGroupTokenReply{
		Code:    200,
		Message: "",
	}
	result, err := s.authUC.IssueGroupToken(ctx, req.GetToken(), req.GetGroupId())
	if err != nil {
		s.log.WithContext(ctx).Errorf("IssueGroupToken failed, groupId=%d err=%v", req.GetGroupId(), err)
		reply.Code = err.GetCode()
		reply.Message = err.GetReason()
		return reply, nil
	}
	return s.toIssueGroupTokenReply(result), nil
}

// toIssueGroupTokenReply 将换发结果装配为响应 (group 为当前选中企业的角色聚合)
func (s *AuthV2Service) toIssueGroupTokenReply(result *bizV2.LoginResult) *pb.IssueGroupTokenReply {
	data := &pb.IssueGroupTokenReply_Data{
		Token:        result.Token,
		ExpireTime:   result.ExpireTime,
		IsSuperAdmin: result.IsSuperAdmin,
	}
	if result.User != nil && result.User.Id != nil {
		data.UserInfo = &pb.UserInfo{
			UserId:   *result.User.Id,
			Username: result.User.Username,
			Nickname: result.User.Nickname,
		}
	}
	// 换发接口只返回当前选中的企业 (LoginResult.Groups 最多 1 个; 超管为空)
	if len(result.Groups) > 0 {
		data.Group = toGroupInfo(result.Groups[0])
	}
	data.Permissions = make([]*pb.PermissionInfo, 0, len(result.Permissions))
	for _, p := range result.Permissions {
		data.Permissions = append(data.Permissions, toPermissionInfo(p))
	}
	return &pb.IssueGroupTokenReply{Code: 200, Message: "", Data: data}
}

// GetUserInfo 查询当前用户信息 (需 token, userId 从 ctx 取, groupId 从请求取)
// 超管: groupId 可传0, 返回 isSuperAdmin=true, groups/permissions 为空 (默认拥有全部)
// 非超管: groupId 必填, 只返回当前 groupId 对应的权限与企业信息
func (s *AuthV2Service) GetUserInfo(ctx context.Context, req *pb.GetUserInfoRequest) (*pb.GetUserInfoReply, error) {
	userId := common.ReqCtxFromContext(ctx).GetUserId()
	reply := &pb.GetUserInfoReply{
		Code:    200,
		Message: "",
	}
	result, err := s.authUC.GetUserInfo(ctx, userId, req.GetGroupId())
	if err != nil {
		s.log.WithContext(ctx).Errorf("GetUserInfo failed, userId=%d groupId=%d err=%v", userId, req.GetGroupId(), err)
		reply.Code = err.GetCode()
		reply.Message = err.GetReason()
		return reply, nil
	}
	data := &pb.GetUserInfoReply_Data{
		IsSuperAdmin: result.IsSuperAdmin,
	}
	if result.User != nil && result.User.Id != nil {
		data.UserInfo = &pb.UserInfo{
			UserId:   *result.User.Id,
			Username: result.User.Username,
			Nickname: result.User.Nickname,
		}
	}
	data.Groups = make([]*pb.GroupInfo, 0, len(result.Groups))
	for _, g := range result.Groups {
		data.Groups = append(data.Groups, toGroupInfo(g))
	}
	data.Permissions = make([]*pb.PermissionInfo, 0, len(result.Permissions))
	for _, p := range result.Permissions {
		data.Permissions = append(data.Permissions, toPermissionInfo(p))
	}
	reply.Data = data
	return reply, nil
}
