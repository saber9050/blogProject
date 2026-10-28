<template>
  <div class="admin-page">
    <NavBar />
    <div class="admin-layout">
      <!-- 侧边栏 -->
      <aside class="admin-sidebar">
        <div class="admin-sidebar__title">
          <img src="/logo-mark.svg" alt="" class="admin-sidebar__logo" />
          <span>后台管理</span>
        </div>
        <button
          v-for="tab in tabs"
          :key="tab.key"
          class="admin-sidebar__link"
          :class="{ 'admin-sidebar__link--active': activeTab === tab.key }"
          @click="activeTab = tab.key"
        >
          <span>{{ tab.icon }}</span>
          <span>{{ tab.label }}</span>
        </button>
      </aside>

      <!-- 主内容区 -->
      <main class="admin-main">
        <!-- ========== 用户管理 ========== -->
        <section v-if="activeTab === 'users'" class="admin-panel">
          <div class="admin-panel__header">
            <h2 class="admin-panel__title">用户管理</h2>
            <div class="admin-panel__actions">
              <button class="btn-sm btn-sm--primary" @click="openModal('user')">+ 新增</button>
            </div>
          </div>
          <div class="filter-bar">
            <input v-model="userFilter.keyword" class="filter-input" placeholder="昵称" @keyup.enter="loadUsers(1)" />
            <select v-model="userFilter.status" class="filter-select" @change="loadUsers(1)">
              <option value="">全部状态</option>
              <option value="1">正常</option>
              <option value="0">封禁</option>
            </select>
            <input type="date" v-model="userFilter.startTime" class="filter-date" @change="loadUsers(1)" title="注册开始" />
            <span class="filter-date-sep">~</span>
            <input type="date" v-model="userFilter.endTime" class="filter-date" @change="loadUsers(1)" title="注册结束" />
            <button class="btn-sm" @click="resetFilter(userFilter); loadUsers(1)">重置</button>
          </div>
          <table class="admin-table">
            <thead>
              <tr>
                <th>ID</th>
                <th>昵称</th>
                <th>头像</th>
                <th>状态</th>
                <th>注册时间</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="u in users" :key="u.id">
                <td>{{ u.id }}</td>
                <td>{{ u.user_name }}</td>
                <td>
                  <img v-if="u.avatar_url" :src="u.avatar_url" class="table-avatar" />
                  <span v-else class="table-avatar table-avatar--placeholder">&#128100;</span>
                </td>
                <td>{{ u.status === 1 ? '正常' : '封禁' }}</td>
                <td>{{ fmt(u.created_at) }}</td>
                <td>
                  <div class="table-actions">
                    <button class="btn-sm" @click="openModal('user', u)">编辑</button>
                    <button class="btn-sm btn-sm--danger" @click="handleDelete('user', u.id)">删除</button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
          <div v-if="!users.length" class="admin-empty">暂无用户数据</div>
          <div v-if="users.length" class="pagination">
            <button class="pagination__btn" :disabled="userPage <= 1" @click="loadUsers(userPage - 1)">上一页</button>
            <template v-for="p in getPageRange(userPage, calcPages(userTotal, userPageSize))" :key="p">
              <span v-if="p === '...'" class="pagination__ellipsis">...</span>
              <button v-else class="pagination__btn" :class="{ 'pagination__btn--active': p === userPage }" @click="loadUsers(p)">{{ p }}</button>
            </template>
            <button class="pagination__btn" :disabled="userPage >= calcPages(userTotal, userPageSize)" @click="loadUsers(userPage + 1)">下一页</button>
            <span class="pagination__info">共 {{ userTotal }} 条</span>
            <select class="pagination__size" v-model.number="userPageSize" @change="loadUsers(1)">
              <option :value="10">10条/页</option>
              <option :value="20">20条/页</option>
              <option :value="50">50条/页</option>
            </select>
          </div>
        </section>

        <!-- ========== 文章管理 ========== -->
        <section v-if="activeTab === 'articles'" class="admin-panel">
          <div class="admin-panel__header">
            <h2 class="admin-panel__title">文章管理</h2>
            <div class="admin-panel__actions">
              <button class="btn-sm" @click="transferModalVisible = true">一键转移</button>
              <button class="btn-sm btn-sm--primary" @click="goToArticleEditor('/admin/article/new')">+ 新增</button>
            </div>
          </div>
          <div class="filter-bar">
            <input v-model="articleFilter.keyword" class="filter-input" placeholder="文章标题" @keyup.enter="loadArticles(1)" />
            <select v-model="articleFilter.categoryId" class="filter-select" @change="loadArticles(1)">
              <option value="">全部分类</option>
              <option v-for="c in adminCategories" :key="c.id" :value="c.id">{{ c.name }}</option>
            </select>
            <select v-model="articleFilter.status" class="filter-select" @change="loadArticles(1)">
              <option value="">全部状态</option>
              <option value="1">已发布</option>
              <option value="0">草稿</option>
            </select>
            <button class="btn-sm" @click="resetFilter(articleFilter); loadArticles(1)">重置</button>
          </div>
          <table class="admin-table">
            <thead>
              <tr>
                <th>ID</th>
                <th>文章标题</th>
                <th>分类</th>
                <th>标签</th>
                <th>状态</th>
                <th>向量化</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="a in adminArticles" :key="a.id">
                <td>{{ a.id }}</td>
                <td>{{ a.title }}</td>
                <td>
                  <span class="cat-tag-badge">{{ a.category?.name || '-' }}</span>
                </td>
                <td>
                  <template v-if="a.tags && a.tags.length">
                    <span v-for="t in a.tags" :key="t.id" class="cat-tag-badge cat-tag-badge--tag">{{ t.name }}</span>
                  </template>
                  <span v-else>-</span>
                </td>
                <td>
                  <span class="status-badge" :class="a.status === 1 ? 'status-badge--normal' : 'status-badge--draft'">
                    {{ a.status === 1 ? '已发布' : '草稿' }}
                  </span>
                </td>
                <td>
                  <!-- vector_state 返回文案与样式类；失败时 hover 显示原因 -->
                  <span
                    class="status-badge"
                    :class="vectorState(a).cls"
                    :title="a.vector_status === 4 ? a.vector_error : vectorState(a).tip"
                  >
                    {{ vectorState(a).text }}
                  </span>
                </td>
                <td>
                  <div class="table-actions">
                    <button
                      class="btn-sm"
                      :disabled="a.status !== 1 || a.vector_status === 1 || a.vector_status === 2 || vectorBusyId === a.id"
                      :title="a.status !== 1 ? '草稿不允许向量化' : ''"
                      @click="openVectorModal(a)"
                    >
                      {{ a.vector_status === 1 || a.vector_status === 2 ? '处理中' : '向量化' }}
                    </button>
                    <button class="btn-sm" @click="goToArticleEditor('/admin/article/' + a.id + '/edit')">编辑</button>
                    <button class="btn-sm btn-sm--danger" @click="handleDelete('article', a.id)">删除</button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
          <div v-if="!adminArticles.length" class="admin-empty">暂无文章数据</div>
          <div v-if="adminArticles.length" class="pagination">
            <button class="pagination__btn" :disabled="articlePage <= 1" @click="loadArticles(articlePage - 1)">上一页</button>
            <template v-for="p in getPageRange(articlePage, calcPages(articleTotal, articlePageSize))" :key="p">
              <span v-if="p === '...'" class="pagination__ellipsis">...</span>
              <button v-else class="pagination__btn" :class="{ 'pagination__btn--active': p === articlePage }" @click="loadArticles(p)">{{ p }}</button>
            </template>
            <button class="pagination__btn" :disabled="articlePage >= calcPages(articleTotal, articlePageSize)" @click="loadArticles(articlePage + 1)">下一页</button>
            <span class="pagination__info">共 {{ articleTotal }} 条</span>
            <select class="pagination__size" v-model.number="articlePageSize" @change="loadArticles(1)">
              <option :value="10">10条/页</option>
              <option :value="20">20条/页</option>
              <option :value="50">50条/页</option>
            </select>
          </div>
        </section>

        <!-- ========== 评论管理 ========== -->
        <section v-if="activeTab === 'comments'" class="admin-panel">
          <div class="admin-panel__header">
            <h2 class="admin-panel__title">评论管理</h2>
            <div class="admin-panel__actions">
              <button v-if="!batchMode" class="btn-sm btn-sm--danger" @click="batchMode = true">批量删除</button>
              <template v-else>
                <button class="btn-sm" @click="exitBatchMode">取消</button>
                <button class="btn-sm btn-sm--danger" :disabled="!selectedCommentIds.length" @click="handleBatchDeleteComments">确认删除 ({{ selectedCommentIds.length }})</button>
              </template>
            </div>
          </div>
          <div class="filter-bar">
            <input v-model="commentFilter.articleTitle" class="filter-input" placeholder="所属文章" @keyup.enter="loadComments(1)" />
            <input v-model="commentFilter.userName" class="filter-input" placeholder="评论者" @keyup.enter="loadComments(1)" />
            <input type="date" v-model="commentFilter.startTime" class="filter-date" @change="loadComments(1)" title="开始时间" />
            <span class="filter-date-sep">~</span>
            <input type="date" v-model="commentFilter.endTime" class="filter-date" @change="loadComments(1)" title="结束时间" />
            <button class="btn-sm" @click="resetFilter(commentFilter); loadComments(1)">重置</button>
          </div>
          <table class="admin-table">
            <thead>
              <tr>
                <th v-if="batchMode"><input type="checkbox" :checked="allCommentsSelected" @change="toggleAllComments" /></th>
                <th>ID</th>
                <th>所属文章</th>
                <th>评论者</th>
                <th>评论内容</th>
                <th>时间</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="c in adminComments" :key="c.id">
                <td v-if="batchMode"><input type="checkbox" :value="c.id" v-model="selectedCommentIds" /></td>
                <td>{{ c.id }}</td>
                <td>{{ c.article_title }}</td>
                <td>{{ c.user_name }}</td>
                <td>{{ c.content }}</td>
                <td>{{ fmt(c.created_at) }}</td>
                <td>
                  <button class="btn-sm btn-sm--danger" @click="handleDelete('comment', c.id)">删除</button>
                </td>
              </tr>
            </tbody>
          </table>
          <div v-if="!adminComments.length" class="admin-empty">暂无评论数据</div>
          <div v-if="adminComments.length" class="pagination">
            <button class="pagination__btn" :disabled="commentPage <= 1" @click="loadComments(commentPage - 1)">上一页</button>
            <template v-for="p in getPageRange(commentPage, calcPages(commentTotal, commentPageSize))" :key="p">
              <span v-if="p === '...'" class="pagination__ellipsis">...</span>
              <button v-else class="pagination__btn" :class="{ 'pagination__btn--active': p === commentPage }" @click="loadComments(p)">{{ p }}</button>
            </template>
            <button class="pagination__btn" :disabled="commentPage >= calcPages(commentTotal, commentPageSize)" @click="loadComments(commentPage + 1)">下一页</button>
            <span class="pagination__info">共 {{ commentTotal }} 条</span>
            <select class="pagination__size" v-model.number="commentPageSize" @change="loadComments(1)">
              <option :value="10">10条/页</option>
              <option :value="20">20条/页</option>
              <option :value="50">50条/页</option>
            </select>
          </div>
        </section>

        <!-- ========== 分类管理 ========== -->
        <section v-if="activeTab === 'categories'" class="admin-panel">
          <div class="admin-panel__header">
            <h2 class="admin-panel__title">分类管理</h2>
            <div class="admin-panel__actions">
              <button class="btn-sm btn-sm--primary" @click="openModal('category')">+ 新增</button>
            </div>
          </div>
          <div class="filter-bar">
            <input v-model="categoryFilter.keyword" class="filter-input" placeholder="分类名称" @keyup.enter="loadCategories(1)" />
            <select v-model="categoryFilter.status" class="filter-select" @change="loadCategories(1)">
              <option value="">全部状态</option>
              <option value="1">启用</option>
              <option value="0">禁用</option>
            </select>
            <input type="date" v-model="categoryFilter.startTime" class="filter-date" @change="loadCategories(1)" title="创建开始" />
            <span class="filter-date-sep">~</span>
            <input type="date" v-model="categoryFilter.endTime" class="filter-date" @change="loadCategories(1)" title="创建结束" />
            <button class="btn-sm" @click="resetFilter(categoryFilter); loadCategories(1)">重置</button>
          </div>
          <table class="admin-table">
            <thead>
              <tr>
                <th>ID</th>
                <th>名称</th>
                <th>状态</th>
                <th>创建时间</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="c in adminCategories" :key="c.id">
                <td>{{ c.id }}</td>
                <td>{{ c.name }}</td>
                <td>
                  <span class="status-badge" :class="c.status === 1 ? 'status-badge--normal' : 'status-badge--banned'">
                    {{ c.status === 1 ? '启用' : '禁用' }}
                  </span>
                </td>
                <td>{{ fmt(c.created_at) }}</td>
                <td>
                  <div class="table-actions">
                    <button class="btn-sm" @click="openModal('category', c)">编辑</button>
                    <button class="btn-sm btn-sm--danger" @click="handleDelete('category', c.id)">删除</button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
          <div v-if="!adminCategories.length" class="admin-empty">暂无分类数据</div>
          <div v-if="adminCategories.length" class="pagination">
            <button class="pagination__btn" :disabled="categoryPage <= 1" @click="loadCategories(categoryPage - 1)">上一页</button>
            <template v-for="p in getPageRange(categoryPage, calcPages(categoryTotal, categoryPageSize))" :key="p">
              <span v-if="p === '...'" class="pagination__ellipsis">...</span>
              <button v-else class="pagination__btn" :class="{ 'pagination__btn--active': p === categoryPage }" @click="loadCategories(p)">{{ p }}</button>
            </template>
            <button class="pagination__btn" :disabled="categoryPage >= calcPages(categoryTotal, categoryPageSize)" @click="loadCategories(categoryPage + 1)">下一页</button>
            <span class="pagination__info">共 {{ categoryTotal }} 条</span>
            <select class="pagination__size" v-model.number="categoryPageSize" @change="loadCategories(1)">
              <option :value="10">10条/页</option>
              <option :value="20">20条/页</option>
              <option :value="50">50条/页</option>
            </select>
          </div>
        </section>

        <!-- ========== 标签管理 ========== -->
        <section v-if="activeTab === 'tags'" class="admin-panel">
          <div class="admin-panel__header">
            <h2 class="admin-panel__title">标签管理</h2>
            <div class="admin-panel__actions">
              <button class="btn-sm btn-sm--primary" @click="openModal('tag')">+ 新增</button>
            </div>
          </div>
          <div class="filter-bar">
            <input v-model="tagFilter.keyword" class="filter-input" placeholder="标签名称" @keyup.enter="loadTags(1)" />
            <select v-model="tagFilter.status" class="filter-select" @change="loadTags(1)">
              <option value="">全部状态</option>
              <option value="1">启用</option>
              <option value="0">禁用</option>
            </select>
            <input type="date" v-model="tagFilter.startTime" class="filter-date" @change="loadTags(1)" title="创建开始" />
            <span class="filter-date-sep">~</span>
            <input type="date" v-model="tagFilter.endTime" class="filter-date" @change="loadTags(1)" title="创建结束" />
            <button class="btn-sm" @click="resetFilter(tagFilter); loadTags(1)">重置</button>
          </div>
          <table class="admin-table">
            <thead>
              <tr>
                <th>ID</th>
                <th>名称</th>
                <th>状态</th>
                <th>创建时间</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="t in adminTags" :key="t.id">
                <td>{{ t.id }}</td>
                <td>{{ t.name }}</td>
                <td>
                  <span class="status-badge" :class="t.status === 1 ? 'status-badge--normal' : 'status-badge--banned'">
                    {{ t.status === 1 ? '启用' : '禁用' }}
                  </span>
                </td>
                <td>{{ fmt(t.created_at) }}</td>
                <td>
                  <div class="table-actions">
                    <button class="btn-sm" @click="openModal('tag', t)">编辑</button>
                    <button class="btn-sm btn-sm--danger" @click="handleDelete('tag', t.id)">删除</button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
          <div v-if="!adminTags.length" class="admin-empty">暂无标签数据</div>
          <div v-if="adminTags.length" class="pagination">
            <button class="pagination__btn" :disabled="tagPage <= 1" @click="loadTags(tagPage - 1)">上一页</button>
            <template v-for="p in getPageRange(tagPage, calcPages(tagTotal, tagPageSize))" :key="p">
              <span v-if="p === '...'" class="pagination__ellipsis">...</span>
              <button v-else class="pagination__btn" :class="{ 'pagination__btn--active': p === tagPage }" @click="loadTags(p)">{{ p }}</button>
            </template>
            <button class="pagination__btn" :disabled="tagPage >= calcPages(tagTotal, tagPageSize)" @click="loadTags(tagPage + 1)">下一页</button>
            <span class="pagination__info">共 {{ tagTotal }} 条</span>
            <select class="pagination__size" v-model.number="tagPageSize" @change="loadTags(1)">
              <option :value="10">10条/页</option>
              <option :value="20">20条/页</option>
              <option :value="50">50条/页</option>
            </select>
          </div>
        </section>

        <!-- ========== AI 模型配置 ========== -->
        <section v-if="activeTab === 'ai'" class="admin-panel">
          <AIModelPanel />
        </section>

        <!-- ========== 切分模板 ========== -->
        <section v-if="activeTab === 'chunk-templates'" class="admin-panel">
          <ChunkTemplatePanel />
        </section>

        <!-- ========== 切分管理 ========== -->
        <section v-if="activeTab === 'chunks'" class="admin-panel">
          <ChunkViewerPanel />
        </section>
      </main>
    </div>

    <!-- ========== 编辑/新增模态框 ========== -->
    <div v-if="modalVisible" class="modal-overlay" @click.self="modalVisible = false">
      <div class="modal">
        <h3 class="modal__title">{{ modalTitle }}</h3>
        <div class="modal__body">
          <!-- ---- 用户 ---- -->
          <template v-if="modalType === 'user'">
            <!-- 新增：显示昵称、账号、密码、状态 -->
            <template v-if="!editingId">
              <div class="modal__field">
                <label class="modal__label">昵称</label>
                <div class="modal__input-wrapper" :class="{ 'modal__input-wrapper--error': nameError, 'modal__input-wrapper--ok': !nameError && nameChecked }">
                  <input v-model="modalForm.user_name" class="modal__input" :class="{ 'modal__input--error': nameError }" placeholder="请输入昵称" @input="checkName" />
                  <span v-if="nameCheckLoading" class="modal__input-suffix">检查中...</span>
                  <span v-else-if="nameError" class="modal__input-suffix modal__input-suffix--error">昵称已存在</span>
                  <span v-else-if="nameChecked" class="modal__input-suffix modal__input-suffix--ok">可用</span>
                </div>
              </div>
              <div class="modal__field">
                <label class="modal__label">账号</label>
                <div class="modal__input-wrapper" :class="{ 'modal__input-wrapper--error': accountError || accountFormatError, 'modal__input-wrapper--ok': !accountError && !accountFormatError && accountChecked }">
                  <input v-model="modalForm.account" class="modal__input" :class="{ 'modal__input--error': accountError || accountFormatError }" placeholder="请输入 11 位数字账号" maxlength="11" @input="checkAccount" />
                  <span v-if="accountCheckLoading" class="modal__input-suffix">检查中...</span>
                  <span v-else-if="accountError" class="modal__input-suffix modal__input-suffix--error">账号已存在</span>
                  <span v-else-if="accountChecked && !accountFormatError" class="modal__input-suffix modal__input-suffix--ok">可用</span>
                </div>
                <p v-if="accountFormatError" class="modal__field-hint modal__field-hint--error">账号必须为 11 位数字</p>
              </div>
              <div class="modal__field">
                <label class="modal__label">密码</label>
                <div class="modal__input-wrapper" :class="{ 'modal__input-wrapper--error': passwordError }">
                  <input v-model="modalForm.password" class="modal__input" :class="{ 'modal__input--error': passwordError }" type="password" placeholder="11-20 位，需包含字母和数字" @input="checkPasswordFormat" />
                </div>
                <p v-if="passwordError" class="modal__field-hint modal__field-hint--error">{{ passwordErrorMsg }}</p>
                <p v-else class="modal__field-hint">密码需 11-20 位，必须同时包含字母和数字</p>
              </div>
              <div class="modal__field">
                <label class="modal__label">状态</label>
                <select v-model.number="modalForm.status" class="modal__input">
                  <option :value="1">正常</option>
                  <option :value="0">封禁</option>
                </select>
              </div>
            </template>
            <!-- 编辑：只保留状态 -->
            <template v-else>
              <div class="modal__field">
                <label class="modal__label">状态</label>
                <select v-model.number="modalForm.status" class="modal__input">
                  <option :value="1">正常</option>
                  <option :value="0">封禁</option>
                </select>
              </div>
            </template>
          </template>

          <!-- ---- 分类/标签 ---- -->
          <template v-else>
            <div class="modal__field">
              <label class="modal__label"><span class="required-mark">*</span>名称</label>
              <input v-model="modalForm.name" class="modal__input" :class="{ 'modal__input--required-empty': !modalForm.name?.trim() }" placeholder="请输入名称" />
            </div>
            <div class="modal__field">
              <label class="modal__label">状态</label>
              <select v-model.number="modalForm.status" class="modal__input">
                <option :value="1">启用</option>
                <option :value="0">禁用</option>
              </select>
            </div>
          </template>
        </div>
        <div class="modal__footer">
          <button class="btn btn--cancel" @click="closeModal">取消</button>
          <button class="btn btn--primary" :disabled="canSaveDisabled" @click="handleSave">保存</button>
        </div>
      </div>
    </div>

    <!-- ========== 删除确认模态框 ========== -->
    <div v-if="deleteModalVisible" class="modal-overlay" @click.self="deleteModalVisible = false">
      <div class="modal modal--delete">
        <h3 class="modal__title">确认删除</h3>
        <div class="modal__body">
          <p class="delete-confirm-text">确定要删除该{{ deleteTarget?.type === 'user' ? '用户' : deleteTarget?.type === 'article' ? '文章' : deleteTarget?.type === 'comment' ? '评论' : deleteTarget?.type === 'category' ? '分类' : '标签' }}吗？此操作不可恢复。</p>
        </div>
        <div class="modal__footer">
          <button class="btn btn--cancel" @click="deleteModalVisible = false">取消</button>
          <button class="btn btn--danger" @click="confirmDelete">确认删除</button>
        </div>
      </div>
    </div>

    <!-- ========== 向量化：选切分模板 ========== -->
    <div v-if="vectorModalVisible" class="modal-overlay" @click.self="vectorModalVisible = false">
      <div class="modal">
        <h3 class="modal__title">向量化文章</h3>
        <div class="modal__body">
          <p class="vector-modal__target">
            文章：<strong>{{ vectorTarget?.title }}</strong>
          </p>

          <div v-if="vectorTemplates.length" class="modal__field">
            <label class="modal__label"><span class="required-mark">*</span>切分模板</label>
            <select v-model="vectorTemplateId" class="modal__input">
              <option value="" disabled>请选择模板</option>
              <option v-for="t in vectorTemplates" :key="t.id" :value="String(t.id)">
                {{ t.name }}（{{ t.chunk_size }} 字/块，重叠 {{ t.chunk_overlap }}）
              </option>
            </select>
            <p class="vector-modal__hint">
              提交后立即返回，切分与向量化在后台排队执行。可以继续提交其他文章，也可以离开本页。
            </p>
          </div>

          <div v-else class="vector-modal__empty">
            还没有启用的切分模板。
            <button class="btn-sm btn-sm--primary" @click="goToChunkTemplates">去创建模板</button>
          </div>

          <div v-if="vectorTarget && vectorTarget.vector_status === 3" class="vector-modal__hint">
            该文章已有 {{ vectorTarget.vector_chunk_count }} 个块，重新向量化会先删掉旧块。
          </div>
        </div>
        <div class="modal__footer">
          <button class="btn btn--cancel" @click="vectorModalVisible = false">取消</button>
          <button
            class="btn btn--primary"
            :disabled="!vectorTemplateId || vectorSubmitting"
            @click="submitVectorize"
          >
            {{ vectorSubmitting ? '提交中…' : '开始向量化' }}
          </button>
        </div>
      </div>
    </div>

    <!-- ========== 一键转移分类模态框 ========== -->
    <div v-if="transferModalVisible" class="modal-overlay" @click.self="transferModalVisible = false">
      <div class="modal">
        <h3 class="modal__title">一键转移分类</h3>
        <div class="modal__body">
          <div class="modal__field">
            <label class="modal__label"><span class="required-mark">*</span>源分类</label>
            <select v-model.number="transferFrom" class="modal__input" :class="{ 'modal__input--required-empty': !transferFrom }">
              <option :value="0" disabled>请选择源分类</option>
              <option v-for="c in enabledCategories" :key="c.id" :value="c.id">{{ c.name }}</option>
            </select>
          </div>
          <div class="modal__field">
            <label class="modal__label"><span class="required-mark">*</span>目标分类</label>
            <select v-model.number="transferTo" class="modal__input" :class="{ 'modal__input--required-empty': !transferTo }">
              <option :value="0" disabled>请选择目标分类</option>
              <option v-for="c in enabledCategories" :key="c.id" :value="c.id">{{ c.name }}</option>
            </select>
          </div>
          <p class="transfer-hint">将源分类下的所有文章转移到目标分类。</p>
        </div>
        <div class="modal__footer">
          <button class="btn btn--cancel" @click="transferModalVisible = false">取消</button>
          <button class="btn btn--primary" :disabled="!transferFrom || !transferTo || transferFrom === transferTo" @click="handleTransfer">确认转移</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, onBeforeUnmount, type Ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import api from '../api'
