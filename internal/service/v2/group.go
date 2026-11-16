package v2

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"

	pb "wuwen.com/wwuc/api_gen/user/v2"
	"wuwen.com/wwuc/internal/biz/common"
	bizV2 "wuwen.com/wwuc/internal/biz/v2"
)

// GroupV2Service 企业(租户)管理
type GroupV2Service struct {
	pb.UnimplementedGroupV2Server
	groupUC *bizV2.GroupUsecase
	log     *log.Helper
}

// NewGroupV2Service .
func NewGroupV2Service(uc *bizV2.GroupUsecase, logger log.Logger) *GroupV2Service {
	return &GroupV2Service{
		groupUC: uc,
		log:     log.NewHelper(logger),
	}
}

// CreateGroup 超管创建企业 (对应 design 2.2 step1)
func (s *GroupV2Service) CreateGroup(ctx context.Context, req *pb.CreateGroupRequest) (*pb.CreateGroupReply, error) {
	userId := common.ReqCtxFromContext(ctx).GetUserId()
	reply := &pb.CreateGroupReply{
		Code:    200,
		Message: "",
	}
	t := &bizV2.Group{
		Name:      req.GetName(),
		GroupCode: req.GetGroupCode(),
		Remarks:   req.GetRemarks(),
		CreatedBy: userId,
	}
	if err := s.groupUC.CreateGroup(ctx, t); err != nil {
		s.log.WithContext(ctx).Errorf("CreateGroup failed, name=%s err=%v", req.GetName(), err)
		reply.Code = err.GetCode()
		reply.Message = err.GetReason()
		return reply, nil
	}
	apps, err := s.groupUC.ListGroupApps(ctx, *t.Id)
	if err != nil {
		s.log.WithContext(ctx).Errorf("CreateGroup list apps failed, groupId=%d err=%v", *t.Id, err)
		reply.Code = err.GetCode()
		reply.Message = err.GetReason()
		return reply, nil
	}
	reply.Data = &pb.CreateGroupReply_Data{Group: toGroupGroupInfo(t, apps)}
	return reply, nil
}

// UpdateGroup 编辑企业 (status=0 视为未变更, 避免 proto3 默认值误停用)
func (s *GroupV2Service) UpdateGroup(ctx context.Context, req *pb.UpdateGroupRequest) (*pb.UpdateGroupReply, error) {
	userId := common.ReqCtxFromContext(ctx).GetUserId()
	groupId := req.GetGroupId()
	reply := &pb.UpdateGroupReply{
		Code:    200,
		Message: "",
	}
	t := &bizV2.Group{
		Id:        &groupId,
		Name:      req.GetName(),
		GroupCode: req.GetGroupCode(),
		Remarks:   req.GetRemarks(),
		Status:    int8(req.GetStatus()),
		UpdatedBy: userId,
	}
	if err := s.groupUC.UpdateGroup(ctx, t); err != nil {
		s.log.WithContext(ctx).Errorf("UpdateGroup failed, groupId=%d err=%v", groupId, err)
		reply.Code = err.GetCode()
		reply.Message = err.GetReason()
		return reply, nil
	}
	return reply, nil
}

// ListGroups 企业分页列表
func (s *GroupV2Service) ListGroups(ctx context.Context, req *pb.ListGroupsRequest) (*pb.ListGroupsReply, error) {
	reply := &pb.ListGroupsReply{
		Code:    200,
		Message: "",
	}
	offset, limit := common.GetPageOffset(int64(req.GetPage()), int64(req.GetPageSize()))
	cond := bizV2.GroupCondition{}
	if req.GetName() != "" {
		name := req.GetName()
		cond.Name = &name
	}
	if req.GetGroupCode() != "" {
		code := req.GetGroupCode()
		cond.GroupCode = &code
	}
	if req.GetStatus() > 0 {
		status := req.GetStatus()
		cond.Status = &status
	}
	if len(req.GetIds()) > 0 {
		ids := make([]int64, 0, len(req.GetIds())+1)
		ids = append(ids, req.GetIds()...)
		if req.GetId() > 0 {
			ids = append(ids, req.GetId())
		}
		cond.Ids = ids
	} else if req.GetId() > 0 {
		id := req.GetId()
		cond.Ids = []int64{id}
	}
	groups, total, err := s.groupUC.ListGroups(ctx, cond, int(offset), int(limit))
	if err != nil {
		s.log.WithContext(ctx).Errorf("ListGroups failed, err=%v", err)
		reply.Code = err.GetCode()
		reply.Message = err.GetReason()
		return reply, nil
	}
	groupIds := make([]int64, 0, len(groups))
	for _, t := range groups {
		if t != nil && t.Id != nil {
			groupIds = append(groupIds, *t.Id)
		}
	}
	groupApps, err := s.groupUC.ListGroupAppsByGroupIds(ctx, groupIds)
	if err != nil {
		s.log.WithContext(ctx).Errorf("ListGroups list apps failed, err=%v", err)
		reply.Code = err.GetCode()
		reply.Message = err.GetReason()
		return reply, nil
	}
	groupInfos := make([]*pb.GroupInfo, 0, len(groups))
	for _, t := range groups {
		groupInfos = append(groupInfos, toGroupGroupInfo(t, groupApps[*t.Id]))
	}
	reply.Data = &pb.ListGroupsReply_Data{List: groupInfos, Total: total}
	return reply, nil
}

