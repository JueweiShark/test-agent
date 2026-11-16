package oss

import (
    "bytes"
    "context"
    "fmt"
    "io"
    "path/filepath"
    "strings"

    "github.com/go-kratos/kratos/v2/errors"
    kr "github.com/go-kratos/kratos/v2/errors"
    "github.com/go-kratos/kratos/v2/log"
    "github.com/volcengine/ve-tos-golang-sdk/v2/tos"
    "github.com/volcengine/ve-tos-golang-sdk/v2/tos/enum"
    apiErrs "wuwen.com/wwuc/api/errors"
    "wuwen.com/wwuc/internal/conf"
)

type s3Client struct {
    client *tos.ClientV2
    conf   *OssConf
}

func NewTOSClient(conf *conf.Bootstrap, logger log.Logger) StorageService {
    BackendResourcesBasePath := fmt.Sprintf("%s/%s", conf.Oss.TempDir, "resource")
    TempPathAbs := fmt.Sprintf("%s/%s", BackendResourcesBasePath, conf.Oss.TempPath)
    ossConf := OssConf{
        OssType:                  "tos",
        BasePath:                 conf.Oss.BasePath,
        Endpoint:                 conf.Oss.Endpoint,
        DefaultBucket:            conf.Oss.BucketName,
        TempPath:                 conf.Oss.TempPath,
        TempPathAbs:              TempPathAbs,
        BackendResourcesBasePath: BackendResourcesBasePath,
        ReplaceFlag:              conf.Oss.Replace,
        InternalEndpoint:         conf.Oss.InternalEndpoint,
        InternalProtocol:         conf.Oss.InternalProtocol,
    }
    log.Infof("ossConf %+v", ossConf)
    credential := tos.NewStaticCredentials(conf.Oss.GetAccessKey(), conf.Oss.GetSecretKey())
    tosClient, err := tos.NewClientV2(ossConf.Endpoint, tos.WithCredentials(credential), tos.WithRegion(conf.Oss.GetRegion()))
    if err != nil {
        fmt.Println("Error:", err)
        panic(err)
    }

    m := s3Client{
        client: tosClient,
        conf:   &ossConf,
    }
    // m.InitClient(conf.Oss, logger)
    return &m
}

func (s *s3Client) InitClient(conf *conf.OSS, logger log.Logger) {
    //c, err := bos.NewClient(conf.AccessKey, conf.SecretKey, conf.Endpoint)
    //if err != nil {
    //    log.Error("bos client init error", err)
    //}
    //log.Info("bos service init", c)
    //b.client = c
}

func (s *s3Client) DownloadFile(ctx context.Context, objectName, filePath string) *errors.Error {
    _, err := s.client.DownloadFile(ctx, &tos.DownloadFileInput{
        HeadObjectV2Input: tos.HeadObjectV2Input{
            Bucket: s.GetDeFaultBucket(),
            Key:    objectName,
        },
        FilePath: filePath,
    })
    if err != nil {
        log.Context(ctx).Errorf("get object file: %s failed, %s", objectName, err.Error())
        return apiErrs.FileNotExist
    }

    return nil
}

func (s *s3Client) UploadFile(ctx context.Context, objectName, filePath, contentType string) (string, *errors.Error) {
    _, err := s.client.PutObjectFromFile(ctx, &tos.PutObjectFromFileInput{
        PutObjectBasicInput: tos.PutObjectBasicInput{
            Bucket:      s.GetDeFaultBucket(),
            Key:         objectName,
            ContentType: contentType,
        },
        FilePath: filePath,
    })
    if err != nil {
        log.Context(ctx).Errorf("put object error: %v", err)
        return "", apiErrs.FileUploadFail
    }

    split := strings.Split(objectName, "/")
    fileName := split[len(split)-1]
    return s.PresignedGetUrl(ctx, objectName, WithFileName(fileName), WithNeedInternal(true))
}

