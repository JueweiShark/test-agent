package oss

import (
    "bytes"
    "context"
    "errors"
    "fmt"
    "io"
    "os"
    "path/filepath"
    "sort"
    "strconv"
    "strings"
    "sync"
    "time"

    ke "github.com/go-kratos/kratos/v2/errors"
    "github.com/go-kratos/kratos/v2/log"
    "github.com/ks3sdklib/aws-sdk-go/aws"
    "github.com/ks3sdklib/aws-sdk-go/aws/credentials"
    "github.com/ks3sdklib/aws-sdk-go/service/s3"

    apiErr "wuwen.com/wwuc/api/errors"
    "wuwen.com/wwuc/internal/conf"
    "wuwen.com/wwuc/util/wuwen"
)

const (
    KS3 = "ks3"
)

type kS3Client struct {
    client         *s3.S3
    clientInternal *s3.S3
    log            *log.Helper
    conf           *OssConf
    // 已确认存在的 bucket 缓存，避免每次上传前都同步 HeadBucket 一次内网请求
    bucketExists sync.Map
}

func NewKS3Client(conf *conf.Bootstrap, logger log.Logger) StorageService {
    BackendResourcesBasePath := fmt.Sprintf("%s/%s", conf.Oss.TempDir, "resource")
    TempPathAbs := fmt.Sprintf("%s/%s", BackendResourcesBasePath, conf.Oss.TempPath)
    k3s := &kS3Client{
        conf: &OssConf{
            OssType:                  KS3,
            BasePath:                 conf.Oss.BasePath,
            Endpoint:                 conf.Oss.Endpoint,
            DefaultBucket:            conf.Oss.BucketName,
            TempPath:                 conf.Oss.TempPath,
            TempPathAbs:              TempPathAbs,
            BackendResourcesBasePath: BackendResourcesBasePath,
            ReplaceFlag:              false,
            InternalEndpoint:         conf.Oss.InternalEndpoint,
            InternalProtocol:         conf.Oss.InternalProtocol,
            Region:                   conf.Oss.Region,
        },
    }
    log.Infof("ossConf %+v", k3s.conf)
    k3s.InitClient(conf.Oss, logger)
    return k3s
}

