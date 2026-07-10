package request

// AboutUpdateRequest 更新关于页面内容请求
// 此为部分更新接口，只发送需要修改的字段，未发送的字段保持不变
type AboutUpdateRequest struct {
	TechStack *string `json:"tech_stack,omitempty" binding:"omitempty,max=255"`
	MyStory   *string `json:"my_story,omitempty" binding:"omitempty"`
	Why       *string `json:"why,omitempty" binding:"omitempty"`
	Interest  *string `json:"interest,omitempty" binding:"omitempty,max=255"`
	GitHub    *string `json:"git_hub,omitempty" binding:"omitempty,max=255"`
	CSDN      *string `json:"csdn,omitempty" binding:"omitempty,max=255"`
}
