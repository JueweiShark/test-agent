package user

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	jwtv5 "github.com/golang-jwt/jwt/v5"

	"wuwen.com/wwuc/internal/conf"
)

// fakeTokenRepo 内存版 TokenRepo, 模拟 user_token 表
type fakeTokenRepo struct {
	tokens map[string]*Token
}

func newFakeTokenRepo() *fakeTokenRepo {
	return &fakeTokenRepo{tokens: make(map[string]*Token)}
}

func (f *fakeTokenRepo) Create(ctx context.Context, e *Token) error {
	cp := *e
	f.tokens[e.Token] = &cp
	return nil
}

func (f *fakeTokenRepo) First(ctx context.Context, t string) (*Token, error) {
	if tok, ok := f.tokens[t]; ok {
		return tok, nil
	}
	return nil, errors.New("record not found")
}

func (f *fakeTokenRepo) DeleteByUserId(ctx context.Context, userId int64) error {
	for k, tok := range f.tokens {
		if tok.CreatedBy == userId {
			delete(f.tokens, k)
		}
	}
	return nil
}

func newTestTokenUsecase(repo TokenRepo, expireHours int64) *TokenUsecase {
	return NewTokenUsecase(log.DefaultLogger, &conf.Bootstrap{
		Jwt: &conf.Jwt{
			Secret:      "test-secret",
			Issuer:      "WuWen",
			ExpireHours: expireHours,
		},
	}, repo)
}

// TestGenerateTokenWithGroup_ClaimsRoundtrip 换发组 token: claims 应包含 userId+groupId, 解析可还原
func TestGenerateTokenWithGroup_ClaimsRoundtrip(t *testing.T) {
	repo := newFakeTokenRepo()
	uc := newTestTokenUsecase(repo, 2400)
	ctx := context.Background()

	token, err := uc.GenerateTokenWithGroup(ctx, 1001, 8001, "GATEWAY", nil)
	if err != nil {
		t.Fatalf("GenerateTokenWithGroup failed: %v", err)
	}

	parsed, err := uc.ParseToken(ctx, token)
	if err != nil {
		t.Fatalf("ParseToken failed: %v", err)
	}
	if parsed.UserId != 1001 {
		t.Errorf("userId = %d, want 1001", parsed.UserId)
	}
	if parsed.GroupId != 8001 {
		t.Errorf("groupId = %d, want 8001", parsed.GroupId)
	}
	if !parsed.HasGroupClaim {
		t.Error("HasGroupClaim = false, want true")
	}

	// user_token 表应写入含 group_id 的记录 (按组吊销/审计粒度)
	rec, ok := repo.tokens[token]
	if !ok {
		t.Fatal("group token not persisted into user_token repo")
	}
	if rec.GroupId != 8001 {
		t.Errorf("repo record groupId = %d, want 8001", rec.GroupId)
	}
	if rec.CreatedBy != 1001 {
		t.Errorf("repo record createdBy = %d, want 1001", rec.CreatedBy)
	}
	if rec.TokenType != "GATEWAY" {
		t.Errorf("repo record tokenType = %s, want GATEWAY", rec.TokenType)
	}
}

// TestGenerateTokenWithGroup_SuperAdminGlobalToken 超管分支: groupId=0 签发全局 token
func TestGenerateTokenWithGroup_SuperAdminGlobalToken(t *testing.T) {
	repo := newFakeTokenRepo()
	uc := newTestTokenUsecase(repo, 2400)
	ctx := context.Background()

	token, err := uc.GenerateTokenWithGroup(ctx, 1, 0, "GATEWAY", nil)
	if err != nil {
		t.Fatalf("GenerateTokenWithGroup failed: %v", err)
	}
	parsed, err := uc.ParseToken(ctx, token)
	if err != nil {
		t.Fatalf("ParseToken failed: %v", err)
	}
	if parsed.GroupId != 0 {
		t.Errorf("super admin token groupId = %d, want 0", parsed.GroupId)
	}
	if !parsed.HasGroupClaim {
		t.Error("super admin token should still carry explicit groupId=0 claim")
	}
}

// TestParseToken_BaseTokenNoGroupClaim 存量基础 token (GenerateToken 签发): 无 groupId claim, 兜底为 0
func TestParseToken_BaseTokenNoGroupClaim(t *testing.T) {
	repo := newFakeTokenRepo()
	uc := newTestTokenUsecase(repo, 2400)
	ctx := context.Background()

	baseToken, err := uc.GenerateToken(ctx, 2002, "GATEWAY", nil)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}
	parsed, err := uc.ParseToken(ctx, baseToken)
	if err != nil {
		t.Fatalf("ParseToken failed: %v", err)
	}
	if parsed.UserId != 2002 {
		t.Errorf("userId = %d, want 2002", parsed.UserId)
	}
	if parsed.GroupId != 0 {
		t.Errorf("legacy token groupId = %d, want 0", parsed.GroupId)
	}
	if parsed.HasGroupClaim {
		t.Error("legacy token HasGroupClaim = true, want false")
	}
	// 存量 token 的 user_token 记录 group_id 应为 0
	if rec := repo.tokens[baseToken]; rec.GroupId != 0 {
		t.Errorf("legacy repo record groupId = %d, want 0", rec.GroupId)
	}
}

