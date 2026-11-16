package oss

import (
    "bytes"
    "context"
    "errors"
    "fmt"
    kr "github.com/go-kratos/kratos/v2/errors"
    "github.com/go-kratos/kratos/v2/log"
    "github.com/minio/minio-go/v7"
    "github.com/minio/minio-go/v7/pkg/credentials"
    "io"
    "net/url"
    "os"
    "path/filepath"
    "strings"
    "time"

    apiErrs "wuwen.com/wwuc/api/errors"
    "wuwen.com/wwuc/internal/conf"
    "wuwen.com/wwuc/util/wuwen"
)

const (
    MINIO            = "minio"
    CommonRootFolder = "common"
)

type minioClient struct {
    log    *log.Helper
    client *minio.Client
    conf   *OssConf
}

func (uc *minioClient) GetOssConf() *OssConf {
    return uc.conf
}

func NewMinioClient(conf *conf.Bootstrap, logger log.Logger) StorageService {
    BackendResourcesBasePath := fmt.Sprintf("%s/%s", conf.Oss.TempDir, "resource")
    TempPathAbs := fmt.Sprintf("%s/%s", BackendResourcesBasePath, conf.Oss.TempPath)
    m := &minioClient{
        conf: &OssConf{
            OssType:                  MINIO,
            Endpoint:                 conf.Oss.Endpoint,
            DefaultBucket:            conf.Oss.BucketName,
            TempPath:                 conf.Oss.TempPath,
            TempPathAbs:              TempPathAbs,
            BackendResourcesBasePath: BackendResourcesBasePath,
            InternalEndpoint:         conf.Oss.InternalEndpoint,
            InternalProtocol:         conf.Oss.InternalProtocol,
        },
    }
    m.InitClient(conf.Oss, logger)
    return m
}

func (uc *minioClient) InitClient(conf *conf.OSS, logger log.Logger) {
    c, err := minio.New(conf.Endpoint, &minio.Options{
        Creds:  credentials.NewStaticV4(conf.AccessKey, conf.SecretKey, ""),
        Secure: false, // Set to true if MinIO is configured with TLS
    })
    if err != nil {
        log.Error("minio client init error", err)
    }
    log.Info("minio service init", c)
    uc.client = c
}

func (uc *minioClient) parseOSSOpt(oss []OSSOpt) *opt {
    o := &opt{
        FileName:     "",
        NeedDownLoad: false,
        NeedInternal: false,
        BucketName:   uc.GetDeFaultBucket(),
    }
    for _, ossOpt := range oss {
        ossOpt(o)
    }
    return o
}

// replaceURL replaces parts of the URL as needed
func (uc *minioClient) replaceURL(ctx context.Context, originalURL string) string {
    endPoint := "http://localhost"
    //if reqCtx, ok := common.ReqCtxFromContext(ctx); ok {
    //    if len(reqCtx.EndPoint) > 0 && len(reqCtx.Protocol) > 0 {
    //        endPoint = fmt.Sprintf("%s://%s", reqCtx.Protocol, reqCtx.EndPoint)
    //    }
    //    if len(uc.conf.InternalEndpoint) > 0 {
    //        endPoint = fmt.Sprintf("%s://%s", reqCtx.Protocol, uc.conf.InternalEndpoint)
    //    }
    //}

    originalURL = strings.ReplaceAll(originalURL, fmt.Sprintf("%s://%s", "http", uc.conf.Endpoint),
        fmt.Sprintf("%s/%s", endPoint, MINIO))
    return originalURL
}

func (uc *minioClient) DownloadFile(ctx context.Context, objectName, filePath string) *kr.Error {
    bucket := uc.conf.DefaultBucket
    err := uc.client.FGetObject(ctx, bucket, objectName, filePath, minio.GetObjectOptions{})
    if err != nil {
        return kr.New(1, fmt.Sprintf("download file fail %v", err), "")
    }
    return nil
}

