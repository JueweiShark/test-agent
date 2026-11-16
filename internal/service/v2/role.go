package v2

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"

	pb "wuwen.com/wwuc/api_gen/user/v2"
	"wuwen.com/wwuc/internal/biz/common"
	bizV2 "wuwen.com/wwuc/internal/biz/v2"
)

// RoleV2Service 企业角色管理
type RoleV2Service struct {
	pb.UnimplementedRoleV2Server
	roleUC *bizV2.RoleUsecase
	log    *log.Helper
}

// NewRoleV2Service .
func NewRoleV2Service(uc *bizV2.RoleUsecase, logger log.Logger) *RoleV2Service {
	return &RoleV2Service{
		roleUC: uc,
		log:    log.NewHelper(logger),
	}
}

// CreateRole 为指定企业建角色
func (s *RoleV2Service) CreateRole(ctx context.Context, req *pb.CreateRoleV2Request) (*pb.CreateRoleV2Reply, error) {
	userId := common.ReqCtxFromContext(ctx).GetUserId()
	reply := &pb.CreateRoleV2Reply{
		Code:    200,
		Message: "",
	}
	r := &bizV2.Role{
		Name:     req.GetName(),
		GroupId:  req.GetGroupId(),
		Remarks:  req.GetRemarks(),
		CreateBy: userId,
	}
	if err := s.roleUC.CreateRole(ctx, r, req.GetPermissionIds()); err != nil {
		s.log.WithContext(ctx).Errorf("CreateRole failed, name=%s groupId=%d err=%v",
			req.GetName(), req.GetGroupId(), err)
		reply.Code = err.GetCode()
		reply.Message = err.GetReason()
		return reply, nil
	}
	reply.Data = &pb.CreateRoleV2Reply_Data{Role: toRoleInfo(r)}
	return reply, nil
}

// UpdateRole 编辑角色并全量覆盖权限 (System=1 不可改名且权限固定; permission_ids 空=清空)
func (s *RoleV2Service) UpdateRole(ctx context.Context, req *pb.UpdateRoleV2Request) (*pb.UpdateRoleV2Reply, error) {
	userId := common.ReqCtxFromContext(ctx).GetUserId()
	roleId := req.GetRoleId()
	reply := &pb.UpdateRoleV2Reply{
		Code:    200,
		Message: "",
	}
	r := &bizV2.Role{
		Id:       &roleId,
		Name:     req.GetName(),
		Remarks:  req.GetRemarks(),
		UpdateBy: userId,
	}
	if err := s.roleUC.UpdateRole(ctx, r, req.GetPermissionIds()); err != nil {
		s.log.WithContext(ctx).Errorf("UpdateRole failed, roleId=%d err=%v", roleId, err)
		reply.Code = err.GetCode()
		reply.Message = err.GetReason()
		return reply, nil
	}
	return reply, nil
}

// DeleteRole 删除角色 (System=1 不可删)
func (s *RoleV2Service) DeleteRole(ctx context.Context, req *pb.DeleteRoleV2Request) (*pb.DeleteRoleV2Reply, error) {
	userId := common.ReqCtxFromContext(ctx).GetUserId()
	reply := &pb.DeleteRoleV2Reply{
		Code:    200,
		Message: "",
	}
	if err := s.roleUC.DeleteRole(ctx, userId, req.GetRoleId()); err != nil {
		s.log.WithContext(ctx).Errorf("DeleteRole failed, roleId=%d err=%v", req.GetRoleId(), err)
		reply.Code = err.GetCode()
		reply.Message = err.GetReason()
		return reply, nil
	}
	return reply, nil
}

