package chat

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"blog/internal/middleware"
	"blog/internal/model/dto/request"
	"blog/internal/model/dto/response"
	chatSvc "blog/internal/service/chat"

	"github.com/gin-gonic/gin"
)

// fakeService 只用于路由层测试，方法全部返回零值。
// 这些用例不该走到业务逻辑——它们验的是「路由能不能注册」「有没有挂鉴权」。
type fakeService struct{}

func (f *fakeService) Context(userID, articleID uint) (*response.ChatContextResponse, error) {
	return &response.ChatContextResponse{}, nil
}
func (f *fakeService) CreateSession(userID uint, req *request.CreateChatSessionRequest) (*response.ChatSessionItem, error) {
	return &response.ChatSessionItem{}, nil
}
func (f *fakeService) ListMessages(userID, sessionID, beforeID uint, limit int) (*response.ChatMessageListResponse, error) {
	return &response.ChatMessageListResponse{}, nil
}
func (f *fakeService) UpdateSession(userID, sessionID uint, req *request.UpdateChatSessionRequest) error {
	return nil
}
func (f *fakeService) DeleteSession(userID, sessionID uint) error { return nil }
func (f *fakeService) ListArticleCandidates(keyword string, excludeID uint, limit int) ([]*response.ChatArticleCandidateItem, error) {
	return nil, nil
}
func (f *fakeService) AdminList(q *request.AdminChatSessionQuery) (*response.AdminSessionListResponse, error) {
	return &response.AdminSessionListResponse{}, nil
}
func (f *fakeService) AdminDetail(id uint) (*response.AdminSessionDetailResponse, error) {
	return &response.AdminSessionDetailResponse{}, nil
}
func (f *fakeService) AdminDelete(id uint) error { return nil }
func (f *fakeService) ListToolMessages(userID, sessionID, messageID uint) ([]*response.ChatToolMessageItem, error) {
	return nil, nil
}

func (f *fakeService) SendMessage(ctx context.Context, userID, sessionID uint,
	req *request.SendChatMessageRequest, sink chatSvc.StreamSink) (*response.SendChatMessageResponse, error) {
	return &response.SendChatMessageResponse{}, nil
}

var _ chatSvc.Service = (*fakeService)(nil)

// TestChatRoutesRegisteredAndGuarded 验证两件事：
//
//  1. 新路由能和现有 /articles/:id 共存 —— gin 在通配符冲突时是**直接 panic**，
//     这个 panic 发生在启动期，若没有测试兜住，只能靠人手工启动一次才发现。
//  2. 每个接口都挂了 Auth() —— 未登录必须 401。
//
// 第 2 条是刻意用「无 token」来验的：如果哪天有人把 Auth() 换成 OptionalAuth()，
// 请求会带着 user_id=0 继续走到 service，测试就会从 401 变成 200，立刻红。
func TestChatRoutesRegisteredAndGuarded(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// cache 传 nil：无 token 时鉴权中间件在解析阶段就返回错误，不会摸到缓存。
	middleware.InitAuth(nil)

	r := gin.New()
	v1 := r.Group("/api/v1")
	// 把现有 /articles 树的路由形状照搬一遍，验证参数段与静态段能共存。
	// gin 遇到冲突是**启动期 panic**，没有这组模拟就只能在真实启动时才发现。
	noop := func(c *gin.Context) { c.Status(http.StatusOK) }
	v1.GET("/articles", noop)
	v1.GET("/articles/random", noop)
	v1.GET("/articles/stats", noop)
	v1.GET("/articles/:id", noop)
	v1.POST("/articles/:id/like", noop)
	v1.GET("/articles/:id/comments", noop)
	v1.GET("/articles/:id/comments/:commentId/replies", noop)

	NewController(&fakeService{}).RegisterRoutes(v1)

	cases := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/articles/12/chat/context"},
		{http.MethodPost, "/api/v1/chat/sessions"},
		{http.MethodGet, "/api/v1/chat/sessions/1/messages"},
		{http.MethodPost, "/api/v1/chat/sessions/1/messages"},
		{http.MethodPatch, "/api/v1/chat/sessions/1"},
		{http.MethodDelete, "/api/v1/chat/sessions/1"},
		{http.MethodGet, "/api/v1/chat/article-candidates"},
	}

	for _, tc := range cases {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(tc.method, tc.path, nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("%s %s 未带 token 应为 401，实际 %d；"+
				"若变成 200 说明鉴权中间件被换成了 OptionalAuth（匿名请求会全部落到 user_id=0）",
				tc.method, tc.path, w.Code)
		}
	}
}