// GetGroup 企业详情
func (s *GroupV2Service) GetGroup(ctx context.Context, req *pb.GetGroupRequest) (*pb.GetGroupReply, error) {
	reply := &pb.GetGroupReply{
		Code:    200,
		Message: "",
	}
	t, err := s.groupUC.GetGroup(ctx, req.GetGroupId())
	if err != nil {
		s.log.WithContext(ctx).Errorf("GetGroup failed, groupId=%d err=%v", req.GetGroupId(), err)
		reply.Code = err.GetCode()
		reply.Message = err.GetReason()
		return reply, nil
	}
	apps, err := s.groupUC.ListGroupApps(ctx, req.GetGroupId())
	if err != nil {
		s.log.WithContext(ctx).Errorf("GetGroup list apps failed, groupId=%d err=%v", req.GetGroupId(), err)
		reply.Code = err.GetCode()
		reply.Message = err.GetReason()
		return reply, nil
	}
	reply.Data = &pb.GetGroupReply_Data{Group: toGroupGroupInfo(t, apps)}
	return reply, nil
}

// DeleteGroup 软删企业
func (s *GroupV2Service) DeleteGroup(ctx context.Context, req *pb.DeleteGroupRequest) (*pb.DeleteGroupReply, error) {
	userId := common.ReqCtxFromContext(ctx).GetUserId()
	reply := &pb.DeleteGroupReply{
		Code:    200,
		Message: "",
	}
	if err := s.groupUC.DeleteGroup(ctx, userId, req.GetGroupId()); err != nil {
		s.log.WithContext(ctx).Errorf("DeleteGroup failed, groupId=%d err=%v", req.GetGroupId(), err)
		reply.Code = err.GetCode()
		reply.Message = err.GetReason()
		return reply, nil
	}
	return reply, nil
}

// SetGroupPermissions 配置企业权限范围 (全量覆盖 + 级联软删越界角色权限, 对应 design 4.2.2)
func (s *GroupV2Service) SetGroupPermissions(ctx context.Context, req *pb.SetGroupPermissionsRequest) (*pb.SetGroupPermissionsReply, error) {
	userId := common.ReqCtxFromContext(ctx).GetUserId()
	reply := &pb.SetGroupPermissionsReply{
		Code:    200,
		Message: "",
	}
	if err := s.groupUC.SetGroupPermissions(ctx, userId, req.GetGroupId(), req.GetPermissionIds()); err != nil {
		s.log.WithContext(ctx).Errorf("SetGroupPermissions failed, groupId=%d permIds=%v err=%v",
			req.GetGroupId(), req.GetPermissionIds(), err)
		reply.Code = err.GetCode()
		reply.Message = err.GetReason()
		return reply, nil
	}
	return reply, nil
}

// ListGroupPermissions 查企业权限范围
func (s *GroupV2Service) ListGroupPermissions(ctx context.Context, req *pb.ListGroupPermissionsRequest) (*pb.ListGroupPermissionsReply, error) {
	reply := &pb.ListGroupPermissionsReply{
		Code:    200,
		Message: "",
	}
	refs, err := s.groupUC.ListGroupPermissions(ctx, req.GetGroupId())
	if err != nil {
		s.log.WithContext(ctx).Errorf("ListGroupPermissions failed, groupId=%d err=%v", req.GetGroupId(), err)
		reply.Code = err.GetCode()
		reply.Message = err.GetReason()
		return reply, nil
	}
	permIds := make([]int64, 0, len(refs))
	for _, r := range refs {
		permIds = append(permIds, r.PermissionId)
	}
	reply.Data = &pb.ListGroupPermissionsReply_Data{PermissionIds: permIds}
	return reply, nil
}

// toGroupGroupInfo 将 biz Group 转换为 proto GroupInfo (不含 roles, 仅企业信息 + 关联的 app 列表)
// apps 为该企业 scope 内 permission.app 去重推导出的可用产品列表 (design 4.2.4)
func toGroupGroupInfo(t *bizV2.Group, apps []*bizV2.App) *pb.GroupInfo {
	if t == nil || t.Id == nil {
		return nil
	}
	out := &pb.GroupInfo{
		GroupId:   *t.Id,
		GroupName: t.Name,
		GroupCode: t.GroupCode,
		Status:    int32(t.Status),
		Remarks: t.Remarks,
	}
	// 注意: GroupCreatedAt(企业创建时间, proto field 4) 与 CreatedAt(记录创建时间, field 8) 是两个不同字段, 不可混淆
	if t.GroupCreatedAt != nil {
		out.GroupCreatedAt = t.GroupCreatedAt.Unix()
	}
	if t.CreatedAt != nil {
		out.CreatedAt = t.CreatedAt.Unix()
	}
	if t.UpdatedAt != nil {
		out.UpdatedAt = t.UpdatedAt.Unix()
	}
	out.Apps = make([]*pb.AppInfo, 0, len(apps))
	for _, a := range apps {
		if a == nil {
			continue
		}
		out.Apps = append(out.Apps, toAppInfo(a))
	}
	return out
}
