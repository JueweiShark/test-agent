package errors

import "github.com/go-kratos/kratos/v2/errors"

const (
    codeOssClientInvalid = iota + baseMinioErrCode
    codeOssPresignedUrlFail
    codeOssListFail
    codeOssFileReadFail
    codeBase64ConvertFail
    codeBase64FormatInvalid
    codeOssPutObjectFail
)

var (
    OssClientInvalid    = errors.New(codeOssClientInvalid, "oss client is nil", "")
    OssPresignedUrlFail = errors.New(codeOssPresignedUrlFail, "Oss pre-signed get url fail", "")
    OssListFail         = errors.New(codeOssListFail, "oss list file fail", "")
    OssFileReadFail     = errors.New(codeOssFileReadFail, "oss file read fail", "")
    OssPutObjectFail    = errors.New(codeOssPutObjectFail, "oss file upload fail", "")
    Base64ConvertFail   = errors.New(codeBase64ConvertFail, "base64ImageData convert fail", "")
    Base64FormatInvalid = errors.New(codeBase64ConvertFail, "base64 format invalid", "")
)
