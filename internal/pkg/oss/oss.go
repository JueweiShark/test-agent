package oss

import (
    "context"
    "encoding/base64"
    "fmt"
    s3types "github.com/aws/aws-sdk-go/service/s3"
    kre "github.com/go-kratos/kratos/v2/errors"
    "github.com/go-kratos/kratos/v2/log"
    "github.com/google/wire"
    "github.com/ks3sdklib/aws-sdk-go/service/s3"
    "github.com/minio/minio-go/v7"
    "go.opentelemetry.io/otel/attribute"
    "os"
    "strings"
    "time"

    wae "wuwen.com/wwuc/api/errors"
    "wuwen.com/wwuc/internal/conf"
    "wuwen.com/wwuc/util/wuwen"
)

var ProviderSet = wire.NewSet(NewOssUsecase)

const (
    ResourcesTypeMetric = "metric"
)

type MultipartFile struct {
    UploadId            string                   `json:"uploadId"`
    BucketName          string                   `json:"bucketName"`
    ObjectName          string                   `json:"objectName"`
    ChunkSize           int64                    `json:"chunkSize"`
    FileInfo            os.FileInfo              `json:"fileInfo"`
    CompletedPartsMinio []minio.CompletePart     `json:"-"`
    CompletedPartsKs3   []*s3.CompletedPart      `json:"-"`
    CompletedPartsCos   []*s3types.CompletedPart `json:"-"`
}

type PresignedResult struct {
    FileName     string `json:"fileName"`
    AccessUrl    string `json:"accessUrl"`
    PreSignedUrl string `json:"preSignedUrl"`
    OssPath      string `json:"ossPath"`
}

type FileObject struct {
    Path string `json:"path"`
    Size int64  `json:"size"`
}
type Usecase struct {
    oss  StorageService
    conf *conf.Bootstrap
}

func NewOssUsecase(conf *conf.Bootstrap, logger log.Logger) *Usecase {
    var client StorageService
    switch conf.Oss.Use {
    case "minio":
        client = NewMinioClient(conf, logger)
    case "tos":
        client = NewTOSClient(conf, logger)
    case "ks3":
        client = NewKS3Client(conf, logger)
    case "cos":
        client = NewCosClient(conf, logger)
    }

    return &Usecase{
        oss:  client,
        conf: conf,
    }
}

type GetPreUrlParam struct {
    ResourcesType string
    FileNames     []string
    Download      bool
    GroupId       int64
}

func (uc *Usecase) GetPresignedUrls(ctx context.Context, param *GetPreUrlParam) (results []PresignedResult, kerr *kre.Error) {
    ctx, end := startOssSpan(ctx, "oss.GetPresignedUrls", attribute.String("oss.resources_type", param.ResourcesType))
    defer func() { end(kerr) }()
    log.Context(ctx).Infof("get presigned urls %#v", param)
    fileNames := param.FileNames
    // 如果oriName 大于0 则说明要替换原文件
    results = make([]PresignedResult, len(fileNames))
    basePath := fmt.Sprintf("upload_data/%s/%d/%s/%s", param.ResourcesType, param.GroupId, time.Now().Format(time.DateOnly), wuwen.UUID())
    for i, name := range fileNames {
        oName := fmt.Sprintf("%s/%s", basePath, name)
        accessUrl, err := uc.oss.PresignedGetUrl(ctx, oName)
        if err != nil && err.GetCode() != 0 {
            log.Context(ctx).Errorf("generate presigned access url err: %v", err)
            return nil, wae.PreSignedFail
        }
        putUrl, err := uc.oss.PresignedPutUrl(ctx, oName)
        if err != nil && err.GetCode() != 0 {
            log.Context(ctx).Errorf("generate presigned put url err: %v", err)
            return nil, wae.PreSignedFail
        }
        results[i] = PresignedResult{
            FileName:     name,
            AccessUrl:    accessUrl,
            PreSignedUrl: putUrl,
            OssPath:      oName,
        }
    }
    return results, wae.NilErr
}

func (uc *Usecase) ParseBase64Data(ctx context.Context, base64String string) ([]byte, string, *kre.Error) {
    // 去掉前缀 "data:image/png;base64," 如果有的话
    split := strings.Split(base64String, ";base64,")
    if len(split) != 2 {
        return nil, "", wae.Base64FormatInvalid
    }
    fileFormatSlice := strings.Split(split[0], "/")
    if len(fileFormatSlice) != 2 {
        return nil, "", wae.Base64FormatInvalid
    }
    format := fileFormatSlice[1]

    base64data := split[1]
    result, err := base64.StdEncoding.DecodeString(base64data)
    if err != nil {
        log.Context(ctx).Errorf("base64 to iamage data failed, %v", err)
        return nil, "", wae.Base64ConvertFail
    }
    return result, format, wae.NilErr
}

func (uc *Usecase) UploadFile(ctx context.Context, objectName, filePath, contentType string) (url string, kerr *kre.Error) {
    ctx, end := startOssSpan(ctx, "oss.UploadFile", attribute.String("oss.object", objectName))
    defer func() { end(kerr) }()
    return uc.oss.UploadFile(ctx, objectName, filePath, contentType)
}

