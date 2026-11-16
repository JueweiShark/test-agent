package oss

import (
    "context"
    "fmt"
    "os"
    "path/filepath"
    "strings"
    "testing"
    "wuwen.com/wwuc/internal/conf"
)

var OssClient StorageService

func InitMinioClient() {
    bc := &conf.Bootstrap{
        Oss: &conf.OSS{
            Endpoint:         "tos-cn-beijing.volces.com",
            AccessKey:        os.Getenv("VOLCENGINE_ACCESS_KEY_ID"),
            SecretKey:        os.Getenv("VOLCENGINE_SECRET_ACCESS_KEY"),
            InternalEndpoint: "tos-cn-beijing.volces.com",
            InternalProtocol: "http",
            BucketName:       "wwsim-test",
            TempPath:         "minio/",
            TempDir:          "/tmp/wuwen",
            Region:           "cn-beijing",
            Replace:          false,
        },
    }
    OssClient = NewTOSClient(bc, nil)
}

func TestUploadFile(t *testing.T) {
    InitMinioClient()
    filePath := "/Users/zzj/Downloads/data/规划数据/archive.zip"
    objectName := "test/example/data/pnc/2025-12-02-10-01-32_B1F-PC-110.zip"
    contentType := "application/octet-stream"
    str, apiE := OssClient.UploadFile(context.Background(), objectName, filePath, contentType)
    if apiE != nil {
        _ = fmt.Errorf("err:%+v \n", apiE)
    }
    fmt.Println(str)
}

func TestGetFileSign(t *testing.T) {
    InitMinioClient()
    objectName := "1848974093960425473/2025-12-20/6b822b24-a7e9-4298-af79-81e576457d47_0.zip"
    str, apiE := OssClient.PresignedGetUrl(context.Background(), objectName)
    if apiE != nil {
        _ = fmt.Errorf("err:%+v \n", apiE)
    }
    fmt.Println(str)
}

func GetFilesInDir(dirPath string) (map[string]struct{}, error) {
    fileMap := make(map[string]struct{})

    entries, err := os.ReadDir(dirPath)
    if err != nil {
        return nil, err
    }

    for _, entry := range entries {
        if !entry.IsDir() {
            // 获取绝对路径
            absPath := entry.Name()
            fileMap[absPath] = struct{}{}
        }
    }

    return fileMap, nil
}

func TestDownLoadFolder(t *testing.T) {
    InitMinioClient()
    //path := "./tmp/20251211223000/规划数据/2025-12-02-10-01-32_B1F-PC-110"
    objectName := "wuwen/test/data"
    fileList, err := OssClient.ListFiles(context.Background(), objectName)
    if err != nil && err.GetCode() != 0 {
        _ = fmt.Errorf("err:%+v \n", err)
    }

    for _, file := range fileList {
        //fileName := filepath.Base(file)
        fmt.Println(file)
    }
}

func TestSync(t *testing.T) {
    InitMinioClient()
    // 获取所有文件路径
    filePath := "wuwen/test/"
    filePaths, err := OssClient.ListFiles(context.Background(), filePath)
    if err != nil {
        panic(err)
    }

    // 构建文件树
    fileTree := BuildFileTreeFromPath(filePath, filePaths)

    // 现在可以根据需要处理目录
    for path, node := range fileTree {
        if !node.IsFile {
            // 这是目录节点
            fmt.Printf("目录: %s, 子节点数: %d\n", path, len(node.Children))

            // 可以在这里将目录信息保存到数据库
            fmt.Printf("info:%v, parentPath:%s, depth:%d  \n", node.IsFile, node.ParentPath, node.Depth)

            fmt.Printf("chiled:%v \n", node.Children)

        }
    }
    fmt.Println("success")
}

type FileNode struct {
    Path       string   // 完整路径
    IsFile     bool     // 是否为文件
    ParentPath string   // 父级路径
    Name       string   // 文件名或目录名
    Depth      int      // 路径深度
    Children   []string // 子节点路径列表
}

// 辅助函数：获取父路径下的直接子路径
func getImmediateChild(parent, child string) string {
    if !strings.HasPrefix(child, parent+"/") {
        return child
    }

    remainder := strings.TrimPrefix(child, parent+"/")
    parts := strings.Split(remainder, "/")
    if len(parts) > 0 {
        return parent + "/" + parts[0]
    }
    return child
}

func BuildFileTreeFromPath(startPath string, filePaths []string) map[string]*FileNode {
    nodes := make(map[string]*FileNode)

    // 确保起始路径以/结尾，方便前缀匹配
    if !strings.HasSuffix(startPath, "/") {
        startPath = startPath + "/"
    }

    for _, fullPath := range filePaths {
        // 只处理起始路径下的文件
        if !strings.HasPrefix(fullPath, startPath) {
            continue
        }

        // 1. 创建文件节点
        fileName := filepath.Base(fullPath)
        parentPath := filepath.Dir(fullPath)

        nodes[fullPath] = &FileNode{
            Path:       fullPath,
            IsFile:     true,
            Name:       fileName,
            ParentPath: parentPath,
            Children:   []string{},
            Depth:      strings.Count(strings.TrimPrefix(fullPath, startPath), "/"),
        }

        // 2. 从文件的直接父目录开始，向上创建目录节点，直到起始路径
        currentParent := parentPath
        for {
            // 检查是否已经到达或超过起始路径
            if !strings.HasPrefix(currentParent, strings.TrimSuffix(startPath, "/")) {
                break
            }

            parentName := filepath.Base(currentParent)

            if _, exists := nodes[currentParent]; !exists {
                var grandParent string
                if currentParent == strings.TrimSuffix(startPath, "/") {
                    grandParent = "" // 起始路径的父路径设为空
                } else {
                    grandParent = filepath.Dir(currentParent)
                }

                nodes[currentParent] = &FileNode{
                    Path:       currentParent,
                    IsFile:     false,
                    Name:       parentName,
                    ParentPath: grandParent,
                    Children:   []string{},
                    Depth:      strings.Count(strings.TrimPrefix(currentParent, startPath), "/"),
                }
            }

            // 3. 添加子节点
            childToAdd := getImmediateChild(currentParent, fullPath)
            childType := "0" // 默认文件
            if childToAdd != fullPath {
                childType = "1" // 目录
            }

            childIdentifier := childToAdd + "=_=" + childType
            if !contains(nodes[currentParent].Children, childIdentifier) {
                nodes[currentParent].Children = append(nodes[currentParent].Children, childIdentifier)
            }

            // 4. 如果已经到达起始路径，停止向上
            if currentParent == strings.TrimSuffix(startPath, "/") {
                break
            }

            // 向上移动
            nextParent := filepath.Dir(currentParent)
            if nextParent == currentParent {
                break
            }
            currentParent = nextParent
        }
    }

    return nodes
}

func contains(slice []string, item string) bool {
    for _, s := range slice {
        if s == item {
            return true
        }
    }
    return false
}
