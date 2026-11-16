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
    "time"

    apiErr "wuwen.com/wwuc/api/errors"
    "wuwen.com/wwuc/internal/conf"
    "wuwen.com/wwuc/util/wuwen"

    kr "github.com/go-kratos/kratos/v2/errors"
    "github.com/go-kratos/kratos/v2/log"

    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/credentials"
    "github.com/aws/aws-sdk-go/aws/session"
    s3types "github.com/aws/aws-sdk-go/service/s3"
)

type cosClient struct {
    client         *s3types.S3
    clientInternal *s3types.S3
    tenClient      *s3types.S3
    conf           *OssConf
    log            *log.Helper
}

// NewCosClient 构造腾讯云 COS 后端
func NewCosClient(conf *conf.Bootstrap, logger log.Logger) StorageService {
    BackendResourcesBasePath := fmt.Sprintf("%s/%s", conf.Oss.TempDir, "resource")
    TempPathAbs := fmt.Sprintf("%s/%s", BackendResourcesBasePath, conf.Oss.TempPath)
    c := &cosClient{
        conf: &OssConf{
            OssType:                  "cos",
            Endpoint:                 conf.Oss.Endpoint,
            Region:                   conf.Oss.Region,
            DefaultBucket:            conf.Oss.BucketName,
            TempPath:                 conf.Oss.TempPath,
            TempPathAbs:              TempPathAbs,
            BackendResourcesBasePath: BackendResourcesBasePath,
            InternalEndpoint:         conf.Oss.InternalEndpoint,
            InternalProtocol:         conf.Oss.InternalProtocol,
        },
        log: log.NewHelper(log.With(logger, "module", "oss/cos")),
    }
    c.InitClient(conf.Oss, logger)
    return c
}

// InitClient 初始化 COS S3 兼容客户端
func (c *cosClient) InitClient(conf *conf.OSS, logger log.Logger) {
    cre := credentials.NewStaticCredentials(conf.GetAccessKey(), conf.GetSecretKey(), "")
    sess := session.Must(session.NewSession(&aws.Config{
        Region:           aws.String(c.getRegion()),
        Credentials:      cre,
        Endpoint:         aws.String(c.conf.Endpoint),
        DisableSSL:       aws.Bool(false), // COS 用 HTTPS
        S3ForcePathStyle: aws.Bool(false), // 虚拟主机风格
        MaxRetries:       aws.Int(3),
    }))
    c.client = s3types.New(sess)
    sessInternal := session.Must(session.NewSession(&aws.Config{
        Region:           aws.String(c.getRegion()),
        Credentials:      cre,
        Endpoint:         aws.String(c.conf.InternalEndpoint),
        DisableSSL:       aws.Bool(false), // COS 用 HTTPS
        S3ForcePathStyle: aws.Bool(false), // 虚拟主机风格
        MaxRetries:       aws.Int(3),
    }))
    c.clientInternal = s3types.New(sessInternal)
}

// getRegion 返回 COS region（ap-guangzhou 等），未配置时从 endpoint 解析
func (c *cosClient) getRegion() string {
    if c.conf.Region != "" {
        return c.conf.Region
    }
    // endpoint 形如 cos.ap-guangzhou.myqcloud.com，解析出 ap-guangzhou
    ep := c.conf.Endpoint
    ep = strings.TrimPrefix(ep, "https://")
    ep = strings.TrimPrefix(ep, "http://")
    ep = strings.TrimPrefix(ep, "cos.")
    if idx := strings.Index(ep, "."); idx > 0 {
        return ep[:idx]
    }
    return "ap-guangzhou"
}

func (c *cosClient) parseOSSOpt(oss []OSSOpt) *opt {
    var o = &opt{
        FileName:     "",
        NeedDownLoad: false,
        NeedInternal: false,
        BucketName:   c.GetDeFaultBucket(),
    }
    for _, ossOpt := range oss {
        ossOpt(o)
    }
    return o
}

func (c *cosClient) GetDeFaultBucket() string {
    return c.conf.DefaultBucket
}

func (c *cosClient) GetOssConf() *OssConf {
    return c.conf
}