import NavBar from '../components/NavBar.vue'
import AIModelPanel from '../components/AIModelPanel.vue'
import ChunkTemplatePanel from '../components/ChunkTemplatePanel.vue'
import ChunkViewerPanel from '../components/ChunkViewerPanel.vue'
import { showToast } from '../utils/toast'

// ---------- 类型 ----------
interface UserItem {
  id: number
  user_name: string
  avatar_url: string
  role_id: number
  status: number
  created_at: string
}

interface ArticleItem {
  id: number
  title: string
  type_id: number
  category?: { id: number; name: string }
  tags?: { id: number; name: string }[]
  summary: string
  content: string
  cover_url: string
  status: number
  views?: number
  likes?: number
  // ===== 向量化（仅后台列表返回）=====
  vector_status?: number // 0未处理 1排队中 2处理中 3已完成 4失败
  vector_template_id?: number
  vector_template_name?: string
  vector_chunk_count?: number
  vector_error?: string
  vectorized_at?: string
  updated_at?: string
}

interface CatTagItem {
  id: number
  name: string
  status: number
  created_at: string
}

interface AdminCommentItem {
  id: number
  article_title: string
  user_name: string
  content: string
  created_at: string
}

// ---------- 状态 ----------
const router = useRouter()
const route = useRoute()
const activeTab = ref('users')
const tabs = [
  { key: 'users', label: '用户管理', icon: '👥' },
  { key: 'articles', label: '文章管理', icon: '📝' },
  { key: 'comments', label: '评论管理', icon: '💬' },
  { key: 'categories', label: '分类管理', icon: '📁' },
  { key: 'tags', label: '标签管理', icon: '🏷️' },
  { key: 'ai', label: 'AI 模型', icon: '🤖' },
  { key: 'chunk-templates', label: '切分模板', icon: '✂️' },
  { key: 'chunks', label: '切分管理', icon: '🧩' }
]

