package dbupdate

import (
	"context"
	"strconv"
	"strings"

	"github.com/go-kratos/kratos/v2/log"
	"golang.org/x/crypto/bcrypt"

	"wuwen.com/wwuc/internal/data/common"
	"wuwen.com/wwuc/internal/data/schema"
)

var (
    createAdmin = NewCreateAdminUser()
)

func CreateAdminInstance() *CreateAdminUser {
    return createAdmin
}

func extractNickname(username string) string {
    if idx := strings.Index(username, "@"); idx != -1 {
        return username[:idx]
    }
    return username
}

type CreateAdminUser struct {
    log        *log.Helper
    Username   string
    Password   string
    Nickname   string
    AppId      int64
    RoleId     int64
    GroupId    int64
}

func NewCreateAdminUser() *CreateAdminUser {
    return &CreateAdminUser{
        log: log.NewHelper(log.With(log.DefaultLogger, "module", "dbupdate.createAdmin")),
    }
}

func (c *CreateAdminUser) MethodName() string {
    return "createAdmin"
}

func (c *CreateAdminUser) SetParams(params string) bool {
    if params == "" {
        c.log.Error("params is required for createAdmin method")
        return false
    }

    c.RoleId = 0

    pairs := strings.Split(params, "&")
    for _, pair := range pairs {
        kv := strings.Split(pair, "=")
        if len(kv) != 2 {
            continue
        }
        key := strings.TrimSpace(kv[0])
        value := strings.TrimSpace(kv[1])

        switch key {
        case "username":
            c.Username = value
        case "password":
            c.Password = value
        case "nickname":
            c.Nickname = value
        case "appId":
            if v, err := strconv.ParseInt(value, 10, 64); err == nil {
                c.AppId = v
            }
        case "roleId":
            if v, err := strconv.ParseInt(value, 10, 64); err == nil {
                c.RoleId = v
            }
        case "groupId":
            if v, err := strconv.ParseInt(value, 10, 64); err == nil {
                c.GroupId = v
            }
        }
    }

    if c.Username == "" {
        c.log.Error("username is required")
        return false
    }
    if c.Password == "" {
        c.log.Error("password is required")
        return false
    }
    if c.RoleId <= 0 {
        c.log.Error("roleId is required (1-admin, 2-orgAdmin)")
        return false
    }
    if c.GroupId < 0 {
        c.log.Error("groupId must be >= 0")
        return false
    }
    return true
}

func (c *CreateAdminUser) Do(ctx context.Context, data *common.Data) bool {
    if c.Username == "" || c.Password == "" {
        c.log.Error("username and password are required")
        return false
    }

    bytes, err := bcrypt.GenerateFromPassword([]byte(c.Password), bcrypt.DefaultCost)
    if err != nil {
        c.log.Errorf("GenerateFromPassword error: %v", err)
        return false
    }

    nickname := c.Nickname
    if nickname == "" {
        nickname = extractNickname(c.Username)
    }

    user := &schema.User{
        Username: string(c.Username),
        Password: string(bytes),
        Nickname: nickname,
        Provider: "wuwen",
        App:      c.AppId,
        Status:   1,
        CreatedBy: 0,
    }

    err = data.DB(ctx).Model(schema.User{}).Create(user).Error
    if err != nil {
        c.log.Errorf("create user error: %v", err)
        return false
    }
    c.log.Infof("create admin user success, username: %s, appId: %d", c.Username, c.AppId)

    if c.RoleId > 0 && user.Id != nil {
        ref := &schema.RefUserRole{
            UserId:  *user.Id,
            RoleId:  c.RoleId,
            GroupId: c.GroupId,
            CreatedBy: 0,
        }
        err = data.DB(ctx).Model(schema.RefUserRole{}).Create(ref).Error
        if err != nil {
            c.log.Errorf("bind user role error: %v", err)
            return false
        }
        c.log.Infof("bind role success, userId: %d, roleId: %d, groupId: %d", *user.Id, c.RoleId, c.GroupId)
    }

    return true
}