// CreateBucket 检查 bucket 是否存在，不存在则创建
func (c *cosClient) CreateBucket(ctx context.Context, bucket string) error {
    if c.client == nil {
        return errors.New("cos client is nil")
    }
    _, err := c.client.HeadBucket(&s3types.HeadBucketInput{
        Bucket: aws.String(bucket),
    })
    if err == nil {
        return nil
    }
    // COS 创建桶需要 CreateBucketConfiguration 指定 region
    _, err = c.client.CreateBucket(&s3types.CreateBucketInput{
        Bucket: aws.String(bucket),
        CreateBucketConfiguration: &s3types.CreateBucketConfiguration{
            LocationConstraint: aws.String(c.getRegion()),
        },
    })
    if err != nil {
        log.Context(ctx).Errorf("cos CreateBucket err: %v", err)
        return errors.New("create bucket err")
    }
    return nil
}

// UploadFile 上传文件并返回可下载的 presigned url
func (c *cosClient) UploadFile(ctx context.Context, objectName, filePath, contentType string) (string, *kr.Error) {
    if c.client == nil {
        return "", kr.New(1, "cos client is nil", "")
    }
    bucket := c.conf.DefaultBucket
    if err := c.CreateBucket(ctx, bucket); err != nil {
        log.Context(ctx).Errorf("CreateBucket fail %v", err)
    }

    fileBytes, err := os.ReadFile(filePath)
    if err != nil {
        return "", kr.New(1, fmt.Sprintf("get upload file stat fail%v", err), "")
    }
    ct := contentType
    if ct == "" {
        ct = "application/octet-stream"
    }
    _, err = c.client.PutObject(&s3types.PutObjectInput{
        Bucket:      aws.String(c.GetDeFaultBucket()),
        Key:         aws.String(objectName),
        Body:        bytes.NewReader(fileBytes),
        ContentType: aws.String(ct),
    })
    if err != nil {
        return "", kr.New(1, fmt.Sprintf("upload file fail%v", err), "")
    }
    log.Infof("file %s upload sucess,key %s ", filePath, objectName)

    split := strings.Split(objectName, "/")
    fileName := split[len(split)-1]
    return c.PresignedGetUrl(ctx, objectName, WithFileName(fileName), WithNeedInternal(true))
}