const users = ref<UserItem[]>([])
const adminArticles = ref<ArticleItem[]>([])
const adminCategories = ref<CatTagItem[]>([])
const adminTags = ref<CatTagItem[]>([])
const adminComments = ref<AdminCommentItem[]>([])
const selectedCommentIds = ref<number[]>([])
const batchMode = ref(false)

// ---------- 筛选状态 ----------
const userFilter = reactive({ keyword: '', status: '' as string, startTime: '', endTime: '' })
const articleFilter = reactive({ keyword: '', status: '' as string, categoryId: '' as string })
const commentFilter = reactive({ articleTitle: '', userName: '', startTime: '', endTime: '' })
const categoryFilter = reactive({ keyword: '', status: '' as string, startTime: '', endTime: '' })
const tagFilter = reactive({ keyword: '', status: '' as string, startTime: '', endTime: '' })

const resetFilter = (filter: Record<string, any>) => {
  Object.keys(filter).forEach(k => { filter[k] = '' })
}

// 分页状态
const userPage = ref(1)
const userPageSize = ref(10)
const userTotal = ref(0)
const articlePage = ref(1)
const articlePageSize = ref(10)
const articleTotal = ref(0)
const commentPage = ref(1)
const commentPageSize = ref(10)
const commentTotal = ref(0)
const categoryPage = ref(1)
const categoryPageSize = ref(10)
const categoryTotal = ref(0)
const tagPage = ref(1)
const tagPageSize = ref(10)
const tagTotal = ref(0)