func (uc *minioClient) UploadFile(ctx context.Context, objectName, filePath, contentType string) (string, *kr.Error) {
    if uc.client == nil {
        return "", kr.New(1, "minio client is nil", "")
    }
    bucket := uc.conf.DefaultBucket
    if err := uc.CreateBucket(bucket); err != nil {
        log.Context(ctx).Errorf("CreateBucket fail %v", err)
    }

    file, err := os.Open(filePath)
    if err != nil {
        return "", kr.New(1, fmt.Sprintf("open upload file fail%v", err), "")
    }
    fileInfo, err := file.Stat()
    if err != nil {
        return "", kr.New(1, fmt.Sprintf("get upload file stat fail%v", err), "")
    }
    putObject, err := uc.client.PutObject(ctx, bucket, objectName, file, fileInfo.Size(), minio.PutObjectOptions{
        ContentType: contentType,
    })
    if err != nil {
        return "", kr.New(1, fmt.Sprintf("upload file fail%v", err), "")
    }

    log.Infof("file %s upload sucess,key %s size %d ", filePath, putObject.Key, putObject.Size)

    split := strings.Split(objectName, "/")
    fileName := split[len(split)-1]

    return uc.PresignedGetUrl(ctx, objectName, WithFileName(fileName), WithNeedInternal(true))
}

func (uc *minioClient) CreateBucket(bucket string) error {
    if uc.client == nil {
        return errors.New("minio client is nil")
    }
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    exists, err := uc.client.BucketExists(ctx, bucket)
    if err != nil {
        log.Error("check bucket is exists err :", err)
        return errors.New("check bucket is exists err")
    }
    if exists {
        return nil
    }
    if err := uc.client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
        log.Error(err)
        return errors.New("create bucket err")
    }
    return nil
}

func (uc *minioClient) PresignedPutUrl(ctx context.Context, objectName string, oss ...OSSOpt) (string, *kr.Error) {
    if uc.client == nil {
        return "", apiErrs.OssClientInvalid
    }
    o := uc.parseOSSOpt(oss)
    err := uc.CreateBucket(uc.conf.DefaultBucket)
    if err != nil {
        log.Context(ctx).Warnf("CreateBucket fail %v", err)
    }
    backgroundCtx, cancel := context.WithCancel(context.Background())
    defer cancel()
    u, err := uc.client.PresignedPutObject(backgroundCtx, uc.conf.DefaultBucket, objectName, time.Second*60*60*24)
    preUrl := u.String()
    if err != nil {
        return "", apiErrs.OssPresignedUrlFail.WithCause(err)
    }

    if o.NeedInternal {
        preUrl = uc.replaceURL(ctx, preUrl)
    }
    return preUrl, nil
}

func (uc *minioClient) PresignedGetUrl(ctx context.Context, objectName string, oss ...OSSOpt) (string, *kr.Error) {
    presignedURLOptions := url.Values{}
    o := uc.parseOSSOpt(oss)
    if o.NeedDownLoad {
        index := strings.LastIndex(objectName, "/")
        var fileName string
        if index > 0 && index+1 < len(objectName) {
            fileName = objectName[index+1:]
        } else {
            fileName = objectName
        }
        presignedURLOptions.Set("response-content-type", "application/octet-stream")
        presignedURLOptions.Set("response-content-disposition", fmt.Sprintf("attachment; filename=%s", fileName))
    }
    if len(o.FileName) > 0 {
        presignedURLOptions.Set("response-content-disposition", fmt.Sprintf("attachment; filename=%s", o.FileName))
    }
    if o.Inline {
        presignedURLOptions.Set("response-content-disposition", "inline")
        presignedURLOptions.Set("response-content-type", "text/plain")
    }
    presignedURL, err := uc.client.PresignedGetObject(ctx, uc.conf.DefaultBucket, objectName, time.Hour*24, presignedURLOptions)
    if err != nil {
        return "", apiErrs.OssPresignedUrlFail.WithCause(err)
    }
    //<<<<<<< HEAD
    var result string
    if o.NeedInternal {
        result = uc.replaceURL(ctx, presignedURL.String())
    } else {
        result = presignedURL.String()
    }
    return result, nil

    //=======
    //    return uc.replaceURL(ctx, presignedURL.String()), nil
    //>>>>>>> feature/v1.5.0
}

