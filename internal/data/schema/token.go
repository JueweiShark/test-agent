package schema

import (
    "time"
)

type UserToken struct {
    Id        *int64    `gorm:"column:id;primary_key;AUTO_INCREMENT;comment:'Primary key'" mapper:"id"`
    Token     string    `gorm:"column:token;NOT NULL;comment:'token'" mapper:"token"`
    TokenType string    `gorm:"column:token_type;NOT NULL;comment:'token type'" mapper:"tokenType"`
    GroupId   int64     `gorm:"column:group_id;NOT NULL;default:0;comment:'group id (数据隔离: groupId 入 token, 0=无租户上下文/超管)'" mapper:"groupId"`
    ExpireAt  time.Time `gorm:"column:expire_at;comment:'token expire datetime'" mapper:"expireAt"`
    CreatedAt time.Time `gorm:"column:created_at;default:CURRENT_TIMESTAMP;comment:'Create time'" mapper:"createdAt"`
    CreatedBy int64     `gorm:"column:created_by;comment:'Creator id'" mapper:"createdBy"`
    UpdatedAt time.Time `gorm:"column:updated_at;default:CURRENT_TIMESTAMP;comment:'Update time'" mapper:"updatedAt"`
    UpdatedBy int64     `gorm:"column:updated_by;comment:'Modify person id'" mapper:"updatedBy"`
}

func (u *UserToken) TableName() string {
    return "user_token"
}
