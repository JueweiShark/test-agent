package v2

import (
	ke "github.com/go-kratos/kratos/v2/errors"

	apiErr "wuwen.com/wwuc/api/errors"
)

// v2 接口的 proto 响应体没有 Code/Message 包裹字段,因此业务错误以 kratos error 形式返回。
// kratos 的 Error.Code 会被 HTTP ErrorEncoder 直接作为 HTTP 状态码写出,
// 而 api/errors/user_v2.go 中定义的业务码为 15000+ (非合法 HTTP 状态),
// 因此这里用 httpErr 将业务错误描述复用到正确的 HTTP 状态码上 (对应设计 5.1)。
func httpErr(status int, biz *ke.Error) *ke.Error {
	return ke.New(status, biz.Reason, biz.Message)
}

// 复用 api/errors/user_v2.go 中的描述文本, 按 design 5.1 映射到 HTTP 状态码。
var (
	errUserNotExist          = httpErr(401, apiErr.UserNotExist) // 复用 v1 的 UserNotExist 描述
	errUserPWDNotMatch       = httpErr(401, apiErr.UserPWDNotMatch)
	errUseIsForbidden        = httpErr(403, apiErr.UseIsForbidden)
	errUserNoGroup           = httpErr(403, apiErr.UserNoGroup)
	errUserTokenGenerateFail = httpErr(500, apiErr.UserTokenGenerateFail)
	// 数据隔离改造 (groupId 入 Token): 换发组 token 时基础 token 无效返回 401;
	// 绑定不存在返回 403 (与设计文档一致, 区别于 GetUserInfo 的 404 语义)
	errUserTokenInvalid       = httpErr(401, apiErr.GroupTokenInvalid)
	errUserNotInGroupForbidden = httpErr(403, apiErr.UserNotInGroup)

	errGroupNotFound        = httpErr(404, apiErr.GroupNotFound)
	errGroupNameExists      = httpErr(409, apiErr.GroupNameExists)
	errGroupCodeExists      = httpErr(409, apiErr.GroupCodeExists)
	errRoleNameExists       = httpErr(409, apiErr.RoleNameExists)
	errRoleSystemCannotEdit = httpErr(403, apiErr.RoleSystemCannotEdit)
	errRoleNotFound         = httpErr(404, apiErr.RoleNotFound)
	errPermissionNotFound   = httpErr(404, apiErr.PermissionNotFound)
	errPermissionOutOfScope = httpErr(403, apiErr.PermissionOutOfScope)

	errMenuNotFound         = httpErr(404, apiErr.MenuNotFound)
	errMenuHasChildren      = httpErr(409, apiErr.MenuHasChildren)
	errAppNotFound          = httpErr(404, apiErr.AppNotFound)
	errSystemConfigNotFound = httpErr(404, apiErr.SystemConfigNotFound)
	errRoleInUse            = httpErr(409, apiErr.RoleInUse)

	errUserAlreadyInGroup = httpErr(409, apiErr.UserAlreadyInGroup)
	errUserNotInGroup     = httpErr(404, apiErr.UserNotInGroup)
	errRoleNotInGroup     = httpErr(403, apiErr.RoleNotInGroup)

	errUserNotFound  = httpErr(404, apiErr.UserNotExist)
	errParamsInvalid = httpErr(422, apiErr.ParamsInvalid)
	errUserDuplicate = httpErr(409, apiErr.UserDuplicate)
	errValidateInvalid = httpErr(410, apiErr.UserNotEmpty)
	errPwdRuleInvalid = httpErr(422, apiErr.PasswordRuleInvalid)
	errInternalError = ke.New(500, "INTERNAL_ERROR", "内部错误")
)
