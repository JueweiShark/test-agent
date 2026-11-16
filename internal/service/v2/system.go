package v2

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"

	pb "wuwen.com/wwuc/api_gen/user/v2"
	"wuwen.com/wwuc/internal/biz/common"
	bizV2 "wuwen.com/wwuc/internal/biz/v2"
)

// SystemV2Service 系统配置 (登录页配置/产品 app 维护)
type SystemV2Service struct {
	pb.UnimplementedSystemV2Server
	systemUC *bizV2.SystemUsecase
	authUC   *bizV2.AuthUsecase
	log      *log.Helper
}

// NewSystemV2Service .
func NewSystemV2Service(uc *bizV2.SystemUsecase, authUC *bizV2.AuthUsecase, logger log.Logger) *SystemV2Service {
	return &SystemV2Service{
		systemUC: uc,
		authUC:   authUC,
		log:      log.NewHelper(logger),
	}
}

// GetSystemConfig 获取配置 (key 为空则返回全部)
func (s *SystemV2Service) GetSystemConfig(ctx context.Context, req *pb.GetSystemConfigRequest) (*pb.GetSystemConfigReply, error) {
	reply := &pb.GetSystemConfigReply{
		Code:    200,
		Message: "",
	}
	configs, err := s.systemUC.GetSystemConfig(ctx, req.GetConfigKey())
	if err != nil {
		s.log.WithContext(ctx).Errorf("GetSystemConfig failed, key=%s err=%v", req.GetConfigKey(), err)
		reply.Code = err.GetCode()
		reply.Message = err.GetReason()
		return reply, nil
	}
	reply.Data = &pb.GetSystemConfigReply_Data{Configs: configs}
	return reply, nil
}

// SetSystemConfig 批量设置配置 (upsert)
func (s *SystemV2Service) SetSystemConfig(ctx context.Context, req *pb.SetSystemConfigRequest) (*pb.SetSystemConfigReply, error) {
	userId := common.ReqCtxFromContext(ctx).GetUserId()
	reply := &pb.SetSystemConfigReply{
		Code:    200,
		Message: "",
	}
	configs := req.GetConfigs()
	if len(configs) == 0 {
		return reply, nil
	}
	if err := s.systemUC.SetSystemConfig(ctx, userId, configs); err != nil {
		s.log.WithContext(ctx).Errorf("SetSystemConfig failed, operator=%d keys=%v err=%v", userId, configs, err)
		reply.Code = err.GetCode()
		reply.Message = err.GetReason()
		return reply, nil
	}
	return reply, nil
}

// ListApps 拿当前用户在指定企业内有权限的产品列表 (总览图标)
// 超管返回所有启用产品; 普通员工返回该企业内有效权限涉及的产品去重
// type 非空时按 app 类型过滤 (platform=平台, link=链接)
func (s *SystemV2Service) ListApps(ctx context.Context, req *pb.ListAppsRequest) (*pb.ListAppsReply, error) {
	userId := common.ReqCtxFromContext(ctx).GetUserId()
	reply := &pb.ListAppsReply{
		Code:    200,
		Message: "",
	}
	isSuper := s.authUC.IsSuperAdminByUserId(ctx, userId)
	isSuper = true
	apps, err := s.authUC.ListMyApps(ctx, userId, req.GetGroupId(), isSuper, req.GetType())
	if err != nil {
		s.log.WithContext(ctx).Errorf("ListApps failed, userId=%d groupId=%d type=%s err=%v", userId, req.GetGroupId(), req.GetType(), err)
		reply.Code = err.GetCode()
		reply.Message = err.GetReason()
		return reply, nil
	}
	pbApps := make([]*pb.AppInfo, 0, len(apps))
	for _, a := range apps {
		pbApps = append(pbApps, toAppInfo(a))
	}
	reply.Data = &pb.ListAppsReply_Data{Apps: pbApps}
	return reply, nil
}