// 模态框
const modalVisible = ref(false)
const modalType = ref<'user' | 'category' | 'tag'>('user')
const editingId = ref<number | null>(null)
const editingUser = ref<UserItem | null>(null)

// 删除确认模态框
const deleteModalVisible = ref(false)
const deleteTarget = ref<{ type: string; id: number } | null>(null)

// 一键转移分类
const transferModalVisible = ref(false)
const transferFrom = ref(0)
const transferTo = ref(0)
const modalTitle = ref('')
const modalForm = reactive<Record<string, any>>({
  user_name: '',
  account: '',
  password: '',
  status: 1,
  title: '',
  type_id: 0,
  summary: '',
  content: '',
  cover_url: '',
  name: ''
})

// 仅启用的分类和标签（用于文章表单）
const enabledCategories = computed(() => adminCategories.value.filter(c => c.status === 1))
// 唯一性校验状态
const nameCheckLoading = ref(false)
const accountCheckLoading = ref(false)
const nameError = ref(false)
const accountError = ref(false)
const nameChecked = ref(false)
const accountChecked = ref(false)
let nameCheckTimer: ReturnType<typeof setTimeout> | null = null
let accountCheckTimer: ReturnType<typeof setTimeout> | null = null

// 账号格式校验
const accountFormatError = ref(false)
// 密码格式校验
const passwordError = ref(false)
const passwordErrorMsg = ref('')