func (ks3 *kS3Client) InitClient(conf *conf.OSS, logger log.Logger) {
    cre := credentials.NewStaticCredentials(conf.AccessKey, conf.SecretKey, "")
    ks3.client = s3.New(&aws.Config{
        Region:           conf.Region,
        Credentials:      cre,
        Endpoint:         conf.Endpoint,
        DisableSSL:       true,      //是否禁用https
        LogLevel:         aws.Error, //是否开启日志,0为关闭日志，1为开启日志
        LogHTTPBody:      false,     //是否把HTTP请求body打入日志
        S3ForcePathStyle: false,     // 绑定域名这个地方必须是false
        Logger:           nil,       //打日志的位置
        DomainMode:       false,     //是否开启自定义bucket绑定域名，当开启时 S3ForcePathStyle 参数不生效。
        SignerVersion:    "V4",
        MaxRetries:       1,
    })
    // 配置CORS规则
    maxAgeSeconds := int64(100)
    corsConfiguration := &s3.CORSConfiguration{
        Rules: []*s3.CORSRule{
            {
                AllowedHeaders: []string{
                    "*",
                },
                AllowedMethods: []string{
                    "PUT",
                    "GET",
                },
                AllowedOrigins: []string{
                    "*",
                },
                MaxAgeSeconds: &maxAgeSeconds,
            },
        },
    }
    // 设置桶的CORS配置
    _, err := ks3.client.PutBucketCORS(&s3.PutBucketCORSInput{
        Bucket:            aws.String(conf.BucketName),
        CORSConfiguration: corsConfiguration,
    })
    if err != nil {
        log.Errorf("ks3 InitClient err %v", err)
    }
    ks3.clientInternal = s3.New(&aws.Config{
        Region:           "BEIJING",
        Credentials:      cre,
        Endpoint:         ks3.conf.InternalEndpoint,
        DisableSSL:       true,  //是否禁用https
        LogLevel:         0,     //是否开启日志,0为关闭日志，1为开启日志
        LogHTTPBody:      false, //是否把HTTP请求body打入日志
        S3ForcePathStyle: false,
        Logger:           nil,   //打日志的位置
        DomainMode:       false, //是否开启自定义bucket绑定域名，当开启时 S3ForcePathStyle 参数不生效。
        SignerVersion:    "V4",
        MaxRetries:       1,
    })
    log.Info("ks3 InitClient success")
}
func (ks3 *kS3Client) parseOSSOpt(oss []OSSOpt) *opt {
    var o = &opt{
        FileName:     "",
        NeedDownLoad: false,
        NeedInternal: false,
        BucketName:   ks3.GetDeFaultBucket(),
    }
    for _, ossOpt := range oss {
        ossOpt(o)
    }
    return o
}
func (ks3 *kS3Client) UploadFile(ctx context.Context, objectName, filePath, contentType string) (string, *ke.Error) {
    if ks3.client == nil {
        return "", apiErr.OssPutObjectFail
    }
    var fileName string
    if err := ks3.CreateBucket(ctx, ks3.conf.DefaultBucket); err != nil {
        log.Context(ctx).Errorf("CreateBucket fail %v", err)
    }
    title := "ks3Client UploadFile Retry"
    retryOpt := wuwen.RetryOption{
        Mode:             wuwen.NormalRetry,
        RetryInterval:    1 * time.Second,
        RetryTimes:       3,
        PrintErrInterval: 10 * time.Second,
        RetryErrs:        []error{},
    }
    err := wuwen.RetryDo(title, retryOpt, func(info wuwen.RetryInfo) (e error) {
        file, e := os.Open(filePath)
        if e != nil {
            log.Context(ctx).Errorf("open file fail %v", e)
            return errors.New(fmt.Sprintf("open upload file fail%v", e))
        }
        _, e = ks3.client.PutObject(&s3.PutObjectInput{
            Bucket:      aws.String(ks3.conf.DefaultBucket),     // bucket名称
            Key:         aws.String(objectName),                 // object key
            ACL:         aws.String("private"),                  //权限，支持private(私有)，public-read(公开读)
            Body:        file,                                   //要上传的内容
            ContentType: aws.String("application/octet-stream"), //设置content-type
            Metadata:    map[string]*string{},
        })
        if e != nil {
            return errors.New(fmt.Sprintf("upload file fail%v", e))
        }
        split := strings.Split(objectName, "/")
        fileName = split[len(split)-1]
        return
    })
    if err != nil {
        log.Context(ctx).Errorf("ks3 upload file fail %v", err)
        return "", apiErr.OssPutObjectFail
    }

    log.Infof("file %s upload sucess,key %s ", filePath, objectName)

    return ks3.PresignedGetUrl(ctx, objectName, WithFileName(fileName), WithNeedInternal(true))
}

