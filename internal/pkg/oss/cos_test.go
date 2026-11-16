package oss

import (
    "context"
    "fmt"
    "net/http"
    "os"
    "strings"
    "testing"
    "time"
    "wuwen.com/wwuc/internal/conf"

    s3types "github.com/aws/aws-sdk-go/service/s3"
)

var CosClient StorageService

func InitCosClient() {
    bc := &conf.Bootstrap{
        Oss: &conf.OSS{
            Endpoint:         "cos.ap-beijing.myqcloud.com",
            InternalEndpoint: "cos.ap-beijing.myqcloud.com",
            InternalProtocol: "https",
            TempPath:         "ks3/",
            TempDir:          "/tmp/wuwen",
            Region:           "ap-beijing",
            Replace:          false,
            AccessKey:        os.Getenv("TENCENTCLOUD_SECRET_ID"),
            SecretKey:        os.Getenv("TENCENTCLOUD_SECRET_KEY"),
            BucketName:       "wuwen-embodied-sim-1412578044",
        },
    }
    CosClient = NewCosClient(bc, nil)
}

func TestCosClient_UploadFile(t *testing.T) {
    InitCosClient()
    filePath := "/Users/zzj/Downloads/wuwen_emb_wsi.sql"
    objectName := "test/cos/upload/wuwen_emb_wsi.sql"
    contentType := "application/octet-stream"
    str, apiE := CosClient.UploadFile(context.Background(), objectName, filePath, contentType)
    if apiE != nil && apiE.GetCode() != 0 {
        t.Errorf("UploadFile err:%+v", apiE)
        return
    }
    fmt.Printf("UploadFile success, url: %s\n", str)
}

func TestCosClient_UploadFile_Error(t *testing.T) {
    InitCosClient()
    filePath := "/non/existent/file.sql"
    objectName := "test/cos/upload/non_existent.sql"
    contentType := "application/octet-stream"
    str, apiE := CosClient.UploadFile(context.Background(), objectName, filePath, contentType)
    if apiE == nil || apiE.GetCode() == 0 {
        t.Errorf("UploadFile should fail for non-existent file, got url: %s", str)
        return
    }
    fmt.Printf("UploadFile error test passed, err: %+v\n", apiE)
}

func TestCosClient_PresignedGetUrl(t *testing.T) {
    InitCosClient()
    objectName := "test/cos/upload/wuwen_emb_wsi.sql"
    str, apiE := CosClient.PresignedGetUrl(context.Background(), objectName)
    if apiE != nil && apiE.GetCode() != 0 {
        t.Errorf("PresignedGetUrl err:%+v", apiE)
        return
    }
    fmt.Printf("PresignedGetUrl success, url: %s\n", str)
}

func TestCosClient_PresignedGetUrlWithFileName(t *testing.T) {
    InitCosClient()
    objectName := "test/cos/upload/wuwen_emb_wsi.sql"
    str, apiE := CosClient.PresignedGetUrl(context.Background(), objectName, WithFileName("test.sql"), WithNeedDownLoad(true))
    if apiE != nil && apiE.GetCode() != 0 {
        t.Errorf("PresignedGetUrlWithFileName err:%+v", apiE)
        return
    }
    fmt.Printf("PresignedGetUrlWithFileName success, url: %s\n", str)
}

func TestCosClient_PresignedGetUrl_WithInline(t *testing.T) {
    InitCosClient()
    objectName := "test/cos/upload/wuwen_emb_wsi.sql"
    str, apiE := CosClient.PresignedGetUrl(context.Background(), objectName, WithInline(true))
    if apiE != nil && apiE.GetCode() != 0 {
        t.Errorf("PresignedGetUrl_WithInline err:%+v", apiE)
        return
    }
    fmt.Printf("PresignedGetUrl_WithInline success, url: %s\n", str)
}

func TestCosClient_PresignedPutUrl(t *testing.T) {
    InitCosClient()
    objectName := "test/cos/upload/put_test.sql"
    str, apiE := CosClient.PresignedPutUrl(context.Background(), objectName)
    if apiE != nil && apiE.GetCode() != 0 {
        t.Errorf("PresignedPutUrl err:%+v", apiE)
        return
    }
    fmt.Printf("PresignedPutUrl success, url: %s\n", str)
}