// ListAppsByPage 分页查询产品列表 (管理端)
func (s *SystemV2Service) ListAppsByPage(ctx context.Context, req *pb.ListAppsByPageRequest) (*pb.ListAppsByPageReply, error) {
	reply := &pb.ListAppsByPageReply{
		Code:    200,
		Message: "",
	}
	offset, limit := common.GetPageOffset(int64(req.GetPage()), int64(req.GetPageSize()))
	cond := bizV2.AppCondition{}
	if req.GetName() != "" {
		name := req.GetName()
		cond.Name = &name
	}
	if req.GetCode() != "" {
		code := req.GetCode()
		cond.Code = &code
	}
	if req.GetStatus() > 0 {
		status := req.GetStatus()
		cond.Status = &status
	}
	if req.GetType() != "" {
		t := req.GetType()
		cond.Type = &t
	}
	apps, total, err := s.systemUC.ListAppsByPage(ctx, cond, int(offset), int(limit))
	if err != nil {
		s.log.WithContext(ctx).Errorf("ListAppsByPage failed, err=%v", err)
		reply.Code = err.GetCode()
		reply.Message = err.GetReason()
		return reply, nil
	}
	pbApps := make([]*pb.AppInfo, 0, len(apps))
	for _, a := range apps {
		pbApps = append(pbApps, toAppInfo(a))
	}
	reply.Data = &pb.ListAppsByPageReply_Data{List: pbApps, Total: total}
	return reply, nil
}

// CreateApp 超管创建产品
func (s *SystemV2Service) CreateApp(ctx context.Context, req *pb.CreateAppRequest) (*pb.CreateAppReply, error) {
	userId := common.ReqCtxFromContext(ctx).GetUserId()
	reply := &pb.CreateAppReply{
		Code:    200,
		Message: "",
	}
	a := &bizV2.App{
		Name:      req.GetName(),
		Code:      req.GetCode(),
		EntryUrl:  req.GetEntryUrl(),
		Icon:      req.GetIcon(),
		Sort:      req.GetSort(),
		Type:      req.GetType(),
		Desc:      req.GetDesc(),
		Detail:    req.GetDetail(),
		Status:    int8(req.GetStatus()),
		CreatedBy: userId,
	}
	if err := s.systemUC.CreateApp(ctx, a); err != nil {
		s.log.WithContext(ctx).Errorf("CreateApp failed, name=%s err=%v", req.GetName(), err)
		reply.Code = err.GetCode()
		reply.Message = err.GetReason()
		return reply, nil
	}
	reply.Data = &pb.CreateAppReply_Data{App: toAppInfo(a)}
	return reply, nil
}

// UpdateApp 超管更新产品 (status=0 视为未变更)
func (s *SystemV2Service) UpdateApp(ctx context.Context, req *pb.UpdateAppRequest) (*pb.UpdateAppReply, error) {
	userId := common.ReqCtxFromContext(ctx).GetUserId()
	appId := req.GetAppId()
	reply := &pb.UpdateAppReply{
		Code:    200,
		Message: "",
	}
	a := &bizV2.App{
		Id:        &appId,
		Name:      req.GetName(),
		Code:      req.GetCode(),
		EntryUrl:  req.GetEntryUrl(),
		Icon:      req.GetIcon(),
		Sort:      req.GetSort(),
		Type:      req.GetType(),
		Desc:      req.GetDesc(),
		Detail:    req.GetDetail(),
		Status:    int8(req.GetStatus()),
		UpdatedBy: userId,
	}
	if err := s.systemUC.UpdateApp(ctx, a); err != nil {
		s.log.WithContext(ctx).Errorf("UpdateApp failed, appId=%d err=%v", req.GetAppId(), err)
		reply.Code = err.GetCode()
		reply.Message = err.GetReason()
		return reply, nil
	}
	return reply, nil
}

// DeleteApp 超管删除产品 (软删)
func (s *SystemV2Service) DeleteApp(ctx context.Context, req *pb.DeleteAppRequest) (*pb.DeleteAppReply, error) {
	userId := common.ReqCtxFromContext(ctx).GetUserId()
	appId := req.GetAppId()
	reply := &pb.DeleteAppReply{
		Code:    200,
		Message: "",
	}
	if err := s.systemUC.DeleteApp(ctx, appId, userId); err != nil {
		s.log.WithContext(ctx).Errorf("DeleteApp failed, appId=%d err=%v", appId, err)
		reply.Code = err.GetCode()
		reply.Message = err.GetReason()
		return reply, nil
	}
	return reply, nil
}