func (ks3 *kS3Client) PresignedGetUrl(ctx context.Context, objectName string, opt ...OSSOpt) (string, *ke.Error) {
    var urlConf s3.GeneratePresignedUrlInput
    if ks3.client == nil {
        return "", apiErr.OssClientInvalid
    }
    // 预签名是本地签名，无网络请求，读路径不需要建桶（对象存在则桶必存在）
    o := ks3.parseOSSOpt(opt)

    urlConf.Bucket = aws.String(ks3.conf.DefaultBucket)
    urlConf.Key = aws.String(objectName)
    urlConf.Expires = 3600 * 24 * 7
    urlConf.HTTPMethod = s3.GET

    if o.FileName != "" {
        // 生成带有请求参数的签名URL
        urlConf.ResponseContentDisposition = aws.String(fmt.Sprintf("attachment; filename=%s", o.FileName))
    }
    ks3Client := ks3.client
    if o.NeedInternal {
        ks3Client = ks3.clientInternal
    }
    // 生成带有content type URL
    urlConf.ResponseContentType = aws.String("application/octet-stream")
    getUrl, err1 := ks3Client.GeneratePresignedUrl(&urlConf)
    if err1 != nil {
        log.Context(ctx).Error("Error generating signed URL:", err1)
        return "", apiErr.OssPresignedUrlFail
    }
    return getUrl, apiErr.NilErr
}
func (ks3 *kS3Client) PresignedGetUrlWithBucket(ctx context.Context, bucket, objectName string, opt ...OSSOpt) (string, *ke.Error) {
    var urlConf s3.GeneratePresignedUrlInput
    if ks3.client == nil {
        return "", apiErr.OssClientInvalid
    }
    // 预签名是本地签名，无网络请求，读路径不需要建桶
    o := ks3.parseOSSOpt(opt)

    urlConf.Bucket = aws.String(bucket)
    urlConf.Key = aws.String(objectName)
    urlConf.Expires = 3600 * 24 * 7
    urlConf.HTTPMethod = s3.GET

    if o.FileName != "" {
        // 生成带有请求参数的签名URL
        urlConf.ResponseContentDisposition = aws.String(fmt.Sprintf("attachment; filename=%s", o.FileName))
    }

    // 生成带有content type URL
    urlConf.ResponseContentType = aws.String("application/octet-stream")
    ks3Client := ks3.client
    if o.NeedInternal {
        ks3Client = ks3.clientInternal
    }
    putUrl, err1 := ks3Client.GeneratePresignedUrl(&urlConf)
    if err1 != nil {
        log.Context(ctx).Error("Error generating signed URL:", err1)
        return "", apiErr.OssPresignedUrlFail
    }
    return putUrl, apiErr.NilErr
}

func (ks3 *kS3Client) PresignedPutUrl(ctx context.Context, objectName string, opt ...OSSOpt) (string, *ke.Error) {
    if ks3.client == nil {
        return "", apiErr.OssClientInvalid
    }
    o := ks3.parseOSSOpt(opt)
    err := ks3.CreateBucket(ctx, o.BucketName)
    if err != nil {
        log.Context(ctx).Warnf("CreateBucket fail %v", err)
    }
    ks3Client := ks3.client
    if o.NeedInternal {
        ks3Client = ks3.clientInternal
    }
    preUrl, err := ks3Client.GeneratePresignedUrl(&s3.GeneratePresignedUrlInput{
        Bucket:      aws.String(ks3.conf.DefaultBucket),     // 设置 bucket 名称
        Key:         aws.String(objectName),                 // 设置 object key
        ContentType: aws.String("application/octet-stream"), //如果是PUT方法，需要设置content-type
        Expires:     86400,
        HTTPMethod:  s3.PUT,
    })
    if err != nil {
        return "", apiErr.OssPresignedUrlFail
    }
    return preUrl, apiErr.NilErr
}

func (ks3 *kS3Client) UploadFileList(ctx context.Context, localDir, targetPath string, filePathMap map[string]struct{}) (map[string]string, error) {
    if ks3.client == nil {
        log.Context(ctx).Errorf("ks3 client is nil")
        return nil, apiErr.OssClientInvalid
    }
    var files []*os.File
    defer func() {
        for _, file := range files {
            _ = file.Close()
        }
    }()
    if err := ks3.CreateBucket(ctx, ks3.GetDeFaultBucket()); err != nil {
        log.Context(ctx).Errorf("CreateBucket fail %v", err)
    }

    title := "ks3Client UploadFileList Retry"
    retryOpt := wuwen.RetryOption{
        Mode:             wuwen.NormalRetry,
        RetryInterval:    1 * time.Second,
        RetryTimes:       3,
        PrintErrInterval: 10 * time.Second,
        RetryErrs:        []error{},
    }
    remotePathMap := make(map[string]string)
    for filePath := range filePathMap {
        retryOpt.RetryErrs = retryOpt.RetryErrs[:0]
        err := wuwen.RetryDo(title, retryOpt, func(info wuwen.RetryInfo) (e error) {
            uuid := wuwen.UUID()
            key := fmt.Sprintf("%s/%s/%s", targetPath, uuid, filePath)
            file, e := os.Open(filepath.Join(localDir, filePath))
            remotePathMap[filePath] = key
            if e != nil {
                log.Context(ctx).Errorf("open file fail %v", e)
                return errors.New("open file fail")
            }
            files = append(files, file)
            _, e = ks3.clientInternal.PutObject(&s3.PutObjectInput{
                Bucket:   aws.String(ks3.GetDeFaultBucket()),
                Key:      aws.String(key),
                Body:     file,
                Metadata: map[string]*string{},
            })
            if e != nil {
                log.Context(ctx).Error(fmt.Sprintf("upload file fail %v", e))
                return e
            }
            return
        })
        if err != nil {
            return nil, errors.New(fmt.Sprintf("kS3Client.UploadFileList error. filePath=%v", filePath))
        }
    }
    return remotePathMap, nil
}