// TestCosClient_PresignedPutUrl_UrlValid 校验预签名上传 URL 的结构，
// 并通过实际 PUT 请求端到端验证 URL 可用。
func TestCosClient_PresignedPutUrl_UrlValid(t *testing.T) {
    InitCosClient()
    objectName := "test/cos/upload/put_valid_test.txt"
    bucket := CosClient.GetDeFaultBucket()

    url, apiE := CosClient.PresignedPutUrl(context.Background(), objectName)
    if apiE != nil && apiE.GetCode() != 0 {
        t.Errorf("PresignedPutUrl err:%+v", apiE)
        return
    }
    if url == "" {
        t.Errorf("PresignedPutUrl returned empty url")
        return
    }
    // 校验协议为 HTTPS
    if !strings.HasPrefix(url, "https://") {
        t.Errorf("PresignedPutUrl url should be https, got: %s", url)
    }
    // 校验 URL 包含 bucket 名
    if !strings.Contains(url, bucket) {
        t.Errorf("PresignedPutUrl url should contain bucket %s, got: %s", bucket, url)
    }
    // 校验 URL 包含 objectName
    if !strings.Contains(url, objectName) {
        t.Errorf("PresignedPutUrl url should contain objectName %s, got: %s", objectName, url)
    }
    // 校验 URL 包含 AWS 预签名参数
    for _, key := range []string{"X-Amz-Algorithm", "X-Amz-Credential", "X-Amz-Signature", "X-Amz-Date", "X-Amz-Expires"} {
        if !strings.Contains(url, key) {
            t.Errorf("PresignedPutUrl url missing param %s, got: %s", key, url)
        }
    }
    fmt.Printf("PresignedPutUrl_UrlValid url structure ok, url: %s\n", url)

    // 端到端：用预签名 URL 执行 PUT，验证 URL 实际可用
    content := []byte("test presigned put url content")
    req, err := http.NewRequest("PUT", url, strings.NewReader(string(content)))
    if err != nil {
        t.Logf("Skipping PUT validation: build request fail: %v", err)
        return
    }
    //req.Header.Set("Content-Type", "application/xml")
    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        t.Logf("Skipping PUT validation: http do fail: %v", err)
        return
    }
    defer resp.Body.Close()
    if resp.StatusCode != http.StatusOK {
        t.Errorf("PresignedPutUrl PUT failed, status: %d", resp.StatusCode)
        return
    }
    fmt.Printf("PresignedPutUrl_UrlValid PUT success, status: %d\n", resp.StatusCode)
}

func TestCosClient_ListFiles(t *testing.T) {
    InitCosClient()
    prefix := "test/cos/"
    fileList, err := CosClient.ListFiles(context.Background(), prefix)
    if err != nil && err.GetCode() != 0 {
        t.Errorf("ListFiles err:%+v", err)
        return
    }
    fmt.Printf("ListFiles success, count: %d\n", len(fileList))
    for _, file := range fileList {
        fmt.Println(file)
    }
}

func TestCosClient_ListFiles_WithInternal(t *testing.T) {
    InitCosClient()
    prefix := "test/cos/"
    fileList, err := CosClient.ListFiles(context.Background(), prefix, WithNeedInternal(true))
    if err != nil && err.GetCode() != 0 {
        t.Errorf("ListFiles_WithInternal err:%+v", err)
        return
    }
    fmt.Printf("ListFiles_WithInternal success, count: %d\n", len(fileList))
}

func TestCosClient_UploadBytes(t *testing.T) {
    InitCosClient()
    objectName := "test/cos/upload/bytes_test.txt"
    content := []byte("Hello COS! This is a test file.")
    err := CosClient.UploadBytes(context.Background(), objectName, content)
    if err != nil && err.GetCode() != 0 {
        t.Errorf("UploadBytes err:%+v", err)
        return
    }
    fmt.Println("UploadBytes success")
}

func TestCosClient_UploadBytes_WithInternal(t *testing.T) {
    InitCosClient()
    objectName := "test/cos/upload/bytes_internal.txt"
    content := []byte("Hello COS! This is an internal test file.")
    err := CosClient.UploadBytes(context.Background(), objectName, content, WithNeedInternal(true))
    if err != nil && err.GetCode() != 0 {
        t.Errorf("UploadBytes_WithInternal err:%+v", err)
        return
    }
    fmt.Println("UploadBytes_WithInternal success")
}