const checkPasswordFormat = () => {
  const val = modalForm.password || ''
  if (!val) {
    passwordError.value = false
    passwordErrorMsg.value = ''
    return
  }
  if (val.length < 11 || val.length > 20) {
    passwordError.value = true
    passwordErrorMsg.value = '密码长度需为 11-20 位'
    return
  }
  if (!/[a-zA-Z]/.test(val) || !/\d/.test(val)) {
    passwordError.value = true
    passwordErrorMsg.value = '密码必须同时包含字母和数字'
    return
  }
  if (/[^a-zA-Z0-9]/.test(val)) {
    passwordError.value = true
    passwordErrorMsg.value = '密码只能包含字母和数字'
    return
  }
  passwordError.value = false
  passwordErrorMsg.value = ''
}

// 是否全选评论
const allCommentsSelected = computed(() => {
  return adminComments.value.length > 0 && selectedCommentIds.value.length === adminComments.value.length
})

// 全选/取消全选评论
const toggleAllComments = () => {
  if (allCommentsSelected.value) {
    selectedCommentIds.value = []
  } else {
    selectedCommentIds.value = adminComments.value.map(c => c.id)
  }
}

// 退出批量模式
const exitBatchMode = () => {
  batchMode.value = false
  selectedCommentIds.value = []
}

// 批量删除评论
const handleBatchDeleteComments = async () => {
  if (!selectedCommentIds.value.length) return
  try {
    await api.delete('/admin/comments', { data: { ids: selectedCommentIds.value } })
    showToast('批量删除成功', 'success')
    exitBatchMode()
    await loadComments(commentPage.value)
  } catch (error: any) {
    console.error('批量删除失败:', error)
    const msg = error?.response?.data?.message || '批量删除失败，请重试'
    showToast(msg, 'error')
  }
}

// ---------- 分页加载 ----------
const loadUsers = async (page = 1) => {
  userPage.value = page
  try {
    const params: Record<string, any> = { page: userPage.value, page_size: userPageSize.value }
    if (userFilter.keyword) params.keyword = userFilter.keyword
    if (userFilter.status !== '') params.status = userFilter.status
    if (userFilter.startTime) params.start_time = userFilter.startTime
    if (userFilter.endTime) params.end_time = userFilter.endTime
    const res = await api.get('/admin/users', { params })
    users.value = res.data.data?.list || []
    userTotal.value = res.data.data?.total || 0
  } catch (e) { console.error('加载用户失败:', e) }
}

const loadArticles = async (page = 1) => {
  articlePage.value = page
  try {
    const params: Record<string, any> = { page: articlePage.value, page_size: articlePageSize.value }
    if (articleFilter.keyword) params.keyword = articleFilter.keyword
    if (articleFilter.status !== '') params.status = articleFilter.status
    if (articleFilter.categoryId) params.category_id = articleFilter.categoryId
    const res = await api.get('/admin/articles', { params })
    adminArticles.value = res.data.data?.list || []
    articleTotal.value = res.data.data?.total || 0
  } catch (e) { console.error('加载文章失败:', e) }
}

// ---------- 向量化 ----------
interface ChunkTemplateOption {
  id: number
  name: string
  chunk_size: number
  chunk_overlap: number
}

const vectorModalVisible = ref(false)
const vectorTarget = ref<ArticleItem | null>(null)
const vectorTemplates = ref<ChunkTemplateOption[]>([])
const vectorTemplateId = ref('')
const vectorSubmitting = ref(false)
// 正在提交请求的那一行（与「已入队」状态区分开，避免按钮提前变灰引起困惑）
const vectorBusyId = ref<number | null>(null)

// 每个文章一个轮询定时器，页面卸载时统一清掉
const pollTimers = new Map<number, number>()

/**
 * 向量化状态 → 文案与样式。
 *
 * 特意区分「排队中(1)」和「处理中(2)」：两者都属于同一条链路，
 * 用同一个蓝但靠有无旋转图标区分（这里用文案前缀区分，样式相同）。
 * 「待更新」是 3 且 vectorized_at 早于 updated_at —— 正文改过了，旧块已过期。
 */
const vectorState = (a: ArticleItem) => {
  const s = a.vector_status ?? 0
  if (s === 1) return { text: '排队中', cls: 'status-badge--queued', tip: '已提交，等待处理' }
  if (s === 2) return { text: '处理中…', cls: 'status-badge--running', tip: '正在切分与向量化' }
  if (s === 4) {
    return { text: '失败', cls: 'status-badge--failed', tip: a.vector_error || '向量化失败' }
  }
  if (s === 3) {
    // 正文改过但没重新向量化
    const v = a.vectorized_at ? Date.parse(a.vectorized_at) : NaN
    const u = a.updated_at ? Date.parse(a.updated_at) : NaN
    if (!Number.isNaN(v) && !Number.isNaN(u) && v < u) {
      return { text: '待更新', cls: 'status-badge--stale', tip: '正文已修改，需要重新向量化' }
    }
    return {
      text: `是（${a.vector_chunk_count || 0} 块）`,
      cls: 'status-badge--normal',
      tip: a.vector_template_name ? `模板：${a.vector_template_name}` : ''
    }
  }
  return { text: '否', cls: 'status-badge--idle', tip: '尚未向量化' }
}

const loadVectorTemplates = async () => {
  try {
    const res = await api.get('/admin/chunk-templates', { params: { status: 1 } })
    vectorTemplates.value = (res.data?.data || []).map((t: any) => ({
      id: t.id,
      name: t.name,
      chunk_size: t.chunk_size,
      chunk_overlap: t.chunk_overlap
    }))
  } catch {
    vectorTemplates.value = []
  }
}

const openVectorModal = async (a: ArticleItem) => {
  vectorTarget.value = a
  vectorTemplateId.value = ''
  vectorModalVisible.value = true
  if (!vectorTemplates.value.length) {
    await loadVectorTemplates()
  }
  // 只有一个模板时直接选中，省一次点击
  if (vectorTemplates.value.length === 1) {
    vectorTemplateId.value = String(vectorTemplates.value[0].id)
  }
}

const goToChunkTemplates = () => {
  vectorModalVisible.value = false
  activeTab.value = 'chunk-templates'
}

const stopPoll = (articleId: number) => {
  const timer = pollTimers.get(articleId)
  if (timer) {
    clearTimeout(timer)
    pollTimers.delete(articleId)
  }
}

/**
 * 轮询状态。
 *
 * 用 setTimeout 串行而不是 setInterval：上一次响应回来才排下一次，
 * 避免接口变慢时请求堆叠（本项目的 axios 有 401 自动刷新，堆叠会放大刷新风暴）。
 * 次数上限 100 次，避免用户忘了关页面而一直打接口。
 */
const startPoll = (articleId: number) => {
  stopPoll(articleId)
  let count = 0

  const tick = async () => {
    count += 1
    if (count > 100) {
      pollTimers.delete(articleId)
      return
    }
    try {
      const res = await api.get(`/admin/articles/${articleId}/vector-status`)
      const st = res.data?.data
      if (!st) {
        pollTimers.delete(articleId)
        return
      }
      const row = adminArticles.value.find((a) => a.id === articleId)
      if (row) {
        row.vector_status = st.vector_status
        row.vector_chunk_count = st.vector_chunk_count
        row.vector_error = st.vector_error
        row.vectorized_at = st.vectorized_at
        row.updated_at = st.updated_at
        row.vector_template_id = st.vector_template_id
      }
      if (st.vector_status === 3) {
        pollTimers.delete(articleId)
        showToast(`《${row?.title || ''}》向量化完成，共 ${st.vector_chunk_count} 块`, 'success')
        return
      }
      if (st.vector_status === 4) {
        pollTimers.delete(articleId)
        showToast(`《${row?.title || ''}》向量化失败：${st.vector_error || '未知原因'}`, 'error')
        return
      }
      pollTimers.set(articleId, window.setTimeout(tick, 3000))
    } catch {
      pollTimers.delete(articleId)
    }
  }

  pollTimers.set(articleId, window.setTimeout(tick, 1000))
}