func (uc *minioClient) PresignedGetUrlWithBucket(ctx context.Context, bucket, objectName string, oss ...OSSOpt) (string, *kr.Error) {
    presignedURLOptions := url.Values{}
    o := uc.parseOSSOpt(oss)
    if o.NeedDownLoad {
        index := strings.LastIndex(objectName, "/")
        var fileName string
        if index > 0 && index+1 < len(objectName) {
            fileName = objectName[index+1:]
        } else {
            fileName = objectName
        }
        presignedURLOptions.Set("response-content-type", "application/octet-stream")
        presignedURLOptions.Set("response-content-disposition", fmt.Sprintf("attachment; filename=%s", fileName))
    }
    if len(o.FileName) > 0 {
        presignedURLOptions.Set("response-content-disposition", fmt.Sprintf("attachment; filename=%s", o.FileName))
    }
    if o.Inline {
        presignedURLOptions.Set("response-content-disposition", "inline")
        presignedURLOptions.Set("response-content-type", "text/plain")
    }
    presignedURL, err := uc.client.PresignedGetObject(ctx, bucket, objectName, time.Hour*24, presignedURLOptions)
    if err != nil {
        return "", apiErrs.OssPresignedUrlFail.WithCause(err)
    }
    //<<<<<<< HEAD
    var result string
    if o.NeedInternal {
        result = uc.replaceURL(ctx, presignedURL.String())
    } else {
        result = presignedURL.String()
    }
    return result, nil

    //=======
    //    return uc.replaceURL(ctx, presignedURL.String()), nil
    //>>>>>>> feature/v1.5.0
}

func (uc *minioClient) GetDeFaultBucket() string {
    return uc.conf.DefaultBucket
}

func (uc *minioClient) RenameFile(srcObjectName, destObjectName string) error {
    ctx, cancelFunc := context.WithCancel(context.Background())
    defer cancelFunc()
    _, err := uc.client.CopyObject(ctx, minio.CopyDestOptions{
        Bucket: uc.GetDeFaultBucket(),
        Object: destObjectName,
    }, minio.CopySrcOptions{
        Bucket: uc.GetDeFaultBucket(),
        Object: srcObjectName,
    })
    if err != nil {
        uc.log.WithContext(ctx).Errorf("rename file %s fail %v", srcObjectName, err)
        return err
    }
    return nil
}

// UploadFileList 将本地的文件夹下的文件上传到目标的制定文件夹下
func (uc *minioClient) UploadFileList(ctx context.Context, localDir, targetPath string, filePath map[string]struct{}) (map[string]string, error) {
    input := make(chan minio.SnowballObject, 1)
    opts := minio.SnowballOptions{
        Opts: minio.PutObjectOptions{},
        // Keep in memory. We use this since we have small total payload.
        InMemory: true,
        // Compress data when uploading to a MinIO host.
        Compress: true,
        // skip error
        SkipErrs: true,
    }
    remotePathMap := make(map[string]string)
    uuid := wuwen.UUID()
    go func() {
        defer close(input)

        for filePath := range filePath {
            localPath := filepath.Join(localDir, filePath)
            key := fmt.Sprintf("%s/%s/%s", targetPath, uuid, filePath)
            remotePathMap[filePath] = key
            content, err := os.ReadFile(localPath)
            if err != nil {
                log.Context(ctx).Error("creat upload snow ball err.. ", err)
            }
            input <- minio.SnowballObject{
                // Create path to store objects within the bucket.
                Key:     key,
                Size:    int64(len(content)),
                ModTime: time.Now(),
                Content: bytes.NewBuffer(content),
                Close: func() {
                    log.Infof("file upload end %s", key)
                },
            }
        }
    }()
    background, cancel := context.WithCancel(context.Background())
    defer cancel()
    if err := uc.client.PutObjectsSnowball(background, uc.conf.DefaultBucket, opts, input); err != nil {
        log.Context(ctx).Error(err)
        return nil, err
    }
    return remotePathMap, nil
}