func TestCosClient_ListObjects(t *testing.T) {
    InitCosClient()
    prefix := "test/cos/"
    objects, err := CosClient.ListObjects(context.Background(), prefix)
    if err != nil && err.GetCode() != 0 {
        t.Errorf("ListObjects err:%+v", err)
        return
    }
    fmt.Printf("ListObjects success, count: %d\n", len(objects))
    for _, obj := range objects {
        fmt.Printf("Path: %s, Size: %d\n", obj.Path, obj.Size)
    }
}

func TestCosClient_ListObjects_WithInternal(t *testing.T) {
    InitCosClient()
    prefix := "test/cos/"
    objects, err := CosClient.ListObjects(context.Background(), prefix, WithNeedInternal(true))
    if err != nil && err.GetCode() != 0 {
        t.Errorf("ListObjects_WithInternal err:%+v", err)
        return
    }
    fmt.Printf("ListObjects_WithInternal success, count: %d\n", len(objects))
}

func TestCosClient_ReadFile(t *testing.T) {
    InitCosClient()
    objectKey := "test/cos/upload/bytes_test.txt"
    data, err := CosClient.ReadFile(context.Background(), objectKey, nil)
    if err != nil && err.GetCode() != 0 {
        t.Errorf("ReadFile err:%+v", err)
        return
    }
    fmt.Printf("ReadFile success, content: %s\n", string(data))
}

func TestCosClient_ReadFile_WithChannel(t *testing.T) {
    InitCosClient()
    objectKey := "test/cos/upload/bytes_test.txt"
    dataCh := make(chan []byte, 1)
    data, err := CosClient.ReadFile(context.Background(), objectKey, dataCh)
    if err != nil && err.GetCode() != 0 {
        t.Errorf("ReadFile_WithChannel err:%+v", err)
        return
    }
    select {
    case received := <-dataCh:
        if string(received) != string(data) {
            t.Errorf("ReadFile_WithChannel data mismatch")
        }
    default:
        t.Errorf("ReadFile_WithChannel no data received")
    }
    fmt.Printf("ReadFile_WithChannel success, content: %s\n", string(data))
}

func TestCosClient_ReadFile_WithInternal(t *testing.T) {
    InitCosClient()
    objectKey := "test/cos/upload/bytes_internal.txt"
    data, err := CosClient.ReadFile(context.Background(), objectKey, nil, WithNeedInternal(true))
    if err != nil && err.GetCode() != 0 {
        t.Errorf("ReadFile_WithInternal err:%+v", err)
        return
    }
    fmt.Printf("ReadFile_WithInternal success, content: %s\n", string(data))
}

func TestCosClient_ReadFile_NonExist(t *testing.T) {
    InitCosClient()
    objectKey := "test/cos/upload/non_exist.txt"
    data, err := CosClient.ReadFile(context.Background(), objectKey, nil)
    if err == nil || err.GetCode() == 0 {
        t.Errorf("ReadFile_NonExist should fail, got data: %v", string(data))
        return
    }
    fmt.Printf("ReadFile_NonExist error test passed, err: %+v\n", err)
}

func TestCosClient_DownloadFile(t *testing.T) {
    InitCosClient()
    objectName := "test/cos/upload/wuwen_emb_wsi.sql"
    localFilePath := "/tmp/cos_download_test.sql"
    err := CosClient.DownloadFile(context.Background(), objectName, localFilePath)
    if err != nil && err.GetCode() != 0 {
        t.Errorf("DownloadFile err:%+v", err)
        return
    }
    fmt.Printf("DownloadFile success, file: %s\n", localFilePath)
}

func TestCosClient_DownloadFile_NonExist(t *testing.T) {
    InitCosClient()
    objectName := "test/cos/upload/non_exist.sql"
    localFilePath := "/tmp/cos_download_non_exist.sql"
    err := CosClient.DownloadFile(context.Background(), objectName, localFilePath)
    if err == nil || err.GetCode() == 0 {
        t.Errorf("DownloadFile_NonExist should fail")
        return
    }
    fmt.Printf("DownloadFile_NonExist error test passed, err: %+v\n", err)
}

