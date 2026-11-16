package v2

import (
	"context"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"

	"wuwen.com/wwuc/util/wuwen"
)

// MenuUsecase 菜单管理 (复用 permission 表, type=1 的节点)
type MenuUsecase struct {
	log *log.Helper
	pr  PermissionRepo
}

// NewMenuUsecase .
func NewMenuUsecase(logger log.Logger, pr PermissionRepo) *MenuUsecase {
	return &MenuUsecase{log: log.NewHelper(logger), pr: pr}
}

// CreateMenu 建菜单节点
func (uc *MenuUsecase) CreateMenu(ctx context.Context, p *Permission) *errors.Error {
	if p.Type == 0 {
		p.Type = 1 // 默认前端菜单
	}
	// permission.code 有 UNIQUE 约束且 NOT NULL DEFAULT 0.
	// v2 API 不暴露 code 字段 (调用方不传), 这里在 code 缺省(0)时自动生成唯一值,
	// 避免多次建菜单都落到 code=0 触发唯一键冲突.
	if p.Code == 0 {
		p.Code = wuwen.GenerateSonyNum()
	}
	if err := uc.pr.Create(ctx, p); err != nil {
		uc.log.WithContext(ctx).Errorf("create menu failed: %v", err)
		return errInternalError
	}
	return nil
}

// UpdateMenu 编辑菜单 (只允许改: 名称/排序/显隐/备注, 其余字段一律保留 old 值)
func (uc *MenuUsecase) UpdateMenu(ctx context.Context, p *Permission) *errors.Error {
	old, err := uc.pr.FindById(ctx, p.Id)
	if err != nil || old == nil {
		return errMenuNotFound
	}
	// 保留所有不可变字段 (调用方只传 name/sort/hidden/description)
	p.App = old.App
	p.Type = old.Type
	p.Code = old.Code
	p.Domain = old.Domain
	p.MenuType = old.MenuType
	p.RoutePath = old.RoutePath
	p.Icon = old.Icon
	p.Path = old.Path
	p.Act = old.Act
	p.DependenceChildren = old.DependenceChildren
	p.DependenceParents = old.DependenceParents
	// hidden: nil=不改; 非 nil 则用传入值
	if p.Hidden == nil {
		p.Hidden = old.Hidden
	}
	// name: 空串=不改; 非空则用传入值 (Name 非 optional, 用空串作"未传"哨兵)
	if p.Name == "" {
		p.Name = old.Name
	}
	// sort: 0=不改; 非0则用传入值 (Sort 非 optional, 用 0 作"未传"哨兵)
	//if p.Sort == 0 {
	//	p.Sort = old.Sort
	//}
	// description: nil=不改; 非 nil (含空串) 则用传入值
	if p.Description == nil {
		p.Description = old.Description
	}
	if err := uc.pr.Update(ctx, p); err != nil {
		uc.log.WithContext(ctx).Errorf("update menu failed: %v", err)
		return errInternalError
	}
	return nil
}

// DeleteMenu 删除菜单 (有子节点则拒绝)
func (uc *MenuUsecase) DeleteMenu(ctx context.Context, userId, id int64) *errors.Error {
	if _, err := uc.pr.FindById(ctx, id); err != nil {
		return errMenuNotFound
	}
	count, err := uc.pr.CountChildren(ctx, id)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("count children failed: %v", err)
		return errInternalError
	}
	if count > 0 {
		return errMenuHasChildren
	}
	if err := uc.pr.Delete(ctx, userId, id); err != nil {
		uc.log.WithContext(ctx).Errorf("delete menu failed: %v", err)
		return errInternalError
	}
	return nil
}

// ListMenus 按 app/父节点/类型查菜单列表
// groupId > 0 时按企业 scope 过滤 (join ref_group_permission), 否则全量
func (uc *MenuUsecase) ListMenus(ctx context.Context, app, parent, groupId int64, menuType int32) ([]*Permission, *errors.Error) {
	var all []*Permission
	var err error
	if groupId > 0 {
		all, err = uc.pr.FindByAppAndGroupId(ctx, app, groupId, false)
	} else {
		all, err = uc.pr.FindByApp(ctx, app, false)
	}
	if err != nil {
		uc.log.WithContext(ctx).Errorf("list menus failed: %v", err)
		return nil, errInternalError
	}
	out := make([]*Permission, 0, len(all))
	for _, p := range all {
		if parent > 0 && p.DependenceParents != parent {
			continue
		}
		if menuType > 0 {
			if p.MenuType == nil || *p.MenuType != uint8(menuType) {
				continue
			}
		}
		out = append(out, p)
	}
	return out, nil
}

// GetMenuTree 按 app 拿菜单树
// groupId > 0 时按企业 scope 过滤 (join ref_group_permission), 否则全量
//
// 注意: permission.dependence_parents 存的是父节点的 code, 不是 id.
// 因此 children map 的 key 必须是子节点的 DependenceParents (即父的 code),
// 调用方在递归时也必须用父节点的 Code 去查 children, 不能用 Id.
func (uc *MenuUsecase) GetMenuTree(ctx context.Context, app, groupId int64, hidden bool) ([]*Permission, map[int64][]*Permission, *errors.Error) {
	var all []*Permission
	var err error
	if groupId > 0 {
		all, err = uc.pr.FindByAppAndGroupId(ctx, app, groupId, hidden)
	} else {
		all, err = uc.pr.FindByApp(ctx, app, hidden)
	}
	if err != nil {
		uc.log.WithContext(ctx).Errorf("get menu tree failed: %v", err)
		return nil, nil, errInternalError
	}
	// 用 code 作索引: key=父节点 code, value=该父下的所有子节点.
	// DependenceParents==0 表示根节点.
	roots := make([]*Permission, 0, len(all))
	children := make(map[int64][]*Permission, len(all))
	for _, p := range all {
		if p.DependenceParents == 0 {
			roots = append(roots, p)
		} else {
			children[p.DependenceParents] = append(children[p.DependenceParents], p)
		}
	}
	return roots, children, nil
}
