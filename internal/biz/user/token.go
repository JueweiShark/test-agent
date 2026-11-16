package user

import (
    "context"
    "errors"
    "strconv"
    "time"

    "github.com/go-kratos/kratos/v2/log"
    "github.com/golang-jwt/jwt/v5"

    "wuwen.com/wwuc/internal/conf"
)

const (
    tokenGenerateTypeGateWay = "GATEWAY"
    tokenGenerateTypeAPI     = "API"
)

// ErrTokenRevoked token 已吊销或不存在 (user_token 表无记录)
var ErrTokenRevoked = errors.New("token revoked or not exist")

type Token struct {
    ID        *int64    `mapper:"ID"`
    Token     string    `mapper:"token"`
    TokenType string    `mapper:"tokenType"`
    GroupId   int64     `mapper:"groupId"`
    ExpireAt  time.Time `mapper:"expireAt"`
    CreatedAt time.Time `mapper:"createdAt"`
    CreatedBy int64     `mapper:"createdBy"`
    UpdatedAt time.Time `mapper:"updatedAt"`
    UpdatedBy int64     `mapper:"updatedBy"`
}
type TokenUsecase struct {
    jwtCfg JwtConfig
    repo   TokenRepo
    log    *log.Helper
}
type JwtConfig struct {
    Secret        string
    Issuer        string
    ExpireHours   int64
    ExpireMinutes int64
}
type TokenRepo interface {
	Create(ctx context.Context, e *Token) error
	First(ctx context.Context, t string) (*Token, error)
	DeleteByUserId(ctx context.Context, userId int64) error
}

func NewTokenUsecase(logger log.Logger, conf *conf.Bootstrap, repo TokenRepo) *TokenUsecase {
    return &TokenUsecase{
        log: log.NewHelper(logger),
        jwtCfg: JwtConfig{
            Secret:        conf.Jwt.Secret,
            Issuer:        conf.Jwt.Issuer,
            ExpireHours:   conf.Jwt.ExpireHours,
            ExpireMinutes: conf.Jwt.ExpireMinutes,
        },
        repo: repo,
    }
}

func (uc *TokenUsecase) FindToken(ctx context.Context, t string) bool {
	_, err := uc.repo.First(ctx, t)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("token not exist")
		return false
	}
	return true
}

// DeleteUserTokens 删除指定用户的全部 token（用于修改密码后强制重新登录）.
func (uc *TokenUsecase) DeleteUserTokens(ctx context.Context, userId int64) error {
	return uc.repo.DeleteByUserId(ctx, userId)
}

// ExpireDuration returns the configured token expiration duration (供 v2 AuthUsecase 计算 expire_time 返回前端).
func (uc *TokenUsecase) ExpireDuration() time.Duration {
	return time.Duration(uc.jwtCfg.ExpireHours)*time.Hour + time.Duration(uc.jwtCfg.ExpireMinutes)*time.Minute
}

const (
	// ClaimKeyGroupId JWT 自定义 claim: 租户 id (数据隔离改造: groupId 入 Token).
	// 0 表示无租户上下文 (超管全局 token / 存量基础 token)
	ClaimKeyGroupId = "groupId"
)

// GroupClaims 携带 groupId 的 JWT claims (数据隔离改造).
// groupId 权威来源由前端请求参数迁移到服务端签发的 token。
type GroupClaims struct {
	jwt.RegisteredClaims
	GroupId int64 `json:"groupId"`
}

// ParsedToken ParseToken 解析结果
type ParsedToken struct {
	UserId int64
	// GroupId 为 token 内 groupId claim; 0 表示无租户上下文
	GroupId int64
	// HasGroupClaim token 是否携带 groupId claim (存量 token 为 false, 灰度期兜底)
	HasGroupClaim bool
}