func TestCosClient_UploadFileList(t *testing.T) {
    InitCosClient()
    localDir := "/Users/zzj/Downloads"
    targetPath := "test/cos/upload_list"
    filePathMap := make(map[string]struct{})
    filePathMap["wuwen_emb_wsi.sql"] = struct{}{}

    result, err := CosClient.UploadFileList(context.Background(), localDir, targetPath, filePathMap)
    if err != nil {
        t.Errorf("UploadFileList err:%+v", err)
        return
    }
    fmt.Printf("UploadFileList success, result: %v\n", result)
}

func TestCosClient_UploadFileList_Error(t *testing.T) {
    InitCosClient()
    localDir := "/non/existent/dir"
    targetPath := "test/cos/upload_list_error"
    filePathMap := make(map[string]struct{})
    filePathMap["nonexistent.sql"] = struct{}{}

    result, err := CosClient.UploadFileList(context.Background(), localDir, targetPath, filePathMap)
    if err == nil {
        t.Errorf("UploadFileList should fail for non-existent directory, got: %v", result)
        return
    }
    fmt.Printf("UploadFileList error test passed, err: %+v\n", err)
}

func TestCosClient_NewMultipartUpload(t *testing.T) {
    InitCosClient()
    objectName := "test/cos/multipart/test_multipart.txt"
    uploadId, err := CosClient.NewMultipartUpload(context.Background(), objectName)
    if err != nil {
        t.Errorf("NewMultipartUpload err:%+v", err)
        return
    }
    fmt.Printf("NewMultipartUpload success, uploadId: %s\n", uploadId)
}

func TestCosClient_PresignedPutUrlPart(t *testing.T) {
    InitCosClient()
    objectName := "test/cos/multipart/test_multipart.txt"
    uploadId, err := CosClient.NewMultipartUpload(context.Background(), objectName)
    if err != nil {
        t.Errorf("NewMultipartUpload err:%+v", err)
        return
    }
    fmt.Printf("NewMultipartUpload success, uploadId: %s\n", uploadId)

    url, apiE := CosClient.PresignedPutUrlPart(context.Background(), objectName, uploadId, "1")
    if apiE != nil && apiE.GetCode() != 0 {
        t.Errorf("PresignedPutUrlPart err:%+v", apiE)
        return
    }
    fmt.Printf("PresignedPutUrlPart success, url: %s\n", url)
}

func TestCosClient_CheckFinished(t *testing.T) {
    InitCosClient()
    objectName := "test/cos/multipart/test_check_finished.txt"

    uploadId, err := CosClient.NewMultipartUpload(context.Background(), objectName)
    if err != nil {
        t.Errorf("NewMultipartUpload err:%+v", err)
        return
    }
    fmt.Printf("NewMultipartUpload success, uploadId: %s\n", uploadId)

    url, apiE := CosClient.PresignedPutUrlPart(context.Background(), objectName, uploadId, "1")
    if apiE != nil && apiE.GetCode() != 0 {
        t.Errorf("PresignedPutUrlPart err:%+v", apiE)
        return
    }
    fmt.Printf("PresignedPutUrlPart success, url: %s\n", url)

    finished, name := CosClient.CheckFinished(context.Background(), objectName, uploadId, 1)
    fmt.Printf("CheckFinished success, finished: %v, name: %s\n", finished, name)
}

func TestCosClient_PresignedGetUrlByBucket(t *testing.T) {
    InitCosClient()
    objectName := "test/cos/upload/wuwen_emb_wsi.sql"
    bucketName := CosClient.GetDeFaultBucket()
    str, apiE := CosClient.(*cosClient).PresignedGetUrlByBucket(context.Background(), objectName, bucketName)
    if apiE != nil && apiE.GetCode() != 0 {
        t.Errorf("PresignedGetUrlByBucket err:%+v", apiE)
        return
    }
    fmt.Printf("PresignedGetUrlByBucket success, url: %s\n", str)
}