func (uc *Usecase) DownloadFile(ctx context.Context, objectName, filePath string) (kerr *kre.Error) {
    ctx, end := startOssSpan(ctx, "oss.DownloadFile", attribute.String("oss.object", objectName))
    defer func() { end(kerr) }()
    return uc.oss.DownloadFile(ctx, objectName, filePath)
}

func (uc *Usecase) PresignedGetUrl(ctx context.Context, objectName string, ossOpt ...OSSOpt) (url string, kerr *kre.Error) {
    ctx, end := startOssSpan(ctx, "oss.PresignedGetUrl", attribute.String("oss.object", objectName))
    defer func() { end(kerr) }()
    return uc.oss.PresignedGetUrl(ctx, objectName, ossOpt...)
}
func (uc *Usecase) PresignedGetUrlWithBucket(ctx context.Context, bucket, objectName string, ossOpt ...OSSOpt) (url string, kerr *kre.Error) {
    ctx, end := startOssSpan(ctx, "oss.PresignedGetUrlWithBucket",
        attribute.String("oss.bucket", bucket), attribute.String("oss.object", objectName))
    defer func() { end(kerr) }()
    return uc.oss.PresignedGetUrlWithBucket(ctx, bucket, objectName, ossOpt...)
}

func (uc *Usecase) PresignedPutUrl(ctx context.Context, objectName string, ossOpt ...OSSOpt) (url string, kerr *kre.Error) {
    ctx, end := startOssSpan(ctx, "oss.PresignedPutUrl", attribute.String("oss.object", objectName))
    defer func() { end(kerr) }()
    return uc.oss.PresignedPutUrl(ctx, objectName, ossOpt...)
}

func (uc *Usecase) GetDeFaultBucket() string {
    return uc.oss.GetDeFaultBucket()
}

func (uc *Usecase) GetOssConf() *OssConf {
    return uc.oss.GetOssConf()
}

func (uc *Usecase) UploadFileList(ctx context.Context, localDir, targetPath string, filePath map[string]struct{}) (paths map[string]string, err error) {
    ctx, end := startOssSpan(ctx, "oss.UploadFileList")
    defer func() { end(err) }()
    return uc.oss.UploadFileList(ctx, localDir, targetPath, filePath)
}

func (uc *Usecase) UploadBytes(ctx context.Context, objecName string, bytes []byte) (kerr *kre.Error) {
    ctx, end := startOssSpan(ctx, "oss.UploadBytes", attribute.String("oss.object", objecName))
    defer func() { end(kerr) }()
    return uc.oss.UploadBytes(ctx, objecName, bytes)
}

func (uc *Usecase) ListFiles(ctx context.Context, prefix string) (files []string, kerr *kre.Error) {
    ctx, end := startOssSpan(ctx, "oss.ListFiles", attribute.String("oss.prefix", prefix))
    defer func() { end(kerr) }()
    return uc.oss.ListFiles(ctx, prefix)
}

func (uc *Usecase) ListObjects(ctx context.Context, prefix string) (objects []*FileObject, kerr *kre.Error) {
    ctx, end := startOssSpan(ctx, "oss.ListObjects", attribute.String("oss.prefix", prefix))
    defer func() { end(kerr) }()
    return uc.oss.ListObjects(ctx, prefix)
}

func (uc *Usecase) ReadFile(ctx context.Context, objectKey string, dataCh chan<- []byte, ossOpt ...OSSOpt) (data []byte, kerr *kre.Error) {
    ctx, end := startOssSpan(ctx, "oss.ReadFile", attribute.String("oss.object", objectKey))
    defer func() { end(kerr) }()
    return uc.oss.ReadFile(ctx, objectKey, dataCh, ossOpt...)
}

func (uc *Usecase) NewMultipartUpload(ctx context.Context, objectName string) (uploadId string, err error) {
    ctx, end := startOssSpan(ctx, "oss.NewMultipartUpload", attribute.String("oss.object", objectName))
    defer func() { end(err) }()
    return uc.oss.NewMultipartUpload(ctx, objectName)
}
func (uc *Usecase) PresignedPutUrlPart(ctx context.Context, objectName, uploadID, partNumber string, ossOpt ...OSSOpt) (url string, kerr *kre.Error) {
    ctx, end := startOssSpan(ctx, "oss.PresignedPutUrlPart",
        attribute.String("oss.object", objectName), attribute.String("oss.part_number", partNumber))
    defer func() { end(kerr) }()
    return uc.oss.PresignedPutUrlPart(ctx, objectName, uploadID, partNumber, ossOpt...)
}
func (uc *Usecase) CheckFinished(ctx context.Context, objectName, uploadId string, partNumber int) (ok bool, objectURL string) {
    // 无错误返回值，仅记录耗时；内部失败只打日志
    ctx, end := startOssSpan(ctx, "oss.CheckFinished", attribute.String("oss.object", objectName))
    defer end(nil)
    return uc.oss.CheckFinished(ctx, objectName, uploadId, partNumber)
}