const submitVectorize = async () => {
  const target = vectorTarget.value
  if (!target || !vectorTemplateId.value) return
  vectorBusyId.value = target.id
  vectorSubmitting.value = true
  try {
    const res = await api.post(`/admin/articles/${target.id}/vectorize`, {
      template_id: Number(vectorTemplateId.value)
    })
    const d = res.data?.data
    const row = adminArticles.value.find((a) => a.id === target.id)
    if (row && d) row.vector_status = d.vector_status
    showToast('已提交，排队处理中', 'success')
    vectorModalVisible.value = false
    startPoll(target.id)
  } catch (err: any) {
    showToast(err?.response?.data?.message || '提交失败', 'error')
  } finally {
    vectorBusyId.value = null
    vectorSubmitting.value = false
  }
}

// 离开页面必须清掉所有定时器，否则会一直在后台打接口
onBeforeUnmount(() => {
  for (const timer of pollTimers.values()) {
    clearTimeout(timer)
  }
  pollTimers.clear()
})

const loadComments = async (page = 1) => {
  commentPage.value = page
  try {
    const params: Record<string, any> = { page: commentPage.value, page_size: commentPageSize.value }
    if (commentFilter.articleTitle) params.article_title = commentFilter.articleTitle
    if (commentFilter.userName) params.user_name = commentFilter.userName
    if (commentFilter.startTime) params.start_time = commentFilter.startTime
    if (commentFilter.endTime) params.end_time = commentFilter.endTime
    const res = await api.get('/admin/comments', { params })
    adminComments.value = res.data.data?.list || []
    commentTotal.value = res.data.data?.total || 0
  } catch (e) { console.error('加载评论失败:', e) }
}

const loadCategories = async (page = 1) => {
  categoryPage.value = page
  try {
    const params: Record<string, any> = { page: categoryPage.value, page_size: categoryPageSize.value }
    if (categoryFilter.keyword) params.keyword = categoryFilter.keyword
    if (categoryFilter.status !== '') params.status = categoryFilter.status
    if (categoryFilter.startTime) params.start_time = categoryFilter.startTime
    if (categoryFilter.endTime) params.end_time = categoryFilter.endTime
    const res = await api.get('/admin/categories', { params })
    adminCategories.value = res.data.data?.list || []
    categoryTotal.value = res.data.data?.total || 0
  } catch (e) { console.error('加载分类失败:', e) }
}

const loadTags = async (page = 1) => {
  tagPage.value = page
  try {
    const params: Record<string, any> = { page: tagPage.value, page_size: tagPageSize.value }
    if (tagFilter.keyword) params.keyword = tagFilter.keyword
    if (tagFilter.status !== '') params.status = tagFilter.status
    if (tagFilter.startTime) params.start_time = tagFilter.startTime
    if (tagFilter.endTime) params.end_time = tagFilter.endTime
    const res = await api.get('/admin/tags', { params })
    adminTags.value = res.data.data?.list || []
    tagTotal.value = res.data.data?.total || 0
  } catch (e) { console.error('加载标签失败:', e) }
}

const calcPages = (total: number, size: number) => Math.ceil(total / size) || 1

const getPageRange = (current: number, total: number): (number | '...')[] => {
  if (total <= 7) return Array.from({ length: total }, (_, i) => i + 1)
  const pages: (number | '...')[] = []
  if (current <= 4) {
    for (let i = 1; i <= 5; i++) pages.push(i)
    pages.push('...'); pages.push(total)
  } else if (current >= total - 3) {
    pages.push(1); pages.push('...')
    for (let i = total - 4; i <= total; i++) pages.push(i)
  } else {
    pages.push(1); pages.push('...')
    for (let i = current - 1; i <= current + 1; i++) pages.push(i)
    pages.push('...'); pages.push(total)
  }
  return pages
}

// 保存按钮是否禁用（新增用户时，昵称或校验失败或未通过唯一性校验时禁用）
const canSaveDisabled = computed(() => {
  // 新增用户：唯一性校验通过后才可保存
  if (modalType.value === 'user' && !editingId.value) {
    return nameError.value || accountError.value || accountFormatError.value || passwordError.value || nameCheckLoading.value || accountCheckLoading.value || !nameChecked.value || !accountChecked.value
  }
  // 分类/标签：名称必填
  if (modalType.value === 'category' || modalType.value === 'tag') {
    return !modalForm.name?.trim()
  }
  return false
})

const checkName = () => {
  nameCheckTimer && clearTimeout(nameCheckTimer)
  nameChecked.value = false
  const val = modalForm.user_name?.trim()
  if (!val) {
    nameError.value = false
    nameCheckLoading.value = false
    return
  }
  nameCheckLoading.value = true
  nameCheckTimer = setTimeout(async () => {
    try {
      const res = await api.get('/auth/is_exists_name', { params: { user_name: val } })
      const exists = res.data?.data?.is_exists
      nameError.value = exists === true
      nameChecked.value = true
    } catch {
      nameError.value = false
      nameChecked.value = false
    } finally {
      nameCheckLoading.value = false
    }
  }, 500)
}

const checkAccount = () => {
  accountCheckTimer && clearTimeout(accountCheckTimer)
  accountChecked.value = false
  const val = modalForm.account?.trim()
  if (!val) {
    accountError.value = false
    accountCheckLoading.value = false
    accountFormatError.value = false
    return
  }
  // 格式校验：11 位数字
  if (!/^\d{11}$/.test(val)) {
    accountFormatError.value = true
    accountError.value = false
    accountChecked.value = false
    accountCheckLoading.value = false
    return
  }
  accountFormatError.value = false
  accountCheckLoading.value = true
  accountCheckTimer = setTimeout(async () => {
    try {
      const res = await api.get('/auth/is_exists_account', { params: { account: val } })
      const exists = res.data?.data?.is_exists
      accountError.value = exists === true
      accountChecked.value = true
    } catch {
      accountError.value = false
      accountChecked.value = false
    } finally {
      accountCheckLoading.value = false
    }
  }, 500)
}

// ---------- 方法 ----------
const fmt = (d: string) => {
  if (!d) return ''
  return new Date(d).toLocaleDateString('zh-CN')
}

const openModal = (type: 'user' | 'category' | 'tag', item?: any) => {
  modalType.value = type
  modalVisible.value = true
  editingId.value = item?.id || null
  editingUser.value = item || null

  // 重置唯一性校验状态
  nameError.value = false
  accountError.value = false
  nameChecked.value = false
  accountChecked.value = false
  nameCheckLoading.value = false
  accountCheckLoading.value = false
  accountFormatError.value = false
  passwordError.value = false
  passwordErrorMsg.value = ''

  // reset
  modalForm.user_name = ''
  modalForm.account = ''
  modalForm.password = ''
  modalForm.status = 1
  modalForm.name = ''

  if (item) {
    modalTitle.value = type === 'user' ? '编辑用户' : type === 'category' ? '编辑分类' : '编辑标签'
    if (type === 'user') {
      modalForm.status = item.status ?? 1
    } else {
      modalForm.name = item.name || ''
      modalForm.status = item.status ?? 1
    }
  } else {
    modalTitle.value = type === 'user' ? '新增用户' : type === 'category' ? '新增分类' : '新增标签'
  }
}