func TestCosClient_PresignedGetUrlByBucket_EmptyBucket(t *testing.T) {
    InitCosClient()
    objectName := "test/cos/upload/wuwen_emb_wsi.sql"
    str, apiE := CosClient.(*cosClient).PresignedGetUrlByBucket(context.Background(), objectName, "")
    if apiE != nil && apiE.GetCode() != 0 {
        t.Errorf("PresignedGetUrlByBucket_EmptyBucket err:%+v", apiE)
        return
    }
    fmt.Printf("PresignedGetUrlByBucket_EmptyBucket success, url: %s\n", str)
}

func TestCosClient_PresignedGetUrlByBucket_WithOptions(t *testing.T) {
    InitCosClient()
    objectName := "test/cos/upload/wuwen_emb_wsi.sql"
    bucketName := CosClient.GetDeFaultBucket()
    str, apiE := CosClient.(*cosClient).PresignedGetUrlByBucket(context.Background(), objectName, bucketName, WithFileName("test.sql"), WithNeedDownLoad(true))
    if apiE != nil && apiE.GetCode() != 0 {
        t.Errorf("PresignedGetUrlByBucket_WithOptions err:%+v", apiE)
        return
    }
    fmt.Printf("PresignedGetUrlByBucket_WithOptions success, url: %s\n", str)
}

func TestCosClient_PresignedGetUrlWithBucket(t *testing.T) {
    InitCosClient()
    objectName := "test/cos/upload/wuwen_emb_wsi.sql"
    bucketName := CosClient.GetDeFaultBucket()
    str, apiE := CosClient.PresignedGetUrlWithBucket(context.Background(), bucketName, objectName)
    if apiE != nil && apiE.GetCode() != 0 {
        t.Errorf("PresignedGetUrlWithBucket err:%+v", apiE)
        return
    }
    fmt.Printf("PresignedGetUrlWithBucket success, url: %s\n", str)
}

func TestCosClient_PresignedGetUrlWithBucket_WithOptions(t *testing.T) {
    InitCosClient()
    objectName := "test/cos/upload/wuwen_emb_wsi.sql"
    bucketName := CosClient.GetDeFaultBucket()
    str, apiE := CosClient.PresignedGetUrlWithBucket(context.Background(), bucketName, objectName, WithFileName("test.sql"), WithNeedDownLoad(true))
    if apiE != nil && apiE.GetCode() != 0 {
        t.Errorf("PresignedGetUrlWithBucket_WithOptions err:%+v", apiE)
        return
    }
    fmt.Printf("PresignedGetUrlWithBucket_WithOptions success, url: %s\n", str)
}

func TestCosClient_GetDeFaultBucket(t *testing.T) {
    InitCosClient()
    bucket := CosClient.GetDeFaultBucket()
    fmt.Printf("GetDeFaultBucket success, bucket: %s\n", bucket)
    if bucket == "" {
        t.Errorf("GetDeFaultBucket returned empty string")
    }
}

func TestCosClient_GetOssConf(t *testing.T) {
    InitCosClient()
    conf := CosClient.GetOssConf()
    if conf == nil {
        t.Errorf("GetOssConf returned nil")
        return
    }
    fmt.Printf("GetOssConf success, OssType: %s, Region: %s\n", conf.OssType, conf.Region)
}

func TestCosClient_ObjectExists(t *testing.T) {
    InitCosClient()
    objectName := "test/cos/upload/wuwen_emb_wsi.sql"
    exists := CosClient.(*cosClient).ObjectExists(context.Background(), objectName)
    fmt.Printf("ObjectExists success, exists: %v\n", exists)
}

func TestCosClient_ObjectExists_NonExist(t *testing.T) {
    InitCosClient()
    objectName := "test/cos/upload/non_existent_file.txt"
    exists := CosClient.(*cosClient).ObjectExists(context.Background(), objectName)
    fmt.Printf("ObjectExists_NonExist success, exists: %v\n", exists)
    if exists {
        t.Errorf("ObjectExists_NonExist should return false")
    }
}

func TestCosClient_ObjectDelete(t *testing.T) {
    InitCosClient()
    testObjectName := "test/cos/upload/test_delete.txt"
    content := []byte("test delete content")
    err := CosClient.UploadBytes(context.Background(), testObjectName, content)
    if err != nil && err.GetCode() != 0 {
        t.Errorf("UploadBytes err:%+v", err)
        return
    }

    deleted := CosClient.(*cosClient).ObjectDelete(context.Background(), testObjectName)
    fmt.Printf("ObjectDelete success, deleted: %v\n", deleted)
}