func (ks3 *kS3Client) UploadBytes(ctx context.Context, objectName string, fileBytes []byte, opt ...OSSOpt) *ke.Error {
    o := ks3.parseOSSOpt(opt)
    err := ks3.CreateBucket(ctx, o.BucketName)
    if err != nil {
        log.Context(ctx).Warnf("CreateBucket fail %v", err)
    }

    ks3Client := ks3.client
    if o.NeedInternal {
        ks3Client = ks3.clientInternal
    }
    _, err = ks3Client.PutObject(&s3.PutObjectInput{
        Bucket:      aws.String(ks3.conf.DefaultBucket),     // bucket名称
        Key:         aws.String(objectName),                 // object key
        ACL:         aws.String("public-read"),              //权限，支持private(私有)，public-read(公开读)
        Body:        bytes.NewReader(fileBytes),             //要上传的内容
        ContentType: aws.String("application/octet-stream"), //设置content-type
    })
    if err != nil {
        log.Context(ctx).Errorf("upload memery file error, %s", err.Error())
        return apiErr.FileUploadFail
    }
    return apiErr.NilErr
}

func (ks3 *kS3Client) ListFiles(ctx context.Context, prefix string, opt ...OSSOpt) ([]string, *ke.Error) {
    log.Context(ctx).Infof("list files pre fix:%s", prefix)
    o := ks3.parseOSSOpt(opt)
    var objects []string
    var marker string
    ks3Client := ks3.client
    if o.NeedInternal {
        ks3Client = ks3.clientInternal
    }
    for {
        ListObjectsOutput, err := ks3Client.ListObjects(&s3.ListObjectsInput{
            Bucket: aws.String(o.BucketName),
            Marker: aws.String(marker),
            Prefix: aws.String(prefix),
        })
        if err != nil {
            log.Context(ctx).Errorf("list file error, %s", err.Error())
            return nil, apiErr.OssListFail
        }
        for _, content := range ListObjectsOutput.Contents {
            objects = append(objects, *content.Key)
        }
        if ListObjectsOutput.IsTruncated == nil || !*ListObjectsOutput.IsTruncated {
            break
        }
        marker = *ListObjectsOutput.NextMarker
    }
    return objects, apiErr.NilErr
}

func (ks3 *kS3Client) ListObjects(ctx context.Context, prefix string, opt ...OSSOpt) ([]*FileObject, *ke.Error) {
    log.Context(ctx).Infof("list files pre fix:%s", prefix)
    o := ks3.parseOSSOpt(opt)
    var objects []*FileObject
    var marker string
    ks3Client := ks3.client
    if o.NeedInternal {
        ks3Client = ks3.clientInternal
    }
    for {
        ListObjectsOutput, err := ks3Client.ListObjects(&s3.ListObjectsInput{
            Bucket: aws.String(o.BucketName),
            Marker: aws.String(marker),
            Prefix: aws.String(prefix),
        })
        if err != nil {
            log.Context(ctx).Errorf("list file error, %s", err.Error())
            return nil, apiErr.OssListFail
        }
        for _, content := range ListObjectsOutput.Contents {
            objects = append(objects, &FileObject{
                Path: *content.Key,
                Size: *content.Size,
            })
        }
        if ListObjectsOutput.IsTruncated == nil || !*ListObjectsOutput.IsTruncated {
            break
        }
        marker = *ListObjectsOutput.NextMarker
    }
    return objects, apiErr.NilErr
}
func (ks3 *kS3Client) ReadFile(ctx context.Context, objectKey string, dataCh chan<- []byte, opt ...OSSOpt) ([]byte, *ke.Error) {
    // 读路径不需要建桶（对象存在则桶必存在）
    o := ks3.parseOSSOpt(opt)
    ks3Client := ks3.client
    if o.NeedInternal {
        ks3Client = ks3.clientInternal
    }
    object, err := ks3Client.GetObject(&s3.GetObjectInput{
        Bucket: aws.String(o.BucketName),
        Key:    aws.String(objectKey),
    })
    if err != nil || object == nil {
        log.Context(ctx).Errorf("get object file failed, %s", err.Error())
        return nil, apiErr.FileNotExist
    }

    var result []byte
    buf := make([]byte, 2*1024) // 每次读取 2 字节

    for {
        n, err := object.Body.Read(buf)
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
            return nil, apiErr.OssFileReadFail
        }
    }

    if dataCh != nil {
        close(dataCh)
        return nil, apiErr.NilErr
    } else {
        return result, apiErr.NilErr
    }
}

