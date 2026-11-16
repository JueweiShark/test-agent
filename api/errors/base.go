package errors

import ke "github.com/go-kratos/kratos/v2/errors"

const (
    baseUserErrCode       = 10000
    baseUploadErrCode     = 20000
    basePermissionErrCode = 30000
    baseMinioErrCode      = 40000

    CodeParamsInvalid = 420
    CodePWDErr = 10001
    CodePWDSameErr = 10010
)

var ParamsInvalid = ke.New(CodeParamsInvalid, "params invalid", "")

var NilErr = ke.New(0, "", "")