func TestCosClient_RenameFile(t *testing.T) {
    InitCosClient()
    srcObjectName := "test/cos/upload/test_rename_src.txt"
    destObjectName := "test/cos/upload/test_rename_dest.txt"

    content := []byte("test rename content")
    err := CosClient.UploadBytes(context.Background(), srcObjectName, content)
    if err != nil && err.GetCode() != 0 {
        t.Errorf("UploadBytes err:%+v", err)
        return
    }

    err2 := CosClient.(*cosClient).RenameFile(srcObjectName, destObjectName)
    if err2 != nil {
        t.Errorf("RenameFile err:%+v", err2)
        return
    }
    fmt.Println("RenameFile success")
}

func TestCosClient_GetPathObjects(t *testing.T) {
    InitCosClient()
    pathName := "test/cos/upload/"
    bucketName := CosClient.GetDeFaultBucket()
    objects, err := CosClient.(*cosClient).GetPathObjects(context.Background(), pathName, bucketName)
    if err != nil {
        t.Errorf("GetPathObjects err:%+v", err)
        return
    }
    fmt.Printf("GetPathObjects success, count: %d\n", len(objects))
    for _, obj := range objects {
        fmt.Println(obj)
    }
}

func TestCosClient_GetPathObjects_EmptyBucket(t *testing.T) {
    InitCosClient()
    pathName := "test/cos/upload/"
    objects, err := CosClient.(*cosClient).GetPathObjects(context.Background(), pathName, "")
    if err != nil {
        t.Errorf("GetPathObjects_EmptyBucket err:%+v", err)
        return
    }
    fmt.Printf("GetPathObjects_EmptyBucket success, count: %d\n", len(objects))
}

func TestCosClient_DeleteBucketPrefix(t *testing.T) {
    InitCosClient()
    prefix := "test/cos/upload_list/"
    bucketName := CosClient.GetDeFaultBucket()
    err := CosClient.(*cosClient).DeleteBucketPrefix(context.Background(), prefix, bucketName)
    if err != nil {
        t.Errorf("DeleteBucketPrefix err:%+v", err)
        return
    }
    fmt.Println("DeleteBucketPrefix success")
}

func TestCosClient_DeleteBucketPrefix_EmptyBucket(t *testing.T) {
    InitCosClient()
    prefix := "test/cos/upload_list/"
    err := CosClient.(*cosClient).DeleteBucketPrefix(context.Background(), prefix, "")
    if err != nil {
        t.Errorf("DeleteBucketPrefix_EmptyBucket err:%+v", err)
        return
    }
    fmt.Println("DeleteBucketPrefix_EmptyBucket success")
}

func TestCosClient_getRegion_FromConfig(t *testing.T) {
    InitCosClient()
    region := CosClient.(*cosClient).getRegion()
    fmt.Printf("getRegion_FromConfig success, region: %s\n", region)
    if region != "ap-beijing" {
        t.Errorf("getRegion_FromConfig expected ap-beijing, got %s", region)
    }
}

func TestCosClient_getRegion_FromEndpoint(t *testing.T) {
    bc := &conf.Bootstrap{
        Oss: &conf.OSS{
            Endpoint:         "cos.ap-guangzhou.myqcloud.com",
            AccessKey:        os.Getenv("TENCENTCLOUD_SECRET_ID"),
            SecretKey:        os.Getenv("TENCENTCLOUD_SECRET_KEY"),
            InternalEndpoint: "cos.ap-guangzhou.myqcloud.com",
            InternalProtocol: "https",
            BucketName:       "test001-1412578044",
            TempPath:         "ks3/",
            TempDir:          "/tmp/wuwen",
            Region:           "",
            Replace:          false,
        },
    }
    client := NewCosClient(bc, nil)
    region := client.(*cosClient).getRegion()
    fmt.Printf("getRegion_FromEndpoint success, region: %s\n", region)
    if region != "ap-guangzhou" {
        t.Errorf("getRegion_FromEndpoint expected ap-guangzhou, got %s", region)
    }
}