const closeModal = () => {
  modalVisible.value = false
}

const handleSave = async () => {
  try {
    if (modalType.value === 'user') {
      if (editingId.value) {
        // 更新用户状态
        await api.put(`/admin/users/${editingId.value}`, { status: modalForm.status })
      } else {
        // 创建用户
        await api.post('/admin/users', {
          user_name: modalForm.user_name,
          account: modalForm.account,
          password: modalForm.password,
          status: modalForm.status
        })
      }
      // 重新加载用户列表
      await loadUsers(editingId.value ? userPage.value : 1)
    } else if (modalType.value === 'category') {
      const categoryData = {
        name: modalForm.name,
        status: modalForm.status
      }
      
      if (editingId.value) {
        await api.put(`/admin/categories/${editingId.value}`, categoryData)
      } else {
        await api.post('/admin/categories', categoryData)
      }
      // 重新加载分类列表
      await loadCategories(editingId.value ? categoryPage.value : 1)
    } else if (modalType.value === 'tag') {
      const tagData = {
        name: modalForm.name,
        status: modalForm.status
      }
      
      if (editingId.value) {
        await api.put(`/admin/tags/${editingId.value}`, tagData)
      } else {
        await api.post('/admin/tags', tagData)
      }
      // 重新加载标签列表
      await loadTags(editingId.value ? tagPage.value : 1)
    }
    
    modalVisible.value = false
  } catch (error: any) {
    console.error('保存失败:', error)
    const msg = error?.response?.data?.message || '保存失败，请重试'
    showToast(msg, 'error')
  }
}

const handleDelete = (type: string, id: number) => {
  deleteTarget.value = { type, id }
  deleteModalVisible.value = true
}

const confirmDelete = async () => {
  if (!deleteTarget.value) return
  const { type, id } = deleteTarget.value
  deleteModalVisible.value = false
  deleteTarget.value = null

  const reloadWithFallback = async (loadFn: Function, page: Ref<number>, data: Ref<any[]>) => {
    await loadFn(page.value)
    if (data.value.length === 0 && page.value > 1) {
      await loadFn(page.value - 1)
    }
  }

  try {
    if (type === 'user') {
      await api.delete(`/admin/users/${id}`)
      await reloadWithFallback(loadUsers, userPage, users)
    } else if (type === 'article') {
      await api.delete(`/admin/articles/${id}`)
      await reloadWithFallback(loadArticles, articlePage, adminArticles)
    } else if (type === 'comment') {
      await api.delete(`/admin/comments/${id}`)
      await reloadWithFallback(loadComments, commentPage, adminComments)
    } else if (type === 'category') {
      await api.delete(`/admin/categories/${id}`)
      await reloadWithFallback(loadCategories, categoryPage, adminCategories)
    } else if (type === 'tag') {
      await api.delete(`/admin/tags/${id}`)
      await reloadWithFallback(loadTags, tagPage, adminTags)
    }
    showToast('删除成功', 'success')
  } catch (error: any) {
    console.error('删除失败:', error)
    const msg = error?.response?.data?.message || '删除失败，请重试'
    showToast(msg, 'error')
  }
}

const goToArticleEditor = (path: string) => {
  // 离开前记住当前文章列表页码
  sessionStorage.setItem('admin_article_page', String(articlePage.value))
  router.push(path)
}

const handleTransfer = async () => {
  if (!transferFrom.value || !transferTo.value || transferFrom.value === transferTo.value) return
  try {
    await api.put('/admin/articles/transfer', {
      from_type_id: transferFrom.value,
      to_type_id: transferTo.value
    })
    showToast('转移成功', 'success')
    transferModalVisible.value = false
    transferFrom.value = 0
    transferTo.value = 0
    // 重新加载文章和分类列表
    await Promise.all([
      loadArticles(articlePage.value),
      loadCategories(categoryPage.value)
    ])
  } catch (error: any) {
    console.error('转移失败:', error)
    const msg = error?.response?.data?.message || '转移失败，请重试'
    showToast(msg, 'error')
  }
}

onMounted(async () => {
  // 从 query 参数切到指定 tab（文章编辑器返回时跳转到文章管理）
  const tabParam = route.query.tab
  if (tabParam && tabs.some(t => t.key === tabParam)) {
    activeTab.value = tabParam as string
  }

  // 恢复从文章编辑器返回时保存的页码
  let articleStartPage = 1
  const savedPage = sessionStorage.getItem('admin_article_page')
  if (savedPage) {
    articleStartPage = parseInt(savedPage, 10) || 1
    sessionStorage.removeItem('admin_article_page')
  }

  try {
    await Promise.all([
      loadUsers(1),
      loadArticles(articleStartPage),
      loadComments(1),
      loadCategories(1),
      loadTags(1)
    ])
  } catch (error) {
    console.error('加载数据失败:', error)
    showToast('加载数据失败，请刷新页面重试', 'error')
  }
})
</script>

<style scoped>
.admin-layout {
  display: flex;
  min-height: calc(100vh - 50px);
}

.admin-sidebar {
  width: 200px;
  background: #f5f7fa;
  border-right: 1px solid #e8e8e8;
  padding: 16px 0;
  flex-shrink: 0;
}

.admin-sidebar__title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  font-weight: 600;
  color: #666;
  padding: 0 16px 12px;
  border-bottom: 1px solid #e8e8e8;
  margin-bottom: 8px;
}

.admin-sidebar__logo {
  width: 20px;
  height: 20px;
  object-fit: contain;
  display: block;
  flex-shrink: 0;
}

.admin-sidebar__link {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 10px 16px;
  border: none;
  background: transparent;
  color: #333;
  font-size: 14px;
  cursor: pointer;
  transition: background 0.2s;
  text-align: left;
}

