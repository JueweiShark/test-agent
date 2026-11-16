package v2

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"

	pb "wuwen.com/wwuc/api_gen/user/v2"
	"wuwen.com/wwuc/internal/biz/common"
	bizV2 "wuwen.com/wwuc/internal/biz/v2"
)

// MenuV2Service 菜单管理 (复用 permission 表, type=1 的节点)
type MenuV2Service struct {
	pb.UnimplementedMenuV2Server
	menuUC *bizV2.MenuUsecase
	log    *log.Helper
}

// NewMenuV2Service .
func NewMenuV2Service(uc *bizV2.MenuUsecase, logger log.Logger) *MenuV2Service {
	return &MenuV2Service{
		menuUC: uc,
		log:    log.NewHelper(logger),
	}
}

// CreateMenu 建菜单节点
func (s *MenuV2Service) CreateMenu(ctx context.Context, req *pb.CreateMenuRequest) (*pb.CreateMenuReply, error) {
	userId := common.ReqCtxFromContext(ctx).GetUserId()
	reply := &pb.CreateMenuReply{
		Code:    200,
		Message: "",
	}
	menuType := uint8(req.GetMenuType())
	p := &bizV2.Permission{
		Name:               req.GetName(),
		Type:               uint8(req.GetType()),
		MenuType:           &menuType,
		App:                req.GetApp(),
		RoutePath:          req.GetRoutePath(),
		Icon:               req.GetIcon(),
		Sort:               req.GetSort(),
		Path:               req.GetPath(),
		Act:                strPtrOrNil(req.GetAct()),
		DependenceParents:  req.GetDependenceParents(),
		Domain:             req.GetDomain(),
		CreateBy:           userId,
	}
	if err := s.menuUC.CreateMenu(ctx, p); err != nil {
		s.log.WithContext(ctx).Errorf("CreateMenu failed, name=%s err=%v", req.GetName(), err)
		reply.Code = err.GetCode()
		reply.Message = err.GetReason()
		return reply, nil
	}
	reply.Data = &pb.CreateMenuReply_Data{Menu: toPermissionInfo(p)}
	return reply, nil
}

// UpdateMenu 编辑菜单 (只允许改: 名称/排序/显隐/备注)
func (s *MenuV2Service) UpdateMenu(ctx context.Context, req *pb.UpdateMenuRequest) (*pb.UpdateMenuReply, error) {
	userId := common.ReqCtxFromContext(ctx).GetUserId()
	reply := &pb.UpdateMenuReply{
		Code:    200,
		Message: "",
	}
	p := &bizV2.Permission{
		Id:       req.GetId(),
		Name:     req.GetName(),
		Sort:     req.GetSort(),
		UpdateBy: userId,
	}
	// hidden: optional, nil=不改
	if req.Hidden != nil {
		h := uint8(req.GetHidden())
		p.Hidden = &h
	}
	// description: optional, nil=不改; 传值(含空串)则更新, 空串表示清空
	if req.Description != nil {
		d := req.GetDescription()
		p.Description = &d
	}
	if err := s.menuUC.UpdateMenu(ctx, p); err != nil {
		s.log.WithContext(ctx).Errorf("UpdateMenu failed, id=%d err=%v", req.GetId(), err)
		reply.Code = err.GetCode()
		reply.Message = err.GetReason()
		return reply, nil
	}
	return reply, nil
}

// DeleteMenu 删除菜单 (有子节点则拒绝)
func (s *MenuV2Service) DeleteMenu(ctx context.Context, req *pb.DeleteMenuRequest) (*pb.DeleteMenuReply, error) {
	userId := common.ReqCtxFromContext(ctx).GetUserId()
	reply := &pb.DeleteMenuReply{
		Code:    200,
		Message: "",
	}
	if err := s.menuUC.DeleteMenu(ctx, userId, req.GetId()); err != nil {
		s.log.WithContext(ctx).Errorf("DeleteMenu failed, id=%d err=%v", req.GetId(), err)
		reply.Code = err.GetCode()
		reply.Message = err.GetReason()
		return reply, nil
	}
	return reply, nil
}

// ListMenus 按 app/父节点/类型查菜单列表
// group_id > 0 时只返回该企业 scope 内的菜单 (权限继承)
func (s *MenuV2Service) ListMenus(ctx context.Context, req *pb.ListMenusRequest) (*pb.ListMenusReply, error) {
	reply := &pb.ListMenusReply{
		Code:    200,
		Message: "",
	}
	menus, err := s.menuUC.ListMenus(ctx, req.GetApp(), req.GetDependenceParents(), req.GetGroupId(), req.GetMenuType())
	if err != nil {
		s.log.WithContext(ctx).Errorf("ListMenus failed, app=%d err=%v", req.GetApp(), err)
		reply.Code = err.GetCode()
		reply.Message = err.GetReason()
		return reply, nil
	}
	pbMenus := make([]*pb.PermissionInfo, 0, len(menus))
	for _, m := range menus {
		pbMenus = append(pbMenus, toPermissionInfo(m))
	}
	reply.Data = &pb.ListMenusReply_Data{Menus: pbMenus}
	return reply, nil
}

// GetMenuTree 按 app 拿菜单树 (递归构建)
// group_id > 0 时只返回该企业 scope 内的菜单 (权限继承)
func (s *MenuV2Service) GetMenuTree(ctx context.Context, req *pb.GetMenuTreeRequest) (*pb.GetMenuTreeReply, error) {
	reply := &pb.GetMenuTreeReply{
		Code:    200,
		Message: "",
	}
	roots, children, err := s.menuUC.GetMenuTree(ctx, req.GetApp(), req.GetGroupId(), req.GetHidden())
	if err != nil {
		s.log.WithContext(ctx).Errorf("GetMenuTree failed, app=%d err=%v", req.GetApp(), err)
		reply.Code = err.GetCode()
		reply.Message = err.GetReason()
		return reply, nil
	}
	tree := buildMenuTree(roots, children)
	reply.Data = &pb.GetMenuTreeReply_Data{Tree: tree}
	return reply, nil
}

// buildMenuTree 递归构建菜单树
// 注意: permission.dependence_parents 存的是父节点的 code, 不是 id; 因此 children map 的 key
// 必须是 code (子节点的 dependence_parents), 查找时也用父节点的 code 匹配.
func buildMenuTree(roots []*bizV2.Permission, children map[int64][]*bizV2.Permission) []*pb.MenuNode {
	if len(roots) == 0 {
		return []*pb.MenuNode{}
	}
	out := make([]*pb.MenuNode, 0, len(roots))
	for _, r := range roots {
		node := &pb.MenuNode{
			Info:     toPermissionInfo(r),
			// dependence_parents 存的是父节点的 code, 所以 children map 以 code 为 key,
			// 这里必须用 r.Code 查找, 不能用 r.Id
			Children: buildMenuTree(children[r.Code], children),
		}
		out = append(out, node)
	}
	return out
}

// strPtrOrNil 空字符串返回 nil, 否则返回 *string
func strPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
