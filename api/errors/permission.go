package errors

import "github.com/go-kratos/kratos/v2/errors"

const (
    codePermissionCreateFail = iota + basePermissionErrCode
    codePermissionUpdateFail
    codePermissionDeleteFail
    codePermissionFindFail
    codePermissionNameDuplicate
)

const (
    codePermissionSetCreateFail = iota + basePermissionErrCode + 1000
    codePermissionSetUpdateFail
    codePermissionSetDeleteFail
    codePermissionSetFindFail
    codePermissionSetNameDuplicate
    codePermissionSetMoreGroup
)

const (
    codePermissionTemplateCreateFail = iota + basePermissionErrCode + 2000
    codePermissionTemplateNameDuplicate
    codePermissionTemplateUpdateFail
    codePermissionTemplateDeleteFail
)

var (
    PermissionCreateFail            = errors.New(codePermissionCreateFail, "permission create fail ", "")
    PermissionUpdateFail            = errors.New(codePermissionUpdateFail, "permission update fail ", "")
    PermissionDeleteFail            = errors.New(codePermissionDeleteFail, "permission delete fail ", "")
    PermissionFindFail              = errors.New(codePermissionFindFail, "permission find fail ", "")
    PermissionNameDuplicate         = errors.New(codePermissionNameDuplicate, "permission name duplicate ", "")
    PermissionSetCreateFail         = errors.New(codePermissionSetCreateFail, "permission set create fail ", "")
    PermissionSetUpdateFail         = errors.New(codePermissionSetUpdateFail, "permission set update fail ", "")
    PermissionSetDeleteFail         = errors.New(codePermissionSetDeleteFail, "permission set delete fail ", "")
    PermissionSetFindFail           = errors.New(codePermissionSetFindFail, "permission set find fail ", "")
    PermissionSetNameDuplicate      = errors.New(codePermissionSetNameDuplicate, "permission set name duplicate ", "")
    PermissionSetMoreGroup          = errors.New(codePermissionSetMoreGroup, "permission set contains more group ", "")
    PermissionTemplateCreateFail    = errors.New(codePermissionTemplateCreateFail, "permission template create fail ", "")
    PermissionTemplateNameDuplicate = errors.New(codePermissionTemplateNameDuplicate, "permission template name or code duplicate ", "")
    PermissionTemplateUpdateFail    = errors.New(codePermissionTemplateUpdateFail, "permission template update fail ", "")
    PermissionTemplateDeleteFail    = errors.New(codePermissionTemplateDeleteFail, "permission template delete fail ", "")
)
