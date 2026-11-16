package errors

import ke "github.com/go-kratos/kratos/v2/errors"

// v2 使用 30000 起步
const (
    baseUserV2ErrCode = 30000
)

const (
    codeUserNoGroup          = iota + baseUserV2ErrCode // 30000 用户未关联任何租户
    codeGroupNotFound                                   // 30001 租户不存在
    codeGroupNameExists                                 // 30002 租户名称已存在
    codeGroupCodeExists                                 // 30003 租户编码已存在
    codeRoleNotFound                                    // 30004 角色不存在
    codeRoleNameExists                                  // 30005 角色名称在租户内已存在
    codeRoleSystemCannotEdit                            // 30006 系统内置角色不可编辑/删除
    codePermissionNotFound                              // 30007 权限节点不存在
    codePermissionOutOfScope                            // 30008 权限越界(不在企业 scope 内)
    codeGroupHasUsers                                   // 30009 租户下仍有员工,无法删除
    codeUserAlreadyInGroup                              // 30010 员工已属于该租户
    codeUserNotInGroup                                  // 30011 员工不属于该租户
    codeRoleNotInGroup                                  // 30012 角色不属于该租户
    codeMenuNotFound                                    // 30013 菜单节点不存在
    codeMenuHasChildren                                 // 30014 菜单下有子节点,无法删除
    codeAppNotFound                                     // 30015 应用不存在
    codeSystemConfigNotFound                            // 30016 系统配置项不存在
    codeRoleInUse                                       // 30017 角色尚有员工绑定,无法删除
    codePasswordRuleInvalid                             // 30018 密码规则校验失败
    codeGroupTokenInvalid                               // 30019 换发组 token 时基础 token 无效/已过期/已吊销
)

var (
    UserNoGroup          = ke.New(codeUserNoGroup, "用户未关联任何租户", "")
    GroupNotFound        = ke.New(codeGroupNotFound, "租户不存在", "")
    GroupNameExists      = ke.New(codeGroupNameExists, "租户名称已存在", "")
    GroupCodeExists      = ke.New(codeGroupCodeExists, "租户编码已存在", "")
    RoleNotFound         = ke.New(codeRoleNotFound, "角色不存在", "")
    RoleNameExists       = ke.New(codeRoleNameExists, "角色名称在租户内已存在", "")
    RoleSystemCannotEdit = ke.New(codeRoleSystemCannotEdit, "系统内置角色不可编辑/删除", "")
    PermissionNotFound   = ke.New(codePermissionNotFound, "权限节点不存在", "")
    PermissionOutOfScope = ke.New(codePermissionOutOfScope, "权限越界(不在企业 scope 内)", "")
    GroupHasUsers        = ke.New(codeGroupHasUsers, "租户下仍有员工,无法删除", "")
    UserAlreadyInGroup   = ke.New(codeUserAlreadyInGroup, "员工已属于该租户", "")
    UserNotInGroup       = ke.New(codeUserNotInGroup, "员工不属于该租户", "")
    RoleNotInGroup       = ke.New(codeRoleNotInGroup, "角色不属于该租户", "")
    MenuNotFound         = ke.New(codeMenuNotFound, "菜单节点不存在", "")
    MenuHasChildren      = ke.New(codeMenuHasChildren, "菜单下有子节点,无法删除", "")
    AppNotFound          = ke.New(codeAppNotFound, "应用不存在", "")
    SystemConfigNotFound = ke.New(codeSystemConfigNotFound, "系统配置项不存在", "")
    RoleInUse            = ke.New(codeRoleInUse, "角色尚有员工绑定,无法删除", "")
    PasswordRuleInvalid  = ke.New(codePasswordRuleInvalid, "密码规则校验失败", "")
    GroupTokenInvalid    = ke.New(codeGroupTokenInvalid, "token 无效或已失效, 请重新登录", "")
)