// ListRolesByGroup 查企业下的角色列表 (默认按分页返回全部企业的角色)
// - groupId>0: 按企业ID精准搜; groupName 非空: 按企业名称模糊搜; name 非空: 按角色名称模糊搜
// - 返回字段: 企业ID/企业名称/角色名称/角色编码/创建时间/更新时间 + 角色绑定的菜单权限
func (s *RoleV2Service) ListRolesByGroup(ctx context.Context, req *pb.ListRolesByGroupRequest) (*pb.ListRolesByGroupReply, error) {
	reply := &pb.ListRolesByGroupReply{
		Code:    200,
		Message: "",
	}
	offset, limit := common.GetPageOffset(int64(req.GetPage()), int64(req.GetPageSize()))
	cond := bizV2.QueryRoleCondition{
		GroupId:   req.GetGroupId(),
		Name:      req.GetName(),
		GroupName: req.GetGroupName(),
	}
	roles, total, err := s.roleUC.ListRolesByGroup(ctx, cond, int(offset), int(limit))
	if err != nil {
		s.log.WithContext(ctx).Errorf("ListRolesByGroup failed, groupId=%d err=%v", req.GetGroupId(), err)
		reply.Code = err.GetCode()
		reply.Message = err.GetReason()
		return reply, nil
	}
	if len(roles) == 0 {
		reply.Data = &pb.ListRolesByGroupReply_Data{List: []*pb.RoleDetailInfo{}, Total: total}
		return reply, nil
	}

	// 收集 roleId / groupId 去重, 批量查企业名称与角色权限
	roleIdSet := make(map[int64]bool, len(roles))
	roleIds := make([]int64, 0, len(roles))
	groupIdSet := make(map[int64]bool, len(roles))
	for _, r := range roles {
		if r.Id != nil && !roleIdSet[*r.Id] {
			roleIdSet[*r.Id] = true
			roleIds = append(roleIds, *r.Id)
		}
		if r.GroupId > 0 && !groupIdSet[r.GroupId] {
			groupIdSet[r.GroupId] = true
		}
	}

	// 批量查企业 (groupName 快照)
	groupIds := make([]int64, 0, len(groupIdSet))
	for gid := range groupIdSet {
		groupIds = append(groupIds, gid)
	}
	groupMap := make(map[int64]*bizV2.Group, len(groupIds))
	if len(groupIds) > 0 {
		groups, gerr := s.roleUC.FindGroupsByIds(ctx, groupIds)
		if gerr != nil {
			s.log.WithContext(ctx).Errorf("ListRolesByGroup find groups failed, err=%v", gerr)
		} else {
			for _, g := range groups {
				if g != nil && g.Id != nil {
					groupMap[*g.Id] = g
				}
			}
		}
	}

	// 批量查角色权限 (roleId -> []Permission)
	permsByRole, perr := s.roleUC.FindPermissionsByRoleIds(ctx, roleIds)
	if perr != nil {
		s.log.WithContext(ctx).Errorf("ListRolesByGroup find permissions failed, err=%v", perr)
		permsByRole = make(map[int64][]*bizV2.Permission)
	}

	// 组装返回
	pbRoles := make([]*pb.RoleDetailInfo, 0, len(roles))
	for _, r := range roles {
		groupName := ""
		if g, ok := groupMap[r.GroupId]; ok && g != nil {
			groupName = g.Name
		}
		perms := permsByRole[*r.Id]
		pbPerms := make([]*pb.PermissionInfo, 0, len(perms))
		for _, p := range perms {
			pbPerms = append(pbPerms, toPermissionInfo(p))
		}
		pbRoles = append(pbRoles, toRoleDetailInfo(r, groupName, pbPerms))
	}
	reply.Data = &pb.ListRolesByGroupReply_Data{List: pbRoles, Total: total}
	return reply, nil
}

// AssignRoleToUser 超管给企业加员工 (写 ref_user_group + 多条 ref_user_role)
func (s *RoleV2Service) AssignRoleToUser(ctx context.Context, req *pb.AssignRoleRequest) (*pb.AssignRoleReply, error) {
	operatorId := common.ReqCtxFromContext(ctx).GetUserId()
	reply := &pb.AssignRoleReply{
		Code:    200,
		Message: "",
	}
	if err := s.roleUC.AssignRoleToUser(ctx, req.GetUserId(), req.GetGroupId(),
		req.GetRoleIds(), req.GetStatus(), req.GetRemarks(), operatorId); err != nil {
		s.log.WithContext(ctx).Errorf("AssignRoleToUser failed, userId=%d groupId=%d operator=%d err=%v",
			req.GetUserId(), req.GetGroupId(), operatorId, err)
		reply.Code = err.GetCode()
		reply.Message = err.GetReason()
		return reply, nil
	}
	return reply, nil
}

// UpdateGroupEmployee 修改企业员工的角色/状态/备注
func (s *RoleV2Service) UpdateGroupEmployee(ctx context.Context, req *pb.UpdateGroupEmployeeRequest) (*pb.UpdateGroupEmployeeReply, error) {
	operatorId := common.ReqCtxFromContext(ctx).GetUserId()
	reply := &pb.UpdateGroupEmployeeReply{
		Code:    200,
		Message: "",
	}
	if err := s.roleUC.UpdateGroupEmployee(ctx, req.GetUserId(), req.GetGroupId(),
		req.GetRoleIds(), req.GetStatus(), req.GetRemarks(), operatorId); err != nil {
		s.log.WithContext(ctx).Errorf("UpdateGroupEmployee failed, userId=%d groupId=%d operator=%d err=%v",
			req.GetUserId(), req.GetGroupId(), operatorId, err)
		reply.Code = err.GetCode()
		reply.Message = err.GetReason()
		return reply, nil
	}
	return reply, nil
}