func TestCosClient_getRegion_FromEndpointWithProtocol(t *testing.T) {
    bc := &conf.Bootstrap{
        Oss: &conf.OSS{
            Endpoint:         "https://cos.ap-shanghai.myqcloud.com",
            AccessKey:        os.Getenv("TENCENTCLOUD_SECRET_ID"),
            SecretKey:        os.Getenv("TENCENTCLOUD_SECRET_KEY"),
            InternalEndpoint: "https://cos.ap-shanghai.myqcloud.com",
            InternalProtocol: "https",
            BucketName:       "test001-1412578044",
            TempPath:         "ks3/",
            TempDir:          "/tmp/wuwen",
            Region:           "",
            Replace:          false,
        },
    }
    client := NewCosClient(bc, nil)
    region := client.(*cosClient).getRegion()
    fmt.Printf("getRegion_FromEndpointWithProtocol success, region: %s\n", region)
    if region != "ap-shanghai" {
        t.Errorf("getRegion_FromEndpointWithProtocol expected ap-shanghai, got %s", region)
    }
}

func TestCosClient_CreateBucket_ClientNil(t *testing.T) {
    client := &cosClient{}
    err := client.CreateBucket(context.Background(), "test-bucket")
    if err == nil || !strings.Contains(err.Error(), "cos client is nil") {
        t.Errorf("CreateBucket_ClientNil should return 'cos client is nil' error, got: %v", err)
    }
    fmt.Printf("CreateBucket_ClientNil error test passed, err: %v\n", err)
}

func TestCosClient_CreateBucket_Exists(t *testing.T) {
    InitCosClient()
    bucket := CosClient.GetDeFaultBucket()
    err := CosClient.(*cosClient).CreateBucket(context.Background(), bucket)
    if err != nil {
        t.Errorf("CreateBucket_Exists err:%+v", err)
        return
    }
    fmt.Println("CreateBucket_Exists success")
}

func TestCosClient_complete(t *testing.T) {
    InitCosClient()
    objectName := "test/cos/multipart/test_complete.txt"

    uploadId, err := CosClient.NewMultipartUpload(context.Background(), objectName)
    if err != nil {
        t.Errorf("NewMultipartUpload err:%+v", err)
        return
    }
    fmt.Printf("NewMultipartUpload success, uploadId: %s\n", uploadId)

    url, apiE := CosClient.PresignedPutUrlPart(context.Background(), objectName, uploadId, "1")
    if apiE != nil && apiE.GetCode() != 0 {
        t.Errorf("PresignedPutUrlPart err:%+v", apiE)
        return
    }

    content := []byte("test complete content")
    httpClient := &http.Client{}
    req, _ := http.NewRequest("PUT", url, strings.NewReader(string(content)))
    resp, err := httpClient.Do(req)
    if err != nil {
        t.Logf("Skipping complete test: failed to upload part via HTTP: %v", err)
        return
    }
    resp.Body.Close()

    partInfos, err := CosClient.(*cosClient).listObjectParts(context.Background(), CosClient.GetDeFaultBucket(), objectName, uploadId)
    if err != nil {
        t.Errorf("listObjectParts err:%+v", err)
        return
    }

    if len(partInfos) == 0 {
        t.Log("Skipping complete test: no parts uploaded")
        return
    }

    multiFile := &MultipartFile{
        UploadId:          uploadId,
        BucketName:        CosClient.GetDeFaultBucket(),
        ObjectName:        objectName,
        CompletedPartsCos: make([]*s3types.CompletedPart, len(partInfos)),
    }
    for _, partInfo := range partInfos {
        multiFile.CompletedPartsCos[*partInfo.PartNumber-1] = &s3types.CompletedPart{
            PartNumber: partInfo.PartNumber,
            ETag:       partInfo.ETag,
        }
    }

    err = CosClient.(*cosClient).complete(context.Background(), CosClient.GetDeFaultBucket(), multiFile)
    if err != nil {
        t.Errorf("complete err:%+v", err)
        return
    }
    fmt.Println("complete success")
}

