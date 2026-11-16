package v2

import (
    pb "wuwen.com/wwuc/api_gen/user/v2"
    bizV2 "wuwen.com/wwuc/internal/biz/v2"
)

// toGroupInfo 将 biz 的 GroupGroup 转换为 proto GroupInfo (含 roles)
func toGroupInfo(g *bizV2.GroupGroup) *pb.GroupInfo {
    if g == nil || g.Group == nil || g.Group.Id == nil {
        return nil
    }
    out := &pb.GroupInfo{
        GroupId:   *g.Group.Id,
        GroupName: g.Group.Name,
        GroupCode: g.Group.GroupCode,
        Status:    int32(g.Group.Status),
        CreatedAt: g.Group.CreatedAt.Unix(),
    }
    if g.Group.GroupCreatedAt != nil {
        out.GroupCreatedAt = g.Group.GroupCreatedAt.Unix()
    }
    if g.Group.UpdatedAt!=nil {
        out.UpdatedAt = g.Group.UpdatedAt.Unix()
    }
    out.Roles = make([]*pb.RoleInfo, 0, len(g.Roles))
    for _, r := range g.Roles {
        out.Roles = append(out.Roles, toRoleInfo(r))
    }
    return out
}

// toRoleInfo 将 biz Role 转换为 proto RoleInfo
func toRoleInfo(r *bizV2.Role) *pb.RoleInfo {
    if r == nil || r.Id == nil {
        return nil
    }
    return &pb.RoleInfo{
        RoleId:   *r.Id,
        RoleName: r.Name,
        RoleCode: r.Code,
        Remarks:  r.Remarks,
        System:   int32(r.System),
        GroupId:  r.GroupId,
        App:      r.App,
    }
}

// toRoleDetailInfo 将 biz Role + 企业/权限快照转换为 proto RoleDetailInfo
// groupName 来自批量查询的 group 快照; perms 是该角色绑定的权限列表 (可为空切片)
func toRoleDetailInfo(r *bizV2.Role, groupName string, perms []*pb.PermissionInfo) *pb.RoleDetailInfo {
    if r == nil || r.Id == nil {
        return nil
    }
    out := &pb.RoleDetailInfo{
        RoleId:      *r.Id,
        GroupId:      r.GroupId,
        GroupName:    groupName,
        RoleName:     r.Name,
        RoleCode:     r.Code,
        Remarks:      r.Remarks,
        System:       int32(r.System),
        App:          r.App,
        Permissions:  perms,
    }
    if r.CreateAt != nil {
        out.CreatedAt = r.CreateAt.Unix()
    }
    if r.UpdateAt != nil {
        out.UpdatedAt = r.UpdateAt.Unix()
    }
    return out
}

func toAppInfo(a *bizV2.App) *pb.AppInfo {
    if a == nil {
        return nil
    }
    out := &pb.AppInfo{
        Name:     a.Name,
        Code:     a.Code,
        EntryUrl: a.EntryUrl,
        Icon:     a.Icon,
        Sort:     a.Sort,
        Status:   int32(a.Status),
        Type:     a.Type,
        Desc:     a.Desc,
        Detail: a.Detail,
        IconUrl: a.IconUrl,
    }
    if a.Id != nil {
        out.Id = *a.Id
    }
    return out
}

// toPermissionInfo 将 biz Permission 转换为 proto PermissionInfo
func toPermissionInfo(p *bizV2.Permission) *pb.PermissionInfo {
    if p == nil {
        return nil
    }
    out := &pb.PermissionInfo{
        Id:                p.Id,
        Name:              p.Name,
        Type:              int32(p.Type),
        RoutePath:         p.RoutePath,
        Icon:              p.Icon,
        Sort:              p.Sort,
        App:               p.App,
        Path:              p.Path,
        Code:              p.Code,
        DependenceChildren: p.DependenceChildren,
        DependenceParents: p.DependenceParents,
    }
    if p.MenuType != nil {
        out.MenuType = int32(*p.MenuType)
    }
    if p.Act != nil {
        out.Act = *p.Act
    }
    if p.Hidden != nil {
        out.Hidden = int32(*p.Hidden)
    } else {
        out.Hidden = 1 // 默认显示
    }
    if p.Description != nil {
        out.Description = *p.Description
    }
    return out
}

// toUserDetailInfo 将 biz User 转换为 proto UserDetailInfo (用于用户列表/详情)
func toUserDetailInfo(u *bizV2.User) *pb.UserDetailInfo {
    if u == nil || u.Id == nil {
        return nil
    }
    out := &pb.UserDetailInfo{
        UserId:   *u.Id,
        Username: u.Username,
        Nickname: u.Nickname,
        Status:   int32(u.Status),
        Remarks:  u.Remarks,
    }
    if !u.CreatedAt.IsZero() {
        out.CreatedAt = u.CreatedAt.Unix()
    }
    if u.LastLoginAt != nil {
        out.LastLoginAt = u.LastLoginAt.Unix()
    }
    return out
}
