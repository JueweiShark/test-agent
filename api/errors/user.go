package errors

import "github.com/go-kratos/kratos/v2/errors"

const (
    codeUserNotExist = iota + baseUserErrCode
    codeUserPWDNotMatch
    codeUserTokenGenerateFail
    codeUserRegisterFail
    codeUserLoginFail
    codeUserDeleteFail
    codeUserDuplicate
    codeUserPermissionMissFail
    codeUserUpdateFail
    codeUserFindFail
    codeUsePassWordSameOld //密码和旧密码一样
    codeUseIsForbidden //用户被禁用
    codeUserForbiddenFail//用户禁用失败
)
const (
    codeUserRoleCreateFail = iota + baseUserErrCode + 1000
    codeUserRoleUpdateFail
    codeUserRoleDeleteFail
    codeUserRoleFindFail
    codeUserRoleBindFail
    codeUserRoleDuplicate
    codeUserRoleInUsed
    codeUserRoleBindInfo
    codeUserRoleUnBindFail
)
const (
    codeUserGroupCreateFail = iota + baseUserErrCode + 2000
    codeUserGroupUpdateFail
    codeUserGroupDeleteFail
    codeUserGroupFindFail
    codeGroupNameDuplicate
    codeGroupAddMemberFail
    codeGroupDelMemberFail
    codeGroupMemberListFail
    codeGroupBindInfo
    codeGroupHasMember
)
const (
    codeUserPermissionFindFail = iota + baseUserErrCode + 3000
)

var (
    UserNotExist    = errors.New(codeUserNotExist, "账号或密码错误", "")
    UserPWDNotMatch = errors.New(codeUserPWDNotMatch, "账号或密码错误", "")
    UserTokenGenerateFail = errors.New(codeUserTokenGenerateFail, "token生成失败", "")
    UserRegisterFail       = errors.New(codeUserRegisterFail, "user register fail", "")
    UserLoginFail          = errors.New(codeUserLoginFail, "user login fail", "")
    UserDeleteFail         = errors.New(codeUserDeleteFail, "user delete fail", "")
    UserRoleCreateFail     = errors.New(codeUserRoleCreateFail, "role create fail", "")
    UserRoleUpdateFail     = errors.New(codeUserRoleUpdateFail, "role update fail", "")
    UserRoleDeleteFail     = errors.New(codeUserRoleDeleteFail, "role delete fail", "")
    UserRoleFindFail       = errors.New(codeUserRoleFindFail, "role find fail", "")
    UserRoleBindFail       = errors.New(codeUserRoleBindFail, "role bind fail", "")
    UserGroupCreateFail    = errors.New(codeUserGroupCreateFail, "group create fail", "")
    UserGroupUpdateFail    = errors.New(codeUserGroupUpdateFail, "group update fail", "")
    UserGroupDeleteFail    = errors.New(codeUserGroupDeleteFail, "group del fail", "")
    UserGroupFindFail      = errors.New(codeUserGroupFindFail, "group find fail", "")
    UserPermissionMissFail = errors.New(codeUserPermissionMissFail, "permission miss", "")
    GroupAddMemberFail     = errors.New(codeGroupAddMemberFail, "add member fail", "")
    GroupDelMemberFail     = errors.New(codeGroupDelMemberFail, "del member fail", "")
    GroupMemberListFail    = errors.New(codeGroupMemberListFail, "member list fail", "")
    UserPermissionFindFail = errors.New(codeUserPermissionFindFail, "user permission find fail", "")
    UserRoleDuplicate      = errors.New(codeUserRoleDuplicate, "role duplicate ", "")
    UserDuplicate          = errors.New(codeUserDuplicate, "用户名重复，请修改用户名", "")
    UserNotEmpty = errors.New(codeUserDuplicate, "用户名不能为空", "")
    GroupNameDuplicate     = errors.New(codeGroupNameDuplicate, "group name duplicate ", "")
    UserRoleInUsed         = errors.New(codeUserRoleInUsed, "role is used  ", "")
    UserRoleBindInfo       = errors.New(codeUserRoleBindInfo, "role bind info query fail ", "")
    GroupBindInfo          = errors.New(codeGroupBindInfo, "group bind info query fail ", "")
    GroupHasMember         = errors.New(codeGroupHasMember, "group has member info query fail ", "")
    UserUpdateFail         = errors.New(codeUserUpdateFail, "user update fail ", "")
    UserRoleUnBindFail     = errors.New(codeUserRoleUnBindFail, "ref user role unbind fail ", "")
    UserFindFail           = errors.New(codeUserFindFail, "user find fail", "")
    TokenNotMatch          = errors.New(codeUserFindFail, "token not match", "")
    UsePassWordSameOld     = errors.New(codeUsePassWordSameOld, "新密码与旧密码一样", "")
    UseIsForbidden         = errors.New(codeUseIsForbidden, "用户被禁用", "")
    UserForbiddenFail         = errors.New(codeUserForbiddenFail, "禁用失败", "")
)