func (s *s3Client) PresignedGetUrl(ctx context.Context, objectName string, opt ...OSSOpt) (string, *errors.Error) {
    o := s.parseOSSOpt(opt)
    urlValue := make(map[string]string)
    if o.NeedDownLoad {
        index := strings.LastIndex(objectName, "/")
        var fileName string
        if index > 0 && index+1 < len(objectName) {
            fileName = objectName[index+1:]
        } else {
            fileName = objectName
        }
        urlValue["response-content-type"] = "application/octet-stream"
        urlValue["response-content-disposition"] = fmt.Sprintf("attachment; filename=%s", fileName)
    }
    if len(o.FileName) > 0 {
        urlValue["response-content-disposition"] = fmt.Sprintf("attachment; filename=%s", o.FileName)
    }
    presignedURL, err := s.client.PreSignedURL(&tos.PreSignedURLInput{
        HTTPMethod: enum.HttpMethodGet,
        Bucket:     o.BucketName,
        Key:        objectName,
        Expires:    604800,
        Header:     urlValue,
    })
    if err != nil {
        log.Context(ctx).Errorf("pre signed url error: %v", err)
        return "", apiErrs.OssPresignedUrlFail.WithCause(err)
    }
    return s.replaceUrl(ctx, presignedURL.SignedUrl), nil
}

func (s *s3Client) PresignedGetUrlWithBucket(ctx context.Context, bucket, objectName string, opt ...OSSOpt) (string, *errors.Error) {
    o := s.parseOSSOpt(opt)
    urlValue := make(map[string]string)
    if o.NeedDownLoad {
        index := strings.LastIndex(objectName, "/")
        var fileName string
        if index > 0 && index+1 < len(objectName) {
            fileName = objectName[index+1:]
        } else {
            fileName = objectName
        }
        urlValue["response-content-type"] = "application/octet-stream"
        urlValue["response-content-disposition"] = fmt.Sprintf("attachment; filename=%s", fileName)
    }
    if len(o.FileName) > 0 {
        urlValue["response-content-disposition"] = fmt.Sprintf("attachment; filename=%s", o.FileName)
    }
    presignedURL, err := s.client.PreSignedURL(&tos.PreSignedURLInput{
        HTTPMethod: enum.HttpMethodGet,
        Bucket:     bucket,
        Key:        objectName,
        Expires:    604800,
        Header:     urlValue,
    })
    if err != nil {
        log.Context(ctx).Errorf("pre signed url error: %v", err)
        return "", apiErrs.OssPresignedUrlFail.WithCause(err)
    }
    return s.replaceUrl(ctx, presignedURL.SignedUrl), nil
}

func (s *s3Client) PresignedPutUrl(ctx context.Context, objectName string, opt ...OSSOpt) (string, *errors.Error) {
    o := s.parseOSSOpt(opt)
    presignedURL, err := s.client.PreSignedURL(&tos.PreSignedURLInput{
        HTTPMethod: enum.HttpMethodPut,
        Bucket:     o.BucketName,
        Key:        objectName,
        Expires:    604800,
    })
    if err != nil {
        log.Context(ctx).Errorf("pre signed url error: %v", err)
        return "", apiErrs.OssPresignedUrlFail.WithCause(err)
    }
    return s.replaceUrl(ctx, presignedURL.SignedUrl), nil
}

func (s *s3Client) GetDeFaultBucket() string {
    return s.conf.DefaultBucket
}

func (s *s3Client) GetOssConf() *OssConf {
    return s.conf
}

func (s *s3Client) UploadFileList(ctx context.Context, rootPath string, folder string, fileList map[string]struct{}) (map[string]string, error) {
    results := make(map[string]string)
    for f := range fileList {
        localPath := filepath.Join(rootPath, f)
        objectName := strings.TrimPrefix(filepath.Join(folder, f), "/")
        contentType := "application/octet-stream"
        _, err := s.UploadFile(ctx, objectName, localPath, contentType)
        if err != nil {
            return nil, err
        }
        results[f] = objectName
    }
    return results, nil
}