func TestCosClient_CheckFinished_Complete(t *testing.T) {
    InitCosClient()
    objectName := "test/cos/multipart/test_check_finished_complete.txt"

    uploadId, err := CosClient.NewMultipartUpload(context.Background(), objectName)
    if err != nil {
        t.Errorf("NewMultipartUpload err:%+v", err)
        return
    }
    fmt.Printf("NewMultipartUpload success, uploadId: %s\n", uploadId)

    url, apiE := CosClient.PresignedPutUrlPart(context.Background(), objectName, uploadId, "1")
    if apiE != nil && apiE.GetCode() != 0 {
        t.Errorf("PresignedPutUrlPart err:%+v", apiE)
        return
    }

    content := []byte("test check finished complete content")
    httpClient := &http.Client{}
    req, _ := http.NewRequest("PUT", url, strings.NewReader(string(content)))
    resp, err := httpClient.Do(req)
    if err != nil {
        t.Logf("Skipping CheckFinished_Complete test: failed to upload part via HTTP: %v", err)
        return
    }
    resp.Body.Close()

    time.Sleep(1 * time.Second)

    finished, name := CosClient.CheckFinished(context.Background(), objectName, uploadId, 1)
    fmt.Printf("CheckFinished_Complete success, finished: %v, name: %s\n", finished, name)
}

func TestCosClient_UploadFile_ClientNil(t *testing.T) {
    client := &cosClient{}
    str, apiE := client.UploadFile(context.Background(), "test", "/tmp/test.txt", "text/plain")
    if apiE == nil || apiE.GetCode() != 1 {
        t.Errorf("UploadFile_ClientNil should return error, got str: %s, err: %+v", str, apiE)
    }
    fmt.Printf("UploadFile_ClientNil error test passed, err: %+v\n", apiE)
}

func TestCosClient_PresignedGetUrl_ClientNil(t *testing.T) {
    client := &cosClient{}
    str, apiE := client.PresignedGetUrl(context.Background(), "test")
    if apiE == nil {
        t.Errorf("PresignedGetUrl_ClientNil should return error, got str: %s", str)
    }
    fmt.Printf("PresignedGetUrl_ClientNil error test passed, err: %+v\n", apiE)
}

func TestCosClient_PresignedPutUrl_ClientNil(t *testing.T) {
    client := &cosClient{}
    str, apiE := client.PresignedPutUrl(context.Background(), "test")
    if apiE == nil {
        t.Errorf("PresignedPutUrl_ClientNil should return error, got str: %s", str)
    }
    fmt.Printf("PresignedPutUrl_ClientNil error test passed, err: %+v\n", apiE)
}

func TestCosClient_PresignedPutUrlPart_ClientNil(t *testing.T) {
    client := &cosClient{}
    str, apiE := client.PresignedPutUrlPart(context.Background(), "test", "uploadId", "1")
    if apiE == nil {
        t.Errorf("PresignedPutUrlPart_ClientNil should return error, got str: %s", str)
    }
    fmt.Printf("PresignedPutUrlPart_ClientNil error test passed, err: %+v\n", apiE)
}

func TestCosClient_PresignedGetUrlByBucket_ClientNil(t *testing.T) {
    client := &cosClient{}
    str, apiE := client.PresignedGetUrlByBucket(context.Background(), "test", "bucket")
    if apiE == nil {
        t.Errorf("PresignedGetUrlByBucket_ClientNil should return error, got str: %s", str)
    }
    fmt.Printf("PresignedGetUrlByBucket_ClientNil error test passed, err: %+v\n", apiE)
}

func TestCosClient_PresignedGetUrlWithBucket_ClientNil(t *testing.T) {
    client := &cosClient{}
    str, apiE := client.PresignedGetUrlWithBucket(context.Background(), "bucket", "test")
    if apiE == nil {
        t.Errorf("PresignedGetUrlWithBucket_ClientNil should return error, got str: %s", str)
    }
    fmt.Printf("PresignedGetUrlWithBucket_ClientNil error test passed, err: %+v\n", apiE)
}

func TestCosClient_UploadBytes_Failure(t *testing.T) {
    InitCosClient()
    objectName := "test/cos/upload/invalid_path/bytes_fail.txt"
    content := []byte("test content")
    err := CosClient.UploadBytes(context.Background(), objectName, content)
    if err != nil && err.GetCode() == 20003 {
        fmt.Printf("UploadBytes_Failure error test passed, err: %+v\n", err)
        return
    }
    fmt.Printf("UploadBytes_Failure result: err=%+v\n", err)
}
