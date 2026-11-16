package bootstrap

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
	// nacos config
	nacosKratos "github.com/go-kratos/kratos/contrib/config/nacos/v2"
	nacosClients "github.com/nacos-group/nacos-sdk-go/clients"
	nacosConstant "github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/vo"

	"wuwen.com/wwuc/internal/conf"
)

// getGroupId 获取合法的配置名
func getGroupId(configKey string, useBackslash bool) string {
	if useBackslash {
		return strings.Replace(configKey, `.`, `/`, -1)
	} else {
		return configKey
	}
}

// NewRemoteConfigSource 创建一个远程配置源

func NewRemoteConfigSource(config *conf.RemoteConfig) config.Source {
	switch config.GetType() {
	case "nacos":
		return NewNacosConfigSource(config)
	case "local":
		return nil
	}
	return nil
}

// NewNacosConfigSource 创建一个远程配置源 - Nacos

func NewNacosConfigSource(c *conf.RemoteConfig) config.Source {

	uri, _ := url.Parse(c.Host)
	h := strings.Split(uri.Host, ":")
	addr := h[0]
	port, _ := strconv.Atoi(h[1])
	sc := []nacosConstant.ServerConfig{
		*nacosConstant.NewServerConfig(addr, uint64(port)),
	}

	cc := nacosConstant.ClientConfig{

		NamespaceId:          c.GetNamespaceId(),
		Username:             c.GetUserName(),
		Password:             c.GetPwd(),
		TimeoutMs:            10 * 1000, // http请求超时时间，单位毫秒
		BeatInterval:         5 * 1000,  // 心跳间隔时间，单位毫秒
		UpdateThreadNum:      20,        // 更新服务的线程数
		LogLevel:             c.LogLevel,
		CacheDir:             "./configs/cache", // 缓存目录
		LogDir:               "./configs/log",   // 日志目录
		NotLoadCacheAtStart:  true,              // 在启动时不读取本地缓存数据，true--不读取，false--读取
		UpdateCacheWhenEmpty: true,              // 当服务列表为空时是否更新本地缓存，true--更新,false--不更新
	}

	nacosClient, err := nacosClients.NewConfigClient(
		vo.NacosClientParam{
			ClientConfig:  &cc,
			ServerConfigs: sc,
		},
	)
	if err != nil {
		panic(err)
	}

	return nacosKratos.NewConfigSource(nacosClient,
		nacosKratos.WithGroup(getGroupId(c.GetGroupId(), false)),
		nacosKratos.WithDataID(c.GetDataId()),
	)

}

// NewFileConfigSource 创建一个本地文件配置源
func NewFileConfigSource(filePath string) config.Source {
	return file.NewSource(filePath)
}

// NewConfigProvider 创建一个配置
func NewConfigProvider(bootstrap *conf.RemoteConfig, localPath string) config.Config {
	remoteSource := NewRemoteConfigSource(bootstrap)
	if remoteSource != nil {
		return config.New(
			config.WithSource(
				NewFileConfigSource(localPath),
				NewRemoteConfigSource(bootstrap),
			),
		)
	} else {
		return config.New(
			config.WithSource(
				NewFileConfigSource(localPath),
			),
		)
	}

}

func LoadConfig(configFile string) (*conf.Bootstrap, error) {
	c := config.New(
		config.WithSource(
			file.NewSource(configFile),
		),
	)
	defer c.Close()

	if err := c.Load(); err != nil {
		return nil, err
	}

	var bc conf.Bootstrap
	if err := c.Scan(&bc); err != nil {
		return nil, err
	}

	return &bc, nil
}