// TestParseToken_WrongSecret 验签失败: secret 不匹配必须拒绝
func TestParseToken_WrongSecret(t *testing.T) {
	repo := newFakeTokenRepo()
	uc := newTestTokenUsecase(repo, 2400)
	other := NewTokenUsecase(log.DefaultLogger, &conf.Bootstrap{
		Jwt: &conf.Jwt{Secret: "other-secret", Issuer: "WuWen", ExpireHours: 1},
	}, repo)
	ctx := context.Background()

	token, err := uc.GenerateTokenWithGroup(ctx, 1001, 8001, "GATEWAY", nil)
	if err != nil {
		t.Fatalf("GenerateTokenWithGroup failed: %v", err)
	}
	if _, err := other.ParseToken(ctx, token); err == nil {
		t.Error("ParseToken with wrong secret should fail")
	}
}

// TestParseToken_TamperedSignature 篡改 payload 后验签必须失败 (防越权)
func TestParseToken_TamperedSignature(t *testing.T) {
	repo := newFakeTokenRepo()
	uc := newTestTokenUsecase(repo, 2400)
	ctx := context.Background()

	token, err := uc.GenerateTokenWithGroup(ctx, 1001, 8001, "GATEWAY", nil)
	if err != nil {
		t.Fatalf("GenerateTokenWithGroup failed: %v", err)
	}
	tampered := token + "x"
	if _, err := uc.ParseToken(ctx, tampered); err == nil {
		t.Error("ParseToken with tampered token should fail")
	}
}

// TestParseToken_TokenRevoked 吊销校验: user_token 表无记录的 token 必须拒绝
func TestParseToken_TokenRevoked(t *testing.T) {
	repo := newFakeTokenRepo()
	uc := newTestTokenUsecase(repo, 2400)
	ctx := context.Background()

	token, err := uc.GenerateTokenWithGroup(ctx, 1001, 8001, "GATEWAY", nil)
	if err != nil {
		t.Fatalf("GenerateTokenWithGroup failed: %v", err)
	}
	// 模拟切换分组后旧 token 被吊销
	if err := repo.DeleteByUserId(ctx, 1001); err != nil {
		t.Fatalf("DeleteByUserId failed: %v", err)
	}
	if _, err := uc.ParseToken(ctx, token); !errors.Is(err, ErrTokenRevoked) {
		t.Errorf("revoked token error = %v, want ErrTokenRevoked", err)
	}
}

// TestParseToken_Expired 过期 token 必须拒绝
func TestParseToken_Expired(t *testing.T) {
	repo := newFakeTokenRepo()
	uc := newTestTokenUsecase(repo, 2400)
	ctx := context.Background()

	expired := time.Now().Add(-1 * time.Hour)
	token, err := uc.GenerateTokenWithGroup(ctx, 1001, 8001, "GATEWAY", &expired)
	if err != nil {
		t.Fatalf("GenerateTokenWithGroup failed: %v", err)
	}
	if _, err := uc.ParseToken(ctx, token); err == nil {
		t.Error("expired token should fail")
	}
}

// TestParseToken_UnexpectedSigningMethod 非 HS512 签名算法必须拒绝 (算法混淆攻击防护)
func TestParseToken_UnexpectedSigningMethod(t *testing.T) {
	repo := newFakeTokenRepo()
	uc := newTestTokenUsecase(repo, 2400)
	ctx := context.Background()

	claims := &GroupClaims{
		RegisteredClaims: jwtv5.RegisteredClaims{
			Subject:   "1001",
			ExpiresAt: jwtv5.NewNumericDate(time.Now().Add(time.Hour)),
		},
		GroupId: 8001,
	}
	// 用 none/其他算法签发
	unsigned := jwtv5.NewWithClaims(jwtv5.SigningMethodNone, claims)
	token, err := unsigned.SignedString(jwtv5.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("sign none token failed: %v", err)
	}
	if _, err := uc.ParseToken(ctx, token); err == nil {
		t.Error("none-algorithm token should be rejected")
	}
}

// TestExpireDuration 过期时间配置换算
func TestExpireDuration(t *testing.T) {
	uc := newTestTokenUsecase(newFakeTokenRepo(), 2)
	uc2 := NewTokenUsecase(log.DefaultLogger, &conf.Bootstrap{
		Jwt: &conf.Jwt{Secret: "s", Issuer: "WuWen", ExpireMinutes: 30},
	}, newFakeTokenRepo())

	if got := uc.ExpireDuration(); got != 2*time.Hour {
		t.Errorf("ExpireDuration = %v, want 2h", got)
	}
	if got := uc2.ExpireDuration(); got != 30*time.Minute {
		t.Errorf("ExpireDuration = %v, want 30m", got)
	}
}