func (ks3 *kS3Client) DownloadFile(ctx context.Context, objectName, filePath string) *ke.Error {
    // 读路径不需要建桶（对象存在则桶必存在）
    err := ks3.clientInternal.GetObjectToFile(&s3.GetObjectInput{
        Bucket: aws.String(ks3.conf.DefaultBucket),
        Key:    aws.String(objectName),
    }, filePath)
    if err != nil {
        log.Context(ctx).Errorf("get object file failed, %s", err.Error())
        return apiErr.FileNotExist
    }
    return apiErr.NilErr
}

func (ks3 *kS3Client) GetDeFaultBucket() string {
    return ks3.conf.DefaultBucket
}

func (ks3 *kS3Client) CreateBucket(ctx context.Context, bucket string) error {
    if ks3.clientInternal == nil {
        return errors.New("ks3 client is nil")
    }
    // 命中缓存直接返回，省掉每次 HeadBucket 的内网请求
    if _, ok := ks3.bucketExists.Load(bucket); ok {
        return nil
    }
    exists, err := ks3.clientInternal.HeadBucketExist(bucket)
    if err != nil {
        log.Context(ctx).Error("check bucket is exists err :", err)
        return errors.New("check bucket is exists err")
    }
    if exists {
        ks3.bucketExists.Store(bucket, struct{}{})
        return nil
    }
    _, err = ks3.clientInternal.CreateBucket(&s3.CreateBucketInput{
        Bucket:     aws.String(bucket),
        BucketType: aws.String(s3.BucketTypeIA),
    })
    if err != nil {
        log.Context(ctx).Error(err)
        return errors.New("create bucket err")
    }
    ks3.bucketExists.Store(bucket, struct{}{})
    return nil
}

func (ks3 *kS3Client) RenameFile(ctx context.Context, bucket, srcObjectName, destObjectName string) error {
    _, err := ks3.clientInternal.CopyObject(&s3.CopyObjectInput{
        Bucket:     aws.String(bucket),
        CopySource: aws.String(fmt.Sprintf("%v/%v", bucket, srcObjectName)),
        Key:        aws.String(destObjectName),
    })
    if err != nil {
        log.Context(ctx).Errorf("rename file %s fail %v", srcObjectName, err)
        return err
    }
    return nil
}

func (ks3 *kS3Client) GetOssConf() *OssConf {
    return ks3.conf
}

// ========= 分片上传 =========
func (ks3 *kS3Client) NewMultipartUpload(ctx context.Context, objectName string) (string, error) {
    bucketName := ks3.conf.DefaultBucket
    input := &s3.CreateMultipartUploadInput{
        Bucket:      aws.String(bucketName),
        Key:         aws.String(objectName),
        ContentType: aws.String("application/octet-stream"),
    }
    ret, err := ks3.client.CreateMultipartUpload(input)
    if err != nil {
        return "", err
    }
    uploadId := *ret.UploadID
    return uploadId, nil
}