func (s *s3Client) UploadBytes(ctx context.Context, objectName string, fileBytes []byte, opt ...OSSOpt) *errors.Error {
    o := s.parseOSSOpt(opt)

    _, err := s.client.PutObjectV2(ctx, &tos.PutObjectV2Input{
        PutObjectBasicInput: tos.PutObjectBasicInput{
            Bucket: o.BucketName,
            Key:    objectName,
        },
        Content: bytes.NewReader(fileBytes),
    })
    if err != nil {
        log.Context(ctx).Errorf("upload memery file error, %s", err.Error())
        return apiErrs.FileUploadFail
    }
    return nil
}

func (s *s3Client) ListFiles(ctx context.Context, prefix string, opt ...OSSOpt) ([]string, *errors.Error) {
    o := s.parseOSSOpt(opt)
    // 列举指定桶中所有对象
    truncated := true
    continuationToken := ""
    var objects []string
    for truncated {

        output, err := s.client.ListObjectsType2(ctx, &tos.ListObjectsType2Input{
            Bucket:            o.BucketName,
            Prefix:            prefix,
            MaxKeys:           1000,
            ContinuationToken: continuationToken,
        })
        if err != nil {
            log.Context(ctx).Errorf("List file failed, %s", err.Error())
            return nil, apiErrs.OssListFail
        }
        for _, obj := range output.Contents {
            // 对象 Key
            objects = append(objects, obj.Key)
        }
        truncated = output.IsTruncated
        continuationToken = output.NextContinuationToken
    }
    return objects, nil
}

func (s *s3Client) ListObjects(ctx context.Context, prefix string, opt ...OSSOpt) ([]*FileObject, *errors.Error) {
    o := s.parseOSSOpt(opt)
    // 列举指定桶中所有对象
    truncated := true
    continuationToken := ""
    var objects []*FileObject
    for truncated {

        output, err := s.client.ListObjectsType2(ctx, &tos.ListObjectsType2Input{
            Bucket:            o.BucketName,
            Prefix:            prefix,
            MaxKeys:           1000,
            ContinuationToken: continuationToken,
        })
        if err != nil {
            log.Context(ctx).Errorf("List file failed, %s", err.Error())
            return nil, apiErrs.OssListFail
        }
        for _, obj := range output.Contents {
            // 对象 Key
            objects = append(objects, &FileObject{
                Path: obj.Key,
                Size: obj.Size,
            })
        }
        truncated = output.IsTruncated
        continuationToken = output.NextContinuationToken
    }
    return objects, nil
}