// RemoveRoleFromUser 移除用户角色绑定
func (s *RoleV2Service) RemoveRoleFromUser(ctx context.Context, req *pb.RemoveRoleRequest) (*pb.RemoveRoleReply, error) {
	userId := common.ReqCtxFromContext(ctx).GetUserId()
	reply := &pb.RemoveRoleReply{
		Code:    200,
		Message: "",
	}
	if err := s.roleUC.RemoveRoleFromUser(ctx, req.GetUserId(), req.GetRoleId(), req.GetGroupId()); err != nil {
		s.log.WithContext(ctx).Errorf("RemoveRoleFromUser failed, userId=%d roleId=%d groupId=%d operator=%d err=%v",
			req.GetUserId(), req.GetRoleId(), req.GetGroupId(), userId, err)
		reply.Code = err.GetCode()
		reply.Message = err.GetReason()
		return reply, nil
	}
	return reply, nil
}

// ListUsersByGroup 查企业下员工列表 (一条记录 = 一个员工-企业关系, 含 user/group 快照 + 多角色合并)
func (s *RoleV2Service) ListUsersByGroup(ctx context.Context, req *pb.ListUsersByGroupRequest) (*pb.ListUsersByGroupReply, error) {
	reply := &pb.ListUsersByGroupReply{
		Code:    200,
		Message: "",
	}
	offset, limit := common.GetPageOffset(int64(req.GetPage()), int64(req.GetPageSize()))
	cond := bizV2.UserCondition{}
	if req.GetGroupId() != 0 {
		gid := req.GetGroupId()
		cond.GroupId = &gid
	}
	if req.GetUsername() != "" {
		username := req.GetUsername()
		cond.Username = &username
	}
	if req.GetNickname() != "" {
		nickname := req.GetNickname()
		cond.Nickname = &nickname
	}
	if req.GetGroupName() != "" {
		groupName := req.GetGroupName()
		cond.GroupName = &groupName
	}
	if req.GetStatus() != 0 {
		st := req.GetStatus()
		if st == 2 {
			st = 0
		}
		cond.Status = &st
	}

	ugs, total, err := s.roleUC.ListGroupEmployees(ctx, cond, int(offset), int(limit))
	if err != nil {
		s.log.WithContext(ctx).Errorf("ListUsersByGroup failed, groupId=%d err=%v", req.GetGroupId(), err)
		reply.Code = err.GetCode()
		reply.Message = err.GetReason()
		return reply, nil
	}
	if len(ugs) == 0 {
		reply.Data = &pb.ListUsersByGroupReply_Data{List: []*pb.GroupUserInfo{}, Total: total}
		return reply, nil
	}

	// 收集 userId/groupId/roleId 用于批量查角色
	userIdSet := make(map[int64]bool, len(ugs))
	userIdList := make([]int64, 0, len(ugs))
	groupIdSet := make(map[int64]bool)
	for _, ug := range ugs {
		if !userIdSet[ug.UserId] {
			userIdSet[ug.UserId] = true
			userIdList = append(userIdList, ug.UserId)
		}
		groupIdSet[ug.GroupId] = true
	}

	// 按 (userId, groupId) 聚合 ref_user_role
	// 注意: 当跨企业搜索时 (cond.GroupId == nil), 用户可能在不同企业有多条记录,
	// 需要按 (userId, groupId) 维度分别查角色绑定, 不能简单按 userId 汇总.
	type ugKey struct{ userId, groupId int64 }
	ugKeys := make([]ugKey, 0, len(ugs))
	ugKeySet := make(map[ugKey]bool, len(ugs))
	for _, ug := range ugs {
		k := ugKey{ug.UserId, ug.GroupId}
		if !ugKeySet[k] {
			ugKeySet[k] = true
			ugKeys = append(ugKeys, k)
		}
	}

	// 批量查这些 (userId, groupId) 的角色绑定
	// 由于 RefUserRoleRepo 没有批量按 (userIds, groupIds) 查的接口, 这里按 groupId 分组调用
	groupIds := make([]int64, 0, len(groupIdSet))
	for gid := range groupIdSet {
		groupIds = append(groupIds, gid)
	}
	type roleRef struct {
		UserId  int64
		GroupId int64
		RoleId  int64
	}
	allRefs := make([]roleRef, 0, len(ugs)*2)
	for _, gid := range groupIds {
		refs, rerr := s.roleUC.FindUserRolesByGroupId(ctx, gid)
		if rerr != nil {
			s.log.WithContext(ctx).Errorf("ListUsersByGroup find user roles failed, groupId=%d err=%v", gid, rerr)
			continue
		}
		for _, ref := range refs {
			allRefs = append(allRefs, roleRef{ref.UserId, ref.GroupId, ref.RoleId})
		}
	}

	// 收集所有 roleId 去重, 批量查 Role 信息
	roleIdSet := make(map[int64]bool)
	for _, ref := range allRefs {
		if !roleIdSet[ref.RoleId] {
			roleIdSet[ref.RoleId] = true
		}
	}
	roleIds := make([]int64, 0, len(roleIdSet))
	for rid := range roleIdSet {
		roleIds = append(roleIds, rid)
	}
	roleMap := make(map[int64]*bizV2.Role, len(roleIds))
	if len(roleIds) > 0 {
		roles, rerr := s.roleUC.FindRolesByIds(ctx, roleIds)
		if rerr != nil {
			s.log.WithContext(ctx).Errorf("ListUsersByGroup find roles failed, err=%v", rerr)
		} else {
			for _, r := range roles {
				if r.Id != nil {
					roleMap[*r.Id] = r
				}
			}
		}
	}

	// 按 (userId, groupId) 聚合角色
	type ugKeyRoles struct {
		userId, groupId int64
	}
	rolesByUG := make(map[ugKeyRoles][]*bizV2.Role)
	for _, ref := range allRefs {
		k := ugKeyRoles{ref.UserId, ref.GroupId}
		if r, ok := roleMap[ref.RoleId]; ok && r != nil {
			rolesByUG[k] = append(rolesByUG[k], r)
		}
	}

	// 构造返回
	pbUsers := make([]*pb.GroupUserInfo, 0, len(ugs))
	for _, ug := range ugs {
		info := &pb.GroupUserInfo{
			UserId:    ug.UserId,
			Username:  ug.Username,
			Nickname:  ug.Nickname,
			GroupId:   ug.GroupId,
			GroupName: ug.GroupName,
			GroupCode: ug.GroupCode,
			Status:    int32(ug.Status),
			Remarks:   ug.Remarks,
		}
		if ug.Id != nil {
			info.RefId = *ug.Id
		}
		if ug.CreatedTime != nil {
			info.CreatedAt = ug.CreatedTime.Unix()
		}
		if ug.UpdatedTime != nil {
			info.UpdatedAt = ug.UpdatedTime.Unix()
		}
		roles := rolesByUG[ugKeyRoles{ug.UserId, ug.GroupId}]
		info.Roles = make([]*pb.RoleInfo, 0, len(roles))
		for _, r := range roles {
			info.Roles = append(info.Roles, toRoleInfo(r))
		}
		pbUsers = append(pbUsers, info)
	}
	reply.Data = &pb.ListUsersByGroupReply_Data{List: pbUsers, Total: total}
	return reply, nil
}