func (ks3 *kS3Client) PresignedPutUrlPart(ctx context.Context, objectName, uploadID, partNumber string, oss ...OSSOpt) (string, *ke.Error) {
    if ks3.client == nil {
        return "", apiErr.PreSignedFail
    }
    err := ks3.CreateBucket(ctx, ks3.conf.DefaultBucket)
    if err != nil {
        log.Context(ctx).Warnf("CreateBucket fail %v", err)
    }
    number, _ := strconv.Atoi(partNumber)
    req, _ := ks3.client.UploadPartRequest(&s3.UploadPartInput{
        Bucket:     aws.String(ks3.conf.DefaultBucket),
        Key:        aws.String(objectName),
        UploadID:   aws.String(uploadID),
        PartNumber: aws.Long(int64(number)),
    })
    req.ExpireTime = 86400
    req.Sign()
    return req.HTTPRequest.URL.String(), apiErr.NilErr
}

// complete 合并分片
func (ks3 *kS3Client) complete(ctx context.Context, bucketName string, cpFile *MultipartFile) error {
    sort.Slice(cpFile.CompletedPartsMinio, func(i, j int) bool {
        return cpFile.CompletedPartsMinio[i].PartNumber < cpFile.CompletedPartsMinio[j].PartNumber
    })

    _, err := ks3.client.CompleteMultipartUpload(&s3.CompleteMultipartUploadInput{
        Bucket:   aws.String(bucketName),
        Key:      aws.String(cpFile.ObjectName),
        UploadID: aws.String(cpFile.UploadId),
        MultipartUpload: &s3.CompletedMultipartUpload{
            Parts: cpFile.CompletedPartsKs3,
        },
    })
    if err != nil {
        log.Context(ctx).Errorf("CompleteMultipartUpload err: %s", err)
        return err
    }
    return nil
}

// 列出已上传的分片
func (ks3 *kS3Client) listObjectParts(ctx context.Context, bucketName, objectName, uploadID string) (map[int64]s3.Part, error) {
    // Part number marker for the next batch of request.
    var nextPartNumberMarker int64
    var partsInfo = make(map[int64]s3.Part)
    for {
        listObjPartsResult, err := ks3.clientInternal.ListParts(&s3.ListPartsInput{
            Bucket:           aws.String(bucketName),
            Key:              aws.String(objectName),
            UploadID:         aws.String(uploadID),
            PartNumberMarker: aws.Long(nextPartNumberMarker),
            MaxParts:         aws.Long(10000),
        })
        if err != nil {
            return nil, err
        }
        for _, part := range listObjPartsResult.Parts {
            *part.ETag = strings.TrimPrefix(*part.ETag, "")
            *part.ETag = strings.TrimSuffix(*part.ETag, "")
            partsInfo[*part.PartNumber] = *part
        }
        nextPartNumberMarker = *listObjPartsResult.NextPartNumberMarker
        if !*listObjPartsResult.IsTruncated {
            break
        }
    }

    return partsInfo, nil
}

func (ks3 *kS3Client) CheckFinished(ctx context.Context, objectName, uploadId string, partNumber int) (bool, string) {
    partInfos, err := ks3.listObjectParts(ctx, ks3.conf.DefaultBucket, objectName, uploadId)
    if err != nil {
        log.Context(ctx).Errorf("ks3 CheckFinished listObjectParts err: %+v", err)
        return false, ""
    }
    if len(partInfos) < partNumber {
        log.Context(ctx).Errorf("ks3 CheckFinished [len<partNumber]  partInfos： %+v", partInfos)
        return false, ""
    }
    var completedPartMap = make(map[int]s3.Part)
    multiFile := MultipartFile{}
    multiFile.UploadId = uploadId
    multiFile.BucketName = ks3.conf.DefaultBucket
    multiFile.ObjectName = objectName
    multiFile.CompletedPartsKs3 = make([]*s3.CompletedPart, len(partInfos))
    for _, partInfo := range partInfos {
        multiFile.CompletedPartsKs3[*partInfo.PartNumber-1] = &s3.CompletedPart{PartNumber: partInfo.PartNumber, ETag: partInfo.ETag}
        completedPartMap[int(*partInfo.PartNumber)] = partInfo
    }
    err = ks3.complete(ctx, ks3.conf.DefaultBucket, &multiFile)
    if err != nil {
        log.Context(ctx).Errorf("ks3 CheckFinished complete err: %+v", err)
        return false, ""
    }

    return true, objectName
}