func (s *s3Client) ReadFile(ctx context.Context, objectKey string, dataCh chan<- []byte, opt ...OSSOpt) ([]byte, *errors.Error) {
    response, err := s.client.GetObjectV2(ctx, &tos.GetObjectV2Input{
        Bucket: s.conf.DefaultBucket,
        Key:    objectKey,
    })
    if err != nil {
        log.Context(ctx).Errorf("get object file failed, %s", err.Error())
        return nil, apiErrs.FileNotExist
    }
    defer response.Content.Close()
    var result []byte
    buf := make([]byte, 2*1024) // 每次读取 2 字节

    for {
        n, err := response.Content.Read(buf)
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

func (s *s3Client) parseOSSOpt(oss []OSSOpt) *opt {
    o := &opt{
        FileName:     "",
        NeedDownLoad: false,
        NeedInternal: false,
        BucketName:   s.GetDeFaultBucket(),
    }
    for _, ossOpt := range oss {
        ossOpt(o)
    }
    return o
}

func (s *s3Client) replaceUrl(ctx context.Context, originalURL string) string {
    endPoint := "http://localhost"
    //if reqCtx, ok := common.ReqCtxFromContext(ctx); ok {
    //    if len(reqCtx.EndPoint) > 0 && len(reqCtx.Protocol) > 0 {
    //        endPoint = fmt.Sprintf("%s://%s", reqCtx.Protocol, reqCtx.EndPoint)
    //    }
    //    if len(s.conf.InternalEndpoint) > 0 {
    //        endPoint = fmt.Sprintf("%s://%s", reqCtx.Protocol, s.conf.InternalEndpoint)
    //    }
    //}
    originalURL = strings.ReplaceAll(originalURL, fmt.Sprintf("%s://%s", "http", s.conf.Endpoint),
        fmt.Sprintf("%s/%s", endPoint, MINIO))
    return originalURL
}

// ========= 分片上传 =========
func (s *s3Client) NewMultipartUpload(ctx context.Context, objectName string) (string, error) {
    mimeType := "application/octet-stream"
    input := &tos.CreateMultipartUploadV2Input{
        Bucket:      s.GetDeFaultBucket(),
        Key:         objectName,
        ContentType: mimeType,
        ACL:         enum.ACLPrivate,
    }
    state, err := s.client.CreateMultipartUploadV2(ctx, input)
    if err != nil {
        return "", err
    }
    if state == nil {
        return "", fmt.Errorf("object %s is nil", objectName)
    }
    return state.UploadID, nil
}

// PresignedPutUrlPart 生成分片上传的预签名 URL（返回格式: "<url>|||<Authorization>")
// objectName: 对象名(不进行额外 encode，SDK 会正确处理)
// uploadID: multipart upload id
// partNumber: 分片编号（你的上层是 string，这里直接用 string）
func (s *s3Client) PresignedPutUrlPart(ctx context.Context, objectName, uploadID, partNumber string, oss ...OSSOpt) (string, *kr.Error) {
    input := &tos.PreSignedURLInput{
        Bucket:     s.GetDeFaultBucket(),
        HTTPMethod: enum.HttpMethodPut,
        Key:        objectName,
        Query: map[string]string{
            "uploadId":   uploadID,   // 通过CreateMultipartUpload获取
            "partNumber": partNumber, // 分片序号从1开始
        },
    }
    out, err := s.client.PreSignedURL(input)
    if err != nil {
        return "", apiErrs.OssPresignedUrlFail
    }
    return out.SignedUrl, nil
}

func (s *s3Client) CheckFinished(ctx context.Context, objectName, uploadId string, partNumber int) (bool, string) {
    var allParts []tos.UploadedPartV2
    partNumberMarker := 0
    for {
        input := &tos.ListPartsInput{
            Bucket:           s.GetDeFaultBucket(),
            Key:              objectName,
            UploadID:         uploadId,
            PartNumberMarker: partNumberMarker,
            MaxParts:         100,
        }
        list, err := s.client.ListParts(ctx, input)
        if err != nil {
            return false, ""
        }
        allParts = append(allParts, list.Parts...)
        if !list.IsTruncated {
            break
        }
        partNumberMarker = list.NextPartNumberMarker
    }

    if len(allParts) < partNumber {
        return false, ""
    }

    var etags []string
    for _, p := range allParts {
        etags = append(etags, p.ETag)
    }
    mimeType := "application/octet-stream"

    err := s.FinishMultipartUpload(objectName, uploadId, etags, mimeType)
    if err != nil {
        return false, ""
    }

    return true, objectName
}

// FinishMultipartUpload 完成分片上传。
// uploadID: 初始化分片时返回的 ID
// objectName: 文件名（和 UploadPart 用的一致）
// etags: 每个分片返回的 etag，索引即 partNumber-1
func (s *s3Client) FinishMultipartUpload(objectName, uploadID string, etags []string, mimeType string) error {
    var parts []tos.UploadedPartV2
    for k, part := range etags {
        parts = append(parts, tos.UploadedPartV2{
            PartNumber: k + 1,
            ETag:       part,
        })
    }
    input := &tos.CompleteMultipartUploadV2Input{
        Bucket:   s.GetDeFaultBucket(),
        Key:      objectName,
        UploadID: uploadID,
        Parts:    parts,
    }
    _, err := s.client.CompleteMultipartUploadV2(context.Background(), input)
    if err != nil {
        return err
    }
    return nil
}