// ListUserRolesByGroup 查某用户在某企业的角色
func (s *RoleV2Service) ListUserRolesByGroup(ctx context.Context, req *pb.ListUserRolesRequest) (*pb.ListUserRolesReply, error) {
	reply := &pb.ListUserRolesReply{
		Code:    200,
		Message: "",
	}
	refs, err := s.roleUC.ListUserRolesByGroup(ctx, req.GetUserId(), req.GetGroupId())
	if err != nil {
		s.log.WithContext(ctx).Errorf("ListUserRolesByGroup failed, userId=%d groupId=%d err=%v",
			req.GetUserId(), req.GetGroupId(), err)
		reply.Code = err.GetCode()
		reply.Message = err.GetReason()
		return reply, nil
	}
	roleIdSet := make(map[int64]bool, len(refs))
	roleIds := make([]int64, 0, len(refs))
	for _, ref := range refs {
		if !roleIdSet[ref.RoleId] {
			roleIdSet[ref.RoleId] = true
			roleIds = append(roleIds, ref.RoleId)
		}
	}
	pbRoles := make([]*pb.RoleInfo, 0, len(roleIds))
	if len(roleIds) > 0 {
		roles, rerr := s.roleUC.FindRolesByIds(ctx, roleIds)
		if rerr != nil {
			s.log.WithContext(ctx).Errorf("ListUserRolesByGroup find roles failed, err=%v", rerr)
		} else {
			for _, r := range roles {
				pbRoles = append(pbRoles, toRoleInfo(r))
			}
		}
	}
	reply.Data = &pb.ListUserRolesReply_Data{Roles: pbRoles}
	return reply, nil
}
