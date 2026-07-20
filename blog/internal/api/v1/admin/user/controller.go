package user

import (
	"blog/internal/model/dto/request"
	response2 "blog/internal/model/dto/response"
	userSvc "blog/internal/service/user"
	"blog/pkg/response"
	"strconv"

	"github.com/gin-gonic/gin"
)

// UserController 后台用户管理控制器
type UserController struct {
	userService userSvc.UserService
}

// NewUserController 创建后台用户管理控制器
func NewUserController(userService userSvc.UserService) *UserController {
	return &UserController{userService: userService}
}

// ListUsers 获取普通用户列表
func (ctrl *UserController) ListUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	list, total, err := ctrl.userService.ListNormalUsers(page, pageSize)
	if err != nil {
		response.BizError(c, err)
		return
	}

	response.Success(c, &response2.PaginatedResponse{
		List:     list,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

// CreateUser 创建用户
func (ctrl *UserController) CreateUser(c *gin.Context) {
	var req request.AdminCreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误")
		return
	}

	id, err := ctrl.userService.AdminCreateUser(&req)
	if err != nil {
		response.BizError(c, err)
		return
	}

	response.Success(c, gin.H{"id": id})
}

// UpdateUser 更新用户状态
func (ctrl *UserController) UpdateUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的用户ID")
		return
	}

	var req request.AdminUpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误")
		return
	}

	if err := ctrl.userService.AdminUpdateStatus(uint(id), req.Status); err != nil {
		response.BizError(c, err)
		return
	}

	response.Success(c, nil)
}

// DeleteUser 删除用户
func (ctrl *UserController) DeleteUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的用户ID")
		return
	}

	if err := ctrl.userService.AdminDeleteUser(uint(id)); err != nil {
		response.BizError(c, err)
		return
	}

	response.Success(c, nil)
}
