package errors

import "github.com/go-kratos/kratos/v2/errors"

const (
    codePresignedFail = iota + baseUploadErrCode
    codeUploadFileUrlInvalid
    codeUploadFileParsedFail
    codeUploadFileUploadFail
    codeFileNotExist
    codeFileUpdateFail
    codeFileDeleteFail
    codeUploadRecordCreateFail
    codeUploadRecordFindFail

    codeFileParamsInvalid
    codeUploadInitFail
    codePresignedFailByOriNameLengthErr
)

var (
    PreSignedFail          = errors.New(codePresignedFail, "pre-signed failed", "")
    FileUrlInvalid         = errors.New(codeUploadFileUrlInvalid, "file url is invalid", "")
    FileParsedFail         = errors.New(codeUploadFileParsedFail, "package parse fail", "")
    FileUploadFail         = errors.New(codeUploadFileUploadFail, "file upload fail", "")
    FileNotExist           = errors.New(codeFileNotExist, "file not exist", "")
    FileUpdateFail         = errors.New(codeFileUpdateFail, "file update fail", "")
    FileDeleteFail         = errors.New(codeFileDeleteFail, "file delete fail", "")
    UploadRecordCreateFail = errors.New(codeUploadRecordCreateFail, "upload record fail", "")
    UploadRecordFindFail   = errors.New(codeUploadRecordFindFail, "upload record fail", "")

    FileParamsInvalid       = errors.New(codeFileParamsInvalid, "file params is invalid", "")
    UploadInitFail          = errors.New(codeUploadInitFail, "upload init failed", "")
    PreSignedFailWithOriErr = errors.New(codePresignedFailByOriNameLengthErr, "pre-signed failed, oriName length error", "")
)
