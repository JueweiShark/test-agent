package oss

import (
    "context"

    kr "github.com/go-kratos/kratos/v2/errors"
    "github.com/go-kratos/kratos/v2/log"

    "wuwen.com/wwuc/internal/conf"
)

type File struct {
    Name      string
    MinioPath string
    PathHash  int64
    CreatedBy int64
}
type OssConf struct {
    BasePath                 string
    Endpoint                 string
    DefaultBucket            string
    TempPath                 string
    TempPathAbs              string
    BackendResourcesBasePath string
    ReplaceFlag              bool
    InternalEndpoint         string
    InternalProtocol         string
    OssType                  string
    PublicKey                string
    PrivateKey               string
    FileHost                 string
    VerifyUploadMD5          bool
    PlatForm                 string
    Region                   string
}
type StorageService interface {
    InitClient(conf *conf.OSS, logger log.Logger)
    UploadFile(ctx context.Context, objectName, filePath, contentType string) (string, *kr.Error)
    PresignedGetUrl(ctx context.Context, objectName string, opt ...OSSOpt) (string, *kr.Error)
    PresignedGetUrlWithBucket(ctx context.Context, bucket, objectName string, opt ...OSSOpt) (string, *kr.Error)
    PresignedPutUrl(ctx context.Context, objectName string, opt ...OSSOpt) (string, *kr.Error)
    GetDeFaultBucket() string
    GetOssConf() *OssConf
    UploadFileList(ctx context.Context, localDir, targetPath string, filePath map[string]struct{}) (map[string]string, error)
    UploadBytes(ctx context.Context, objectName string, fileBytes []byte, opt ...OSSOpt) *kr.Error
    ListFiles(ctx context.Context, prefix string, opt ...OSSOpt) ([]string, *kr.Error)
    ListObjects(ctx context.Context, prefix string, opt ...OSSOpt) ([]*FileObject, *kr.Error)
    ReadFile(ctx context.Context, objectKey string, dataCh chan<- []byte, opt ...OSSOpt) ([]byte, *kr.Error)
    DownloadFile(ctx context.Context, objectName, filePath string) *kr.Error
    NewMultipartUpload(ctx context.Context, objectName string) (string, error)
    PresignedPutUrlPart(ctx context.Context, objectName, uploadID, partNumber string, oss ...OSSOpt) (string, *kr.Error)
    CheckFinished(ctx context.Context, objectName, uploadId string, partNumber int) (bool, string)
}

type opt struct {
    FileName     string
    NeedDownLoad bool
    NeedInternal bool
    BucketName   string
    Inline       bool
}

func WithFileName(f string) OSSOpt {
    return func(o *opt) {
        o.FileName = f
    }
}

func WithNeedDownLoad(b bool) OSSOpt {
    return func(o *opt) {
        o.NeedDownLoad = b
    }
}

// 使用
func WithNeedInternal(b bool) OSSOpt {
    return func(o *opt) {
        o.NeedInternal = b
    }
}

func WithBucketName(b string) OSSOpt {
    return func(o *opt) {
        o.BucketName = b
    }
}

func WithInline(b bool) OSSOpt {
    return func(o *opt) {
        o.Inline = b
    }
}

type OSSOpt func(o *opt)