func (uc *minioClient) UploadBytes(ctx context.Context, objectName string, fileBytes []byte, opt ...OSSOpt) *kr.Error {
    o := uc.parseOSSOpt(opt)
    reader := bytes.NewReader(fileBytes)
    _, err := uc.client.PutObject(ctx, o.BucketName, objectName, reader, int64(len(fileBytes)), minio.PutObjectOptions{})
    if err != nil {
        log.Context(ctx).Errorf("upload memery file error, %s", err.Error())
        return apiErrs.FileUploadFail
    }
    return nil
}

func (uc *minioClient) ListFiles(ctx context.Context, prefix string, opt ...OSSOpt) ([]string, *kr.Error) {
    o := uc.parseOSSOpt(opt)
    var objects []string
    resultCh := uc.client.ListObjects(ctx, o.BucketName, minio.ListObjectsOptions{
        Prefix:    fmt.Sprintf("%s/", prefix),
        Recursive: true,
    })

    for obj := range resultCh {
        if obj.Err != nil {
            return nil, apiErrs.OssListFail
        }
        objects = append(objects, obj.Key)
    }
    return objects, nil
}

func (uc *minioClient) ListObjects(ctx context.Context, prefix string, opt ...OSSOpt) ([]*FileObject, *kr.Error) {
    o := uc.parseOSSOpt(opt)
    var objects []*FileObject
    resultCh := uc.client.ListObjects(ctx, o.BucketName, minio.ListObjectsOptions{
        Prefix:    fmt.Sprintf("%s/", prefix),
        Recursive: true,
    })

    for obj := range resultCh {
        if obj.Err != nil {
            return nil, apiErrs.OssListFail
        }
        objects = append(objects, &FileObject{
            Path: obj.Key,
            Size: obj.Size,
        })
    }
    return objects, nil
}

func (uc *minioClient) ReadFile(ctx context.Context, objectKey string, dataCh chan<- []byte, opt ...OSSOpt) ([]byte, *kr.Error) {
    o := uc.parseOSSOpt(opt)
    object, err := uc.client.GetObject(ctx, o.BucketName, objectKey, minio.GetObjectOptions{})
    if err != nil {
        log.Context(ctx).Errorf("read file %s fail %v", objectKey, err)
        return nil, apiErrs.FileNotExist
    }
    defer object.Close()

    var result []byte
    buf := make([]byte, 2*1024) // 每次读取 2 字节

    for {
        n, err := object.Read(buf)
        if n > 0 {
            chunk := buf[:n]
            if dataCh != nil {
                dataCh <- chunk
            } else {
                result = append(result, chunk...)
            }
        }
        if err == io.EOF {
            break
        }
        if err != nil {
            log.Context(ctx).Errorf("read file fail :%+v", err)
            return nil, apiErrs.OssFileReadFail
        }
    }

    if dataCh != nil {
        close(dataCh)
        return nil, nil
    } else {
        return result, nil
    }
}

// ========= 分片上传 =========
func (s *minioClient) NewMultipartUpload(ctx context.Context, objectName string) (string, error) {
    return "", nil
}

// PresignedPutUrlPart 生成分片上传的预签名 URL（返回格式: "<url>|||<Authorization>")
func (s *minioClient) PresignedPutUrlPart(ctx context.Context, objectName, uploadID, partNumber string, oss ...OSSOpt) (string, *kr.Error) {
    return "", nil
}

func (s *minioClient) CheckFinished(ctx context.Context, objectName, uploadId string, partNumber int) (bool, string) {
    return true, objectName
}

// FinishMultipartUpload 完成分片上传。
func (s *minioClient) FinishMultipartUpload(objectName, uploadID string, etags []string, mimeType string) error {
    return nil
}
