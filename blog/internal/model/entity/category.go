package entity

// Category 分类实体
type Category struct {
	BaseEntity
	CategoryName string `gorm:"varchar(50);not null;comment:分类名" json:"category_name"`
	Status       int8   `gorm:"type:tinyint;not null;default:1;comment:状态:1启用,0禁用" json:"status"`
}

func (Category) TableName() string {
	return "categorys"
}
