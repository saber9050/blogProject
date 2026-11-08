package chat

import (
	"strings"

	"blog/internal/model/dto/request"
	"blog/internal/model/dto/response"
	"blog/internal/model/entity"
	repo "blog/internal/repository/chat"
	"blog/pkg/errors"
	"blog/pkg/logger"

	"go.uber.org/zap"
)

// adminSummaryRunes 后台列表里首问摘要的长度。
//
// 给得太长会把表格撑变形，太短又看不出问的是什么。40 个字大约能覆盖一个完整的技术提问。
const adminSummaryRunes = 40

// AdminList 后台会话列表。
//
// ⚠️ 这里**不按文章 status 过滤**：草稿文章、已删除文章的会话都要能看到。
// 后台是「查发生过什么」，和前台「现在能不能检索」是两件事。
// 代价是列表里可能出现标题为空的会话（文章被硬删），所以带了 ArticleDeleted 标记。
func (s *chatService) AdminList(q *request.AdminChatSessionQuery) (*response.AdminSessionListResponse, error) {
	page, size := q.Page, q.PageSize
	if page <= 0 {
		page = 1
	}
	if size <= 0 || size > 100 {
		size = 20
	}

	rows, total, err := s.repo.AdminListSessions(repo.AdminQuery{
		User:     q.User,
		Article:  q.Article,
		Page:     page,
		PageSize: size,
	})
	if err != nil {
		logger.Error("读取后台会话列表失败", zap.Error(err))
		return nil, errors.NewWithErr(errors.CodeInternalError, "读取会话列表失败", err)
	}

	list := make([]*response.AdminSessionItem, 0, len(rows))
	for _, r := range rows {
		list = append(list, toAdminSessionItem(r))
	}
	return &response.AdminSessionListResponse{
		List:     list,
		Total:    total,
		Page:     page,
		PageSize: size,
	}, nil
}

// AdminDetail 会话详情 + 全部消息。
func (s *chatService) AdminDetail(id uint) (*response.AdminSessionDetailResponse, error) {
	if id == 0 {
		return nil, errors.New(errors.CodeBadRequest, "无效的会话ID")
	}
	row, err := s.repo.AdminFindSession(id)
	if err != nil {
		logger.Error("读取后台会话详情失败", zap.Uint("session_id", id), zap.Error(err))
		return nil, errors.NewWithErr(errors.CodeInternalError, "读取会话失败", err)
	}
	if row == nil {
		return nil, errors.New(errors.CodeNotFound, "会话不存在")
	}

	msgs, err := s.repo.ListAllMessages(id)
	if err != nil {
		logger.Error("读取会话消息失败", zap.Uint("session_id", id), zap.Error(err))
		return nil, errors.NewWithErr(errors.CodeInternalError, "读取消息失败", err)
	}

	// 工具消息一次性全取（详情不分页），再按 message_id 分发。
	// 逐条消息查一次会变成 N+1：一个会话最多 400 条消息，那就是 400 次往返。
	toolsByMsg, err := s.repo.ListToolMessagesBySession(id)
	if err != nil {
		// 工具消息是辅助信息，取不到不该让整个详情打不开
		logger.Warn("读取会话工具消息失败", zap.Uint("session_id", id), zap.Error(err))
	}

	items := make([]*response.AdminMessageItem, 0, len(msgs))
	for _, m := range msgs {
		item := toAdminMessageItem(m)
		for _, t := range toolsByMsg[m.ID] {
			item.ToolMessages = append(item.ToolMessages, toToolMessageItem(t))
		}
		items = append(items, item)
	}
	return &response.AdminSessionDetailResponse{
		Session:  toAdminSessionItem(row),
		Messages: items,
	}, nil
}

// AdminDelete 后台硬删会话。
//
// 与用户自己删是同一套语义（先删消息再删会话、同事务、不可恢复）。
// 不复用 DeleteSession 是因为那个要先做归属校验，而管理员本就不受归属限制。
func (s *chatService) AdminDelete(id uint) error {
	if id == 0 {
		return errors.New(errors.CodeBadRequest, "无效的会话ID")
	}
	if err := s.repo.AdminDeleteSession(id); err != nil {
		if isNotFound(err) {
			return errors.New(errors.CodeNotFound, "会话不存在")
		}
		logger.Error("删除会话失败", zap.Uint("session_id", id), zap.Error(err))
		return errors.NewWithErr(errors.CodeInternalError, "删除会话失败", err)
	}
	// 管理员删了之后，用户端那个 localStorage 里还留着会话 id，
	// 下次打开会 404 然后静默回退——前端已处理，这里不用通知。
	return nil
}

func toAdminSessionItem(r *repo.SessionRow) *response.AdminSessionItem {
	item := &response.AdminSessionItem{
		ID:             r.ID,
		ArticleID:      r.ArticleID,
		ArticleTitle:   r.ArticleTitle,
		ArticleDeleted: r.ArticleDeleted,
		UserID:         r.UserID,
		UserName:       r.UserName,
		Title:          r.Title,
		// 摘要去掉换行，否则表格里一行会撑成好几行
		Summary:      truncateRunes(strings.ReplaceAll(r.FirstUserContent, "\n", " "), adminSummaryRunes),
		MessageCount: r.MessageCount,
		CreatedAt:    r.CreatedAt.Format(timeLayout),
	}
	if r.LastMessageAt != nil {
		t := r.LastMessageAt.Format(timeLayout)
		item.LastMessageAt = &t
	}
	return item
}

func toAdminMessageItem(m *entity.ChatMessage) *response.AdminMessageItem {
	return &response.AdminMessageItem{
		ID:               m.ID,
		Role:             m.Role,
		Content:          m.Content,
		Model:            m.Model,
		Refs:             m.Refs,
		ArticleRefs:      m.ArticleRefs,
		Steps:            m.Steps,
		PromptTokens:     m.PromptTokens,
		CompletionTokens: m.CompletionTokens,
		Interrupted:      m.Interrupted,
		Refused:          m.Refused,
		NoRetrieval:      m.NoRetrieval,
		Error:            m.Error,
		CreatedAt:        m.CreatedAt.Format(timeLayout),
	}
}