.admin-sidebar__link:hover { background: #e8f4ff; }
.admin-sidebar__link--active { background: #e8f4ff; color: #1677ff; font-weight: 500; }

.admin-main { flex: 1; padding: 20px; overflow-x: auto; }

.admin-panel__header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.admin-panel__title { font-size: 18px; font-weight: 600; }

.admin-table {
  width: 100%;
  border-collapse: collapse;
  background: #fff;
  border-radius: 8px;
  overflow: hidden;
  box-shadow: 0 1px 4px rgba(0,0,0,0.06);
}

.admin-table th,
.admin-table td {
  padding: 12px 14px;
  text-align: left;
  border-bottom: 1px solid #f0f0f0;
  font-size: 13px;
}

.admin-table th {
  background: #fafafa;
  font-weight: 500;
  color: #555;
}

.admin-table tbody tr:hover { background: #fafafa; }

.table-avatar { width: 36px; height: 36px; border-radius: 50%; object-fit: cover; display: inline-block; }
.table-avatar--placeholder { font-size: 24px; line-height: 36px; text-align: center; display: inline-block; width: 36px; }

.role-badge, .status-badge, .cat-tag-badge {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 12px;
  font-weight: 500;
}

.role-badge--admin { background: #e6f0ff; color: #1677ff; }
.role-badge--user { background: #f0f0f0; color: #666; }

.status-badge--normal { background: #f6ffed; color: #52c41a; }
.status-badge--banned { background: #fff2f0; color: #ff4d4f; }
.status-badge--draft { background: #fffbe6; color: #faad14; }

/* ===== 向量化状态 ===== */
/* 排队中与处理中同色：同一条链路不该用两个色相，否则会让人以为是两种不同的问题 */
.status-badge--queued { background: #e6f4ff; color: #1677ff; }
.status-badge--running { background: #e6f4ff; color: #1677ff; }
.status-badge--failed { background: #fff2f0; color: #ff4d4f; }
.status-badge--stale { background: #fff7e6; color: #fa8c16; }
.status-badge--idle { background: #f5f5f5; color: #999; }

.vector-modal__target {
  margin: 0 0 12px;
  font-size: 0.85rem;
  line-height: 1.5;
  color: #555;
}

.vector-modal__hint {
  margin: 6px 0 0;
  font-size: 0.76rem;
  line-height: 1.55;
  color: #888;
}

.vector-modal__empty {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 10px;
  padding: 14px;
  border-radius: 8px;
  font-size: 0.82rem;
  color: #666;
  background: rgba(148, 163, 184, 0.1);
}

.cat-tag-badge { background: #f0f5ff; color: #2f54eb; margin-right: 4px; }
.cat-tag-badge--tag { background: #f0f0f0; color: #666; }

.table-actions { display: flex; gap: 6px; }

.btn-sm {
  padding: 4px 10px;
  border: 1px solid #d9d9d9;
  border-radius: 4px;
  background: #fff;
  font-size: 12px;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-sm:hover { border-color: #1677ff; color: #1677ff; }
.btn-sm--primary { background: #1677ff; color: #fff; border-color: #1677ff; }
.btn-sm--primary:hover { background: #4096ff; border-color: #4096ff; color: #fff; }
.btn-sm--danger { color: #ff4d4f; border-color: #ff4d4f; }
.btn-sm--danger:hover { background: #ff4d4f; color: #fff; }

.admin-empty {
  text-align: center;
  padding: 40px;
  color: #999;
  font-size: 14px;
}

/* ---------- 输入框唯一性校验 ---------- */
.modal__input-wrapper {
  display: flex;
  align-items: center;
  border: 1px solid #d9d9d9;
  border-radius: 4px;
  transition: border-color 0.2s;
  overflow: hidden;
}

.modal__input-wrapper:focus-within { border-color: #1677ff; box-shadow: 0 0 0 2px rgba(22,119,255,0.1); }
.modal__input-wrapper--error { border-color: #ff4d4f; }
.modal__input-wrapper--error:focus-within { border-color: #ff4d4f; box-shadow: 0 0 0 2px rgba(255,77,79,0.1); }
.modal__input-wrapper--ok { border-color: #52c41a; }
.modal__input-wrapper--ok:focus-within { border-color: #52c41a; box-shadow: 0 0 0 2px rgba(82,196,26,0.1); }

.modal__input-wrapper .modal__input {
  flex: 1;
  border: none;
  box-shadow: none;
}

.modal__input-wrapper .modal__input:focus { box-shadow: none; }

.modal__input-wrapper .modal__input--error { color: #ff4d4f; }

.modal__input-suffix {
  flex-shrink: 0;
  padding: 0 11px;
  font-size: 12px;
  color: #999;
  white-space: nowrap;
}

.modal__input-suffix--error { color: #ff4d4f; }
.modal__input-suffix--ok { color: #52c41a; }

.modal__field-hint {
  margin: 4px 0 0;
  font-size: 12px;
  color: #999;
  line-height: 1.4;
}
.modal__field-hint--error {
  color: #ff4d4f;
}

.btn--primary:disabled { background: #a0c4ff; border-color: #a0c4ff; cursor: not-allowed; }

/* ---------- 模态框 ---------- */
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0,0,0,0.45);
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 1000;
}

.modal {
  background: #fff;
  border-radius: 8px;
  width: 460px;
  max-height: 80vh;
  overflow-y: auto;
  box-shadow: 0 6px 16px rgba(0,0,0,0.12);
}

.modal__title {
  font-size: 16px;
  font-weight: 600;
  padding: 16px 20px;
  border-bottom: 1px solid #f0f0f0;
}

.modal__body { padding: 16px 20px; }

.modal__field { margin-bottom: 14px; }

.modal__label {
  display: block;
  margin-bottom: 4px;
  font-size: 13px;
  font-weight: 500;
  color: #333;
}

.modal__input {
  width: 100%;
  padding: 7px 11px;
  border: 1px solid #d9d9d9;
  border-radius: 4px;
  font-size: 13px;
  transition: border-color 0.2s;
  box-sizing: border-box;
}

.modal__input:focus { border-color: #1677ff; outline: none; box-shadow: 0 0 0 2px rgba(22,119,255,0.1); }

select.modal__input { appearance: auto; }

.modal__input--required-empty { border-color: #ff4d4f; }
.modal__input--required-empty:focus { border-color: #ff4d4f; box-shadow: 0 0 0 2px rgba(255,77,79,0.1); }

.required-mark { color: #ff4d4f; margin-right: 2px; font-size: 14px; }

.modal__footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  padding: 12px 20px;
  border-top: 1px solid #f0f0f0;
}

.btn {
  padding: 8px 20px;
  border-radius: 4px;
  font-size: 13px;
  cursor: pointer;
  border: 1px solid #d9d9d9;
  transition: all 0.2s;
}

.btn--primary { background: #1677ff; color: #fff; border-color: #1677ff; }
.btn--primary:hover { background: #4096ff; }
.btn--cancel { background: #fff; }
.btn--cancel:hover { border-color: #1677ff; color: #1677ff; }
.btn--danger { background: #ff4d4f; color: #fff; border-color: #ff4d4f; }
.btn--danger:hover { background: #ff7875; border-color: #ff7875; }

.modal--delete { width: 380px; }

.delete-confirm-text { font-size: 14px; color: #333; line-height: 1.6; margin: 8px 0; }

.transfer-hint {
  font-size: 12px;
  color: #999;
  margin: 8px 0 0;
}

/* ---------- 分页 ---------- */
.pagination {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 6px;
  padding: 16px 0;
  flex-wrap: wrap;
}

.pagination__btn {
  min-width: 32px;
  height: 32px;
  padding: 0 8px;
  border: 1px solid #d9d9d9;
  border-radius: 4px;
  background: #fff;
  color: #333;
  font-size: 13px;
  cursor: pointer;
  transition: all 0.2s;
}

.pagination__btn:hover { border-color: #1677ff; color: #1677ff; }
.pagination__btn--active { background: #1677ff; color: #fff; border-color: #1677ff; }
.pagination__btn--active:hover { background: #4096ff; border-color: #4096ff; color: #fff; }
.pagination__btn:disabled { color: #d9d9d9; border-color: #d9d9d9; cursor: not-allowed; background: #fafafa; }

.pagination__ellipsis {
  display: inline-block;
  width: 24px;
  text-align: center;
  color: #999;
  font-size: 13px;
}

.pagination__info {
  margin-left: 8px;
  font-size: 13px;
  color: #666;
}

.pagination__size {
  margin-left: 4px;
  padding: 4px 6px;
  border: 1px solid #d9d9d9;
  border-radius: 4px;
  font-size: 13px;
  color: #333;
  background: #fff;
  cursor: pointer;
}

/* ---------- 筛选栏 ---------- */
.filter-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
  flex-wrap: wrap;
}

.filter-input {
  width: 160px;
  padding: 6px 10px;
  border: 1px solid #d9d9d9;
  border-radius: 4px;
  font-size: 13px;
  transition: border-color 0.2s;
  box-sizing: border-box;
}

.filter-input:focus {
  border-color: #1677ff;
  outline: none;
  box-shadow: 0 0 0 2px rgba(22,119,255,0.1);
}

.filter-select {
  padding: 6px 10px;
  border: 1px solid #d9d9d9;
  border-radius: 4px;
  font-size: 13px;
  color: #333;
  background: #fff;
  cursor: pointer;
  box-sizing: border-box;
}

.filter-date {
  padding: 5px 8px;
  border: 1px solid #d9d9d9;
  border-radius: 4px;
  font-size: 13px;
  color: #333;
  background: #fff;
  cursor: pointer;
  box-sizing: border-box;
}

.filter-date-sep {
  color: #999;
  font-size: 13px;
}
</style>