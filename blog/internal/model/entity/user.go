package entity

// User 用户实体
type User struct {
	BaseEntity
	// 注意：PostgreSQL 下 int8 的合法列类型是 smallint（没有 tinyint），
	// string 的默认列类型是 text —— PG 允许 text 建唯一索引，
	// 所以这里显式写 varchar(n) 只是为了保留长度约束，不是为了索引。
	UserName     string `gorm:"type:varchar(50);not null;unique;comment:用户名" json:"user_name"`
	Account      string `gorm:"type:varchar(50);not null;unique;comment:用户账号" json:"account"`
	PasswordHash string `gorm:"type:varchar(255);not null;comment:密码哈希" json:"-"`
	Email        string `gorm:"type:varchar(50);comment:邮箱" json:"email"`
	Introduction string `gorm:"type:text" json:"introduction" comment:"用户简介"`
	AvatarURL    string `gorm:"type:varchar(255);comment:头像url" json:"avatar_url"`
	Status       int8   `gorm:"type:smallint;not null;comment:状态:1启用,0禁用" json:"status"`
	RoleID       int8   `gorm:"type:smallint;not null;default:0;comment:角色:1:管理员(admin),0:普通用户(normal)" json:"role_id"`
}

func (User) TableName() string {
	return "users"
}
