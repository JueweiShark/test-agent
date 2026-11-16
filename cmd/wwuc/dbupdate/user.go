package dbupdate

import (
    "context"
    "fmt"

    "github.com/go-kratos/kratos/v2/log"
    "golang.org/x/crypto/bcrypt"

    "wuwen.com/wwuc/internal/data/common"
    "wuwen.com/wwuc/internal/data/schema"
)

var (
    UserUpdate = NewUpdateMethod(&schema.User{},
        "userUpdate",
        map[string]string{},
        map[string]string{},
    )
    user   = NewUser()
    loadSA = NewLoadSA()
)

type LoadSA struct {
    log *log.Helper
}

func NewLoadSA() *LoadSA {
    return &LoadSA{
        log: log.NewHelper(log.With(log.DefaultLogger, "source", "loadSA")),
    }
}
func (l *LoadSA) MethodName() string {
    return "loadSA"
}
func (l *LoadSA) Do(ctx context.Context, data *common.Data) bool {
    if !loadPermissions.Do(ctx, data) {
        l.log.Errorf("load permission data error")
        return false
    }
    err := data.InTx(ctx, func(ctx context.Context) error {
        success := user.Do(ctx, data) && role.Do(ctx, data) && group.Do(ctx, data) && refs.Do(ctx, data)
        if !success {
            return fmt.Errorf("load sa data error")
        }
        return nil
    })
    if err != nil {
        l.log.Errorf("load sa data error:%v", err)
        return false
    }
    return true
}

type User struct {
    log *log.Helper
}

func NewUser() *User {
    return &User{
        log: log.NewHelper(log.With(log.DefaultLogger, "module", "dbupdate.user")),
    }
}
func (uc *User) MethodName() string {
    return "loadUser"
}

func (uc *User) Do(ctx context.Context, data *common.Data) bool {
    users := getUserData()
    err := data.DB(ctx).Model(schema.User{}).Create(users).Error
    if err != nil {
        uc.log.WithContext(ctx).Errorf("create user error:%v", err)
        return false
    }
    uc.log.WithContext(ctx).Infof("create user success,user name :%s", users.Username)
    return true
}

func getUserData() *schema.User {
    bytes, err := bcrypt.GenerateFromPassword([]byte("Admin@wuwen"), bcrypt.DefaultCost)
    if err != nil {
        fmt.Println("GenerateFromPassword：", err)
        return nil
    }
    var id int64 = 1
    return &schema.User{
        Id:       &id,
        Username: "admin@wuwen.ai",
        Password: string(bytes),
        Nickname: "admin",
    }
}
