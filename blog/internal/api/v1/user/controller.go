package user

import (
	"blog/internal/middleware"
	"blog/internal/model/dto/request"
	userSvc "blog/internal/service/user"
	"blog/pkg/jwt"
	"blog/pkg/response"
	"bytes"
	"html/template"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Controller 用户控制器
type Controller struct {
	userService userSvc.UserService
}

// NewController 创建用户控制器
func NewController(userService userSvc.UserService) *Controller {
	return &Controller{
		userService: userService,
	}
}

// GetUserInfo 获取用户信息
func (c *Controller) GetUserInfo(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)
	if userID == 0 {
		response.Unauthorized(ctx, "请先登录")
		return
	}

	userInfo, err := c.userService.GetUserInfo(userID)
	if err != nil {
		response.BizError(ctx, err)
		return
	}

	response.Success(ctx, userInfo)
}

// UpdateProfile 编辑用户信息
func (c *Controller) UpdateProfile(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)
	if userID == 0 {
		response.Unauthorized(ctx, "请先登录")
		return
	}

	var req request.UpdateUserProfileRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.BadRequest(ctx, "请求参数错误")
		return
	}

	if err := c.userService.UpdateProfile(userID, &req); err != nil {
		response.BizError(ctx, err)
		return
	}

	response.Success(ctx, "更新成功")
}

// UpdateAvatar 更换头像
func (c *Controller) UpdateAvatar(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)
	if userID == 0 {
		response.Unauthorized(ctx, "请先登录")
		return
	}

	file, err := ctx.FormFile("file")
	if err != nil {
		response.BadRequest(ctx, "请上传头像文件")
		return
	}

	userResponse, err := c.userService.UpdateAvatar(userID, file)
	if err != nil {
		response.BizError(ctx, err)
		return
	}

	response.Success(ctx, userResponse)
}

// UpdateEmailRequest 更换邮箱的确认请求
func (c *Controller) UpdateEmailRequest(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)
	if userID == 0 {
		response.Unauthorized(ctx, "请先登录")
		return
	}

	var req request.UpdateUserEmailRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.BadRequest(ctx, "请求参数错误")
		return
	}

	userResponse, err := c.userService.UpdateEmail(userID, &req)
	if err != nil {
		response.BizError(ctx, err)
		return
	}

	response.Success(ctx, userResponse)
}

// UpdateEmail 修改邮箱
func (c *Controller) UpdateEmail(ctx *gin.Context) {
	token := ctx.Query("token")
	// 基础参数校验
	if token == "" {
		renderResultHTML(ctx, false, "邮箱修改失败", "缺少必要的确认信息，请重新发送确认邮件")
		return
	}
	res, err := jwt.ParseToken(token)
	if err != nil {
		renderResultHTML(ctx, false, "邮箱修改失败", "确认链接无效或已过期，请重新申请修改")
		return
	}
	//	修改邮箱并把当前token加入黑名单
	err = c.userService.UpdateAdminEmail(res.UserID, token, res.Username)
	if err != nil {
		renderResultHTML(ctx, false, "邮箱修改失败", "确认链接无效或已过期，请重新申请修改")
		return
	}
	renderResultHTML(ctx, true, "邮箱修改成功", "您已成功修改绑定邮箱，下次登录可以使用新邮箱")
}

// renderResultHTML 渲染统一的结果页面
func renderResultHTML(c *gin.Context, success bool, title, message string) {
	// 定义内嵌模板（含样式与结构）
	const tmplStr = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0, user-scalable=no">
    <title>{{.Title}} - 个人博客</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
            background: #f5f7fa;
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 20px;
        }
        .card {
            max-width: 480px;
            width: 100%;
            background: #ffffff;
            border-radius: 20px;
            box-shadow: 0 10px 25px -5px rgba(0, 0, 0, 0.08), 0 8px 10px -6px rgba(0, 0, 0, 0.02);
            overflow: hidden;
            text-align: center;
            padding: 40px 32px;
            transition: transform 0.2s ease;
        }
        .icon {
            font-size: 64px;
            margin-bottom: 24px;
        }
        .success-icon {
            color: #10b981;
        }
        .error-icon {
            color: #ef4444;
        }
        h2 {
            font-size: 28px;
            font-weight: 600;
            margin-bottom: 16px;
            color: #1f2937;
        }
        .message {
            font-size: 16px;
            color: #4b5563;
            line-height: 1.5;
            margin-bottom: 32px;
            word-break: break-word;
        }
        .btn {
            display: inline-block;
            background: #3b82f6;
            color: white;
            text-decoration: none;
            padding: 12px 28px;
            border-radius: 40px;
            font-weight: 500;
            font-size: 15px;
            transition: background 0.2s;
            border: none;
            cursor: pointer;
        }
        .btn:hover {
            background: #2563eb;
        }
        .btn-secondary {
            background: #9ca3af;
        }
        .btn-secondary:hover {
            background: #6b7280;
        }
        .footer {
            margin-top: 28px;
            font-size: 13px;
            color: #9ca3af;
        }
        @media (max-width: 480px) {
            .card {
                padding: 32px 24px;
            }
            h2 {
                font-size: 24px;
            }
            .icon {
                font-size: 52px;
            }
        }
    </style>
</head>
<body>
    <div class="card">
        <div class="icon {{if .Success}}success-icon{{else}}error-icon{{end}}">
            {{if .Success}}✓{{else}}✗{{end}}
        </div>
        <h2>{{.Title}}</h2>
        <div class="message">{{.Message}}</div>
        <div class="footer">核心力量团队 · 安全验证</div>
    </div>
</body>
</html>`

	// 准备模板数据
	data := struct {
		Success bool
		Title   string
		Message string
	}{
		Success: success,
		Title:   title,
		Message: message,
	}

	// 解析并执行模板
	tmpl, err := template.New("result").Parse(tmplStr)
	if err != nil {
		// 模板解析失败时降级为纯文本
		c.String(http.StatusInternalServerError, "系统错误，请稍后重试")
		return
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		c.String(http.StatusInternalServerError, "渲染失败，请稍后重试")
		return
	}
	c.Data(http.StatusOK, "text/html; charset=utf-8", buf.Bytes())
}