// UploadFileList 批量上传文件，返回本地文件名到 OSS key 的映射
func (c *cosClient) UploadFileList(ctx context.Context, localDir string, targetPath string, filePathMap map[string]struct{}) (map[string]string, error) {
    if c.client == nil {
        log.Context(ctx).Errorf("ks3 client is nil")
        return nil, apiErr.OssClientInvalid
    }
    var files []*os.File
    defer func() {
        for _, file := range files {
            _ = file.Close()
        }
    }()
    if err := c.CreateBucket(ctx, c.GetDeFaultBucket()); err != nil {
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
            _, e = c.clientInternal.PutObject(&s3types.PutObjectInput{
                Bucket:   aws.String(c.GetDeFaultBucket()),
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

// PresignedGetUrl 生成下载 url，支持下载文件名覆盖
func (c *cosClient) PresignedGetUrl(ctx context.Context, objectName string, opt ...OSSOpt) (string, *kr.Error) {
    if c.client == nil {
        return "", apiErr.OssClientInvalid
    }
    o := c.parseOSSOpt(opt)
    input := &s3types.GetObjectInput{
        Bucket: aws.String(c.GetDeFaultBucket()),
        Key:    aws.String(objectName),
    }
    if o.NeedDownLoad {
        fileName := o.FileName
        if fileName == "" {
            if idx := strings.LastIndex(objectName, "/"); idx >= 0 {
                fileName = objectName[idx+1:]
            } else {
                fileName = objectName
            }
        }
        input.ResponseContentType = aws.String("application/octet-stream")
        input.ResponseContentDisposition = aws.String(fmt.Sprintf("attachment; filename=%s", fileName))
    } else if len(o.FileName) > 0 {
        input.ResponseContentDisposition = aws.String(fmt.Sprintf("attachment; filename=%s", o.FileName))
    }
    req, _ := c.client.GetObjectRequest(input)
    url, err := req.Presign(24 * time.Hour)
    if err != nil {
        return "", apiErr.OssPresignedUrlFail.WithCause(err)
    }
    return url, apiErr.NilErr
}

// PresignedGetUrlByBucket 指定 bucket 生成下载 url
func (c *cosClient) PresignedGetUrlByBucket(ctx context.Context, objectName, bucketName string, oss ...OSSOpt) (string, *kr.Error) {
    if c.client == nil {
        return "", apiErr.OssClientInvalid
    }
    if len(bucketName) == 0 {
        bucketName = c.GetDeFaultBucket()
    }
    o := c.parseOSSOpt(oss)
    input := &s3types.GetObjectInput{
        Bucket: aws.String(bucketName),
        Key:    aws.String(objectName),
    }
    if o.NeedDownLoad {
        fileName := o.FileName
        if fileName == "" {
            if idx := strings.LastIndex(objectName, "/"); idx >= 0 {
                fileName = objectName[idx+1:]
            } else {
                fileName = objectName
            }
        }
        input.ResponseContentType = aws.String("application/octet-stream")
        input.ResponseContentDisposition = aws.String(fmt.Sprintf("attachment; filename=%s", fileName))
    } else if len(o.FileName) > 0 {
        input.ResponseContentDisposition = aws.String(fmt.Sprintf("attachment; filename=%s", o.FileName))
    }
    req, _ := c.client.GetObjectRequest(input)
    url, err := req.Presign(24 * time.Hour)
    if err != nil {
        return "", apiErr.OssPresignedUrlFail.WithCause(err)
    }
    return url, nil
}

// PresignedPutUrl 生成上传 url
func (c *cosClient) PresignedPutUrl(ctx context.Context, objectName string, oss ...OSSOpt) (string, *kr.Error) {
    if c.client == nil {
        return "", apiErr.OssClientInvalid
    }
    if err := c.CreateBucket(ctx, c.GetDeFaultBucket()); err != nil {
        log.Context(ctx).Warnf("CreateBucket fail %v", err)
    }
    req, _ := c.client.PutObjectRequest(&s3types.PutObjectInput{
        Bucket: aws.String(c.GetDeFaultBucket()),
        Key:    aws.String(objectName),
        //ContentType: aws.String("application/octet-stream"),
    })
    url, err := req.Presign(24 * time.Hour)
    if err != nil {
        return "", apiErr.OssPresignedUrlFail.WithCause(err)
    }
    return url, nil
}

// RenameFile 通过 CopyObject 实现重命名
func (c *cosClient) RenameFile(srcObjectName, destObjectName string) error {
    ctx, cancelFunc := context.WithCancel(context.Background())
    defer cancelFunc()
    bucket := c.GetDeFaultBucket()
    src := fmt.Sprintf("%s/%s", bucket, srcObjectName)
    _, err := c.client.CopyObject(&s3types.CopyObjectInput{
        Bucket:     aws.String(bucket),
        CopySource: aws.String(src),
        Key:        aws.String(destObjectName),
    })
    if err != nil {
        c.log.WithContext(ctx).Errorf("rename file %s fail %v", srcObjectName, err)
        return err
    }
    return nil
}

// ObjectExists 通过 HeadObject 判断对象是否存在
func (c *cosClient) ObjectExists(ctx context.Context, objectName string) bool {
    if err := c.CreateBucket(ctx, c.GetDeFaultBucket()); err != nil {
        log.Context(ctx).Warnf("CreateBucket fail %v", err)
    }
    _, err := c.client.HeadObject(&s3types.HeadObjectInput{
        Bucket: aws.String(c.conf.DefaultBucket),
        Key:    aws.String(objectName),
    })
    if err != nil {
        return false
    }
    return true
}

// ObjectDelete 删除单个对象
func (c *cosClient) ObjectDelete(ctx context.Context, objectName string) bool {
    if err := c.CreateBucket(ctx, c.GetDeFaultBucket()); err != nil {
        log.Context(ctx).Warnf("CreateBucket fail %v", err)
    }
    _, err := c.client.DeleteObject(&s3types.DeleteObjectInput{
        Bucket: aws.String(c.conf.DefaultBucket),
        Key:    aws.String(objectName),
    })
    if err != nil {
        return false
    }
    return true
}

// NewMultipartUpload 初始化分片上传，返回 uploadId
func (c *cosClient) NewMultipartUpload(ctx context.Context, objectName string) (string, error) {
    bucketName := c.conf.DefaultBucket
    input := &s3types.CreateMultipartUploadInput{
        Bucket:      aws.String(bucketName),
        Key:         aws.String(objectName),
        ContentType: aws.String("application/octet-stream"),
    }
    ret, err := c.client.CreateMultipartUpload(input)
    if err != nil {
        return "", err
    }
    return *ret.UploadId, nil
}

// PresignedPutUrlPart 生成分片上传 url
func (c *cosClient) PresignedPutUrlPart(ctx context.Context, objectName, uploadID, partNumber string, oss ...OSSOpt) (string, *kr.Error) {
    if c.client == nil {
        return "", apiErr.OssClientInvalid
    }
    if err := c.CreateBucket(ctx, c.GetDeFaultBucket()); err != nil {
        log.Context(ctx).Warnf("CreateBucket fail %v", err)
    }
    number, _ := strconv.Atoi(partNumber)
    req, _ := c.client.UploadPartRequest(&s3types.UploadPartInput{
        Bucket:     aws.String(c.GetDeFaultBucket()),
        Key:        aws.String(objectName),
        UploadId:   aws.String(uploadID),
        PartNumber: aws.Int64(int64(number)),
    })
    url, err := req.Presign(24 * time.Hour)
    if err != nil {
        return "", apiErr.OssPresignedUrlFail.WithCause(err)
    }
    return url, nil
}

// listObjectParts 列出已上传的分片
func (c *cosClient) listObjectParts(ctx context.Context, bucketName, objectName, uploadID string) (map[int64]*s3types.Part, error) {
    var nextPartNumberMarker *int64
    partsInfo := make(map[int64]*s3types.Part)
    for {
        input := &s3types.ListPartsInput{
            Bucket:   aws.String(bucketName),
            Key:      aws.String(objectName),
            UploadId: aws.String(uploadID),
            MaxParts: aws.Int64(1000),
        }
        if nextPartNumberMarker != nil {
            input.PartNumberMarker = nextPartNumberMarker
        }
        resp, err := c.client.ListParts(input)
        if err != nil {
            return nil, err
        }
        for _, part := range resp.Parts {
            partsInfo[*part.PartNumber] = part
        }
        if resp.IsTruncated == nil || !*resp.IsTruncated {
            break
        }
        nextPartNumberMarker = resp.NextPartNumberMarker
    }
    return partsInfo, nil
}

// complete 合并分片（修复 ks3 complete 排序错位的 bug）
func (c *cosClient) complete(ctx context.Context, bucketName string, cpFile *MultipartFile) error {
    sort.Slice(cpFile.CompletedPartsCos, func(i, j int) bool {
        return *cpFile.CompletedPartsCos[i].PartNumber < *cpFile.CompletedPartsCos[j].PartNumber
    })
    _, err := c.client.CompleteMultipartUpload(&s3types.CompleteMultipartUploadInput{
        Bucket:   aws.String(bucketName),
        Key:      aws.String(cpFile.ObjectName),
        UploadId: aws.String(cpFile.UploadId),
        MultipartUpload: &s3types.CompletedMultipartUpload{
            Parts: cpFile.CompletedPartsCos,
        },
    })
    if err != nil {
        log.Context(ctx).Errorf("cos CompleteMultipartUpload err: %s", err)
        return err
    }
    return nil
}

// CheckFinished 检查分片是否全部上传完成，完成后合并
func (c *cosClient) CheckFinished(ctx context.Context, objectName, uploadId string, partNumber int) (bool, string) {
    partInfos, err := c.listObjectParts(ctx, c.conf.DefaultBucket, objectName, uploadId)
    if err != nil {
        log.Context(ctx).Errorf("cos CheckFinished listObjectParts err: %+v", err)
        return false, ""
    }
    if len(partInfos) < partNumber {
        log.Context(ctx).Errorf("cos CheckFinished [len<partNumber] partInfos: %+v", partInfos)
        return false, ""
    }
    multiFile := MultipartFile{
        UploadId:          uploadId,
        BucketName:        c.conf.DefaultBucket,
        ObjectName:        objectName,
        CompletedPartsCos: make([]*s3types.CompletedPart, len(partInfos)),
    }
    for _, partInfo := range partInfos {
        multiFile.CompletedPartsCos[*partInfo.PartNumber-1] = &s3types.CompletedPart{
            PartNumber: partInfo.PartNumber,
            ETag:       partInfo.ETag,
        }
    }
    err = c.complete(ctx, c.conf.DefaultBucket, &multiFile)
    if err != nil {
        log.Context(ctx).Errorf("cos CheckFinished complete err: %+v", err)
        return false, ""
    }
    return true, objectName
}

// GetPathObjects 获取指定前缀下的所有对象 key
func (c *cosClient) GetPathObjects(ctx context.Context, pathName, bucketName string) ([]string, error) {
    res := make([]string, 0)
    if bucketName == "" {
        bucketName = c.GetDeFaultBucket()
    }
    params := &s3types.ListObjectsInput{
        Bucket: aws.String(bucketName),
        Prefix: aws.String(pathName),
    }
    for {
        resp, err := c.client.ListObjects(params)
        if err != nil {
            return res, err
        }
        for _, obj := range resp.Contents {
            res = append(res, *obj.Key)
        }
        if resp.IsTruncated == nil || !*resp.IsTruncated {
            break
        }
        params.Marker = resp.Contents[len(resp.Contents)-1].Key
    }
    return res, nil
}

// DeleteBucketPrefix 删除指定前缀下所有对象（COS 不支持 KS3 的 DeleteBucketPrefix 扩展，用 ListObjects + DeleteObjects）
func (c *cosClient) DeleteBucketPrefix(ctx context.Context, prefix, bucketName string) error {
    if bucketName == "" {
        bucketName = c.GetDeFaultBucket()
    }
    listParams := &s3types.ListObjectsInput{
        Bucket: aws.String(bucketName),
        Prefix: aws.String(prefix),
    }
    for {
        resp, err := c.client.ListObjects(listParams)
        if err != nil {
            return fmt.Errorf("cos DeleteBucketPrefix ListObjects err: %w", err)
        }
        if len(resp.Contents) == 0 {
            break
        }
        var objects []*s3types.ObjectIdentifier
        for _, obj := range resp.Contents {
            objects = append(objects, &s3types.ObjectIdentifier{Key: obj.Key})
        }
        _, err = c.client.DeleteObjects(&s3types.DeleteObjectsInput{
            Bucket: aws.String(bucketName),
            Delete: &s3types.Delete{
                Objects: objects,
                Quiet:   aws.Bool(true),
            },
        })
        if err != nil {
            return fmt.Errorf("cos DeleteBucketPrefix DeleteObjects err: %w", err)
        }
        if resp.IsTruncated == nil || !*resp.IsTruncated {
            break
        }
        listParams.Marker = resp.Contents[len(resp.Contents)-1].Key
    }
    return nil
}

func (c *cosClient) UploadBytes(ctx context.Context, objectName string, fileBytes []byte, opt ...OSSOpt) *kr.Error {
    o := c.parseOSSOpt(opt)
    err := c.CreateBucket(ctx, o.BucketName)
    if err != nil {
        log.Context(ctx).Warnf("CreateBucket fail %v", err)
    }

    client := c.client
    if o.NeedInternal {
        client = c.clientInternal
    }
    _, err = client.PutObject(&s3types.PutObjectInput{
        Bucket:      aws.String(c.conf.DefaultBucket),       // bucket名称
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

func (c *cosClient) PresignedGetUrlWithBucket(ctx context.Context, bucket, objectName string, opt ...OSSOpt) (string, *kr.Error) {
    if c.client == nil {
        return "", apiErr.OssClientInvalid
    }
    o := c.parseOSSOpt(opt)
    input := &s3types.GetObjectInput{
        Bucket: aws.String(bucket),
        Key:    aws.String(objectName),
    }
    if o.NeedDownLoad {
        fileName := o.FileName
        if fileName == "" {
            if idx := strings.LastIndex(objectName, "/"); idx >= 0 {
                fileName = objectName[idx+1:]
            } else {
                fileName = objectName
            }
        }
        input.ResponseContentType = aws.String("application/octet-stream")
        input.ResponseContentDisposition = aws.String(fmt.Sprintf("attachment; filename=%s", fileName))
    } else if len(o.FileName) > 0 {
        input.ResponseContentDisposition = aws.String(fmt.Sprintf("attachment; filename=%s", o.FileName))
    }
    req, _ := c.client.GetObjectRequest(input)
    url, err := req.Presign(24 * time.Hour)
    if err != nil {
        return "", apiErr.OssPresignedUrlFail.WithCause(err)
    }
    return url, apiErr.NilErr
}

func (c *cosClient) ListFiles(ctx context.Context, prefix string, opt ...OSSOpt) ([]string, *kr.Error) {
    log.Context(ctx).Infof("list files pre fix:%s", prefix)
    o := c.parseOSSOpt(opt)
    err := c.CreateBucket(ctx, o.BucketName)
    if err != nil {
        log.Context(ctx).Warnf("CreateBucket fail %v", err)
    }
    var objects []string
    var marker string
    client := c.client
    if o.NeedInternal {
        client = c.clientInternal
    }
    for {
        ListObjectsOutput, err := client.ListObjects(&s3types.ListObjectsInput{
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

func (c *cosClient) DownloadFile(ctx context.Context, objectName, filePath string) *kr.Error {
    err := c.CreateBucket(ctx, c.GetDeFaultBucket())
    if err != nil {
        log.Context(ctx).Warnf("CreateBucket fail %v", err)
    }
    // 1. 从 COS 获取对象
    resp, err := c.clientInternal.GetObject(&s3types.GetObjectInput{
        Bucket: aws.String(c.GetDeFaultBucket()),
        Key:    aws.String(objectName),
    })
    if err != nil {
        log.Context(ctx).Errorf("cos get object failed, %s", err.Error())
        return apiErr.FileNotExist
    }
    defer resp.Body.Close()
    // 2. 创建本地目录（如不存在）
    if dir := filepath.Dir(filePath); dir != "" {
        if e := os.MkdirAll(dir, os.ModePerm); e != nil {
            log.Context(ctx).Errorf("cos mkdir failed, %s", e.Error())
            return apiErr.FileUploadFail
        }
    }
    // 3. 创建本地文件并写入
    file, err := os.Create(filePath)
    if err != nil {
        log.Context(ctx).Errorf("cos create local file failed, %s", err.Error())
        return apiErr.FileUploadFail
    }
    defer file.Close()
    if _, err := io.Copy(file, resp.Body); err != nil {
        log.Context(ctx).Errorf("cos download file write failed, %s", err.Error())
        return apiErr.FileUploadFail
    }
    return apiErr.NilErr
}

func (c *cosClient) ListObjects(ctx context.Context, prefix string, opt ...OSSOpt) ([]*FileObject, *kr.Error) {
    log.Context(ctx).Infof("list objects prefix:%s", prefix)
    o := c.parseOSSOpt(opt)
    err := c.CreateBucket(ctx, o.BucketName)
    if err != nil {
        log.Context(ctx).Warnf("CreateBucket fail %v", err)
    }
    var objects []*FileObject
    var marker string
    client := c.client
    if o.NeedInternal {
        client = c.clientInternal
    }
    for {
        ListObjectsOutput, err := client.ListObjects(&s3types.ListObjectsInput{
            Bucket: aws.String(o.BucketName),
            Marker: aws.String(marker),
            Prefix: aws.String(prefix),
        })
        if err != nil {
            log.Context(ctx).Errorf("list objects error, %s", err.Error())
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

func (c *cosClient) ReadFile(ctx context.Context, objectKey string, dataCh chan<- []byte, opt ...OSSOpt) ([]byte, *kr.Error) {
    o := c.parseOSSOpt(opt)
    err := c.CreateBucket(ctx, o.BucketName)
    if err != nil {
        log.Context(ctx).Warnf("CreateBucket fail %v", err)
    }
    client := c.client
    if o.NeedInternal {
        client = c.clientInternal
    }
    resp, err := client.GetObject(&s3types.GetObjectInput{
        Bucket: aws.String(o.BucketName),
        Key:    aws.String(objectKey),
    })
    if err != nil {
        log.Context(ctx).Errorf("read file %s fail %v", objectKey, err)
        return nil, apiErr.FileNotExist
    }
    defer resp.Body.Close()
    result, err := io.ReadAll(resp.Body)
    if err != nil {
        log.Context(ctx).Errorf("read file %s fail %v", objectKey, err)
        return nil, apiErr.FileNotExist
    }
    if dataCh != nil {
        dataCh <- result
    }
    return result, apiErr.NilErr
}