// GenerateTokenWithGroup 签发携带 groupId claim 的 JWT 并持久化 (含 group_id 用于按组吊销与审计).
// groupId 传 0 表示超管全局 token; 原 GenerateToken 保持不动, 存量签发路径不受影响。
func (uc *TokenUsecase) GenerateTokenWithGroup(ctx context.Context, userId, groupId int64, tokenType string,
	expireTime *time.Time) (token string, err error) {
	if expireTime == nil {
		defaultExpire := time.Now().Add(uc.ExpireDuration())
		expireTime = &defaultExpire
	}
	claims := &GroupClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatInt(userId, 10),
			ExpiresAt: jwt.NewNumericDate(*expireTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    uc.jwtCfg.Issuer,
		},
		GroupId: groupId,
	}

	jToken := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	token, err = jToken.SignedString([]byte(uc.jwtCfg.Secret))
	if err != nil {
		uc.log.WithContext(ctx).Errorf("jwt group token sign error %+v", err)
		return
	}

	err = uc.repo.Create(ctx, &Token{
		Token:     token,
		TokenType: tokenType,
		GroupId:   groupId,
		ExpireAt:  *expireTime,
		CreatedBy: userId,
	})
	if err != nil {
		uc.log.WithContext(ctx).Errorf("jwt group token create error %+v", err)
		return
	}
	return
}

// ParseToken 校验并解析 JWT, 返回 userId 与 groupId claim。
// 1. 验签 (HS512) + 过期校验由 jwt 库完成; 2. 校验 token 是否已被吊销 (user_token 表不存在即无效)。
func (uc *TokenUsecase) ParseToken(ctx context.Context, t string) (*ParsedToken, error) {
	keyFunc := func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(uc.jwtCfg.Secret), nil
	}
	parsed, err := jwt.Parse(t, keyFunc, jwt.WithValidMethods([]string{jwt.SigningMethodHS512.Alg()}))
	if err != nil {
		uc.log.WithContext(ctx).Errorf("parse token failed: %v", err)
		return nil, err
	}
	// 吊销校验: user_token 表中不存在则视为无效
	if !uc.FindToken(ctx, t) {
		return nil, ErrTokenRevoked
	}
	mc, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return nil, jwt.ErrInvalidType
	}
	sub, err := mc.GetSubject()
	if err != nil {
		uc.log.WithContext(ctx).Errorf("parse token subject failed: %v", err)
		return nil, err
	}
	userId, err := strconv.ParseInt(sub, 10, 64)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("parse token subject failed: %v", err)
		return nil, err
	}
	// JSON 数字反序列化为 float64; claim 缺失时 hasGroup=false (存量基础 token)
	var groupId int64
	hasGroup := false
	if v, exists := mc[ClaimKeyGroupId]; exists {
		if f, isNum := v.(float64); isNum {
			groupId = int64(f)
			hasGroup = true
		}
	}
	return &ParsedToken{
		UserId:        userId,
		GroupId:       groupId,
		HasGroupClaim: hasGroup,
	}, nil
}

func (uc *TokenUsecase) GenerateToken(ctx context.Context, userId int64, tokenType string,
    expireTime *time.Time) (token string, err error) {
    if expireTime == nil {
        defaultExpire := time.Now().Add(time.Duration(uc.jwtCfg.ExpireHours)*time.Hour + time.Duration(uc.jwtCfg.ExpireMinutes)*time.Minute)
        expireTime = &defaultExpire
    }
    claims := &jwt.RegisteredClaims{
        Subject:   strconv.FormatInt(userId, 10),
        ExpiresAt: jwt.NewNumericDate(*expireTime),
        IssuedAt:  jwt.NewNumericDate(time.Now()),
        Issuer:    uc.jwtCfg.Issuer,
    }

    jToken := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
    token, err = jToken.SignedString([]byte(uc.jwtCfg.Secret))
    if err != nil {
        log.Info("jwt token sign error", err)
        return
    }

    err = uc.repo.Create(ctx, &Token{
        Token:     token,
        TokenType: tokenType,
        ExpireAt:  *expireTime,
        CreatedBy: userId,
    })
    if err != nil {
        uc.log.WithContext(ctx).Errorf("jwt token create error %+v", err)
        return
    }

    return
}
