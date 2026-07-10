package entity

// About 关于实体
type About struct {
	BaseEntity
	TechStack string `gorm:"type:varchar(255);comment:技术栈，每个用逗号隔开" json:"tech_stack"`
	MyStory   string `gorm:"type:text;comment:我的故事" json:"my_story"`
	Why       string `gorm:"type:text;comment:为什么建立博客" json:"why"`
	Interest  string `gorm:"type:varchar(255);comment:感兴趣的技术，每个用逗号隔开" json:"interest"`
	GitHub    string `gorm:"type:varchar(255);comment:github地址" json:"git_hub"`
	CSDN      string `gorm:"type:varchar(255);comment:csdn地址" json:"csdn"`
}

func (About) TableName() string {
	return "abouts"
}
