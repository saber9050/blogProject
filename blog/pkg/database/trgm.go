package database

import (
	"sync/atomic"

	"blog/pkg/logger"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// trgmReady pg_trgm 扩展是否可用。检索层据此决定关键词路走 word_similarity
// 还是降级成 ILIKE 命中计数。
//
// 用包级原子变量而不是往各处传参：它描述的是「这个库的能力」，属于进程级事实，
// 传参会污染一整条调用链，而真正用它的人只有关键词检索那一处。
var trgmReady atomic.Bool

// EnsureTrgm 启动自检并确保 pg_trgm 扩展存在。
//
// 为什么需要它：模块 05 的关键词检索路用 word_similarity()，这函数来自 contrib 扩展
// pg_trgm。生产镜像是 pgvector/pgvector:pg17（基于官方 postgres 镜像），contrib 是
// 自带的，所以 CREATE EXTENSION 一般能成功；但受权限或精简镜像影响也可能失败。
// 失败不能让整站起不来，只把关键词路降级为 ILIKE，只 warn 一次。
//
// 自检放在 AutoMigrate 之后：扩展属于库级对象，与建表顺序无关，但放在「表结构就绪」
// 之后能让启动日志的因果顺序更清楚。
func EnsureTrgm(db *gorm.DB) bool {
	if db == nil {
		return false
	}

	// 先查 pg_extension 再决定要不要 CREATE：pg_extension 是所有人可读的系统表，
	// 而 CREATE EXTENSION 需要超级用户权限。已经装好的库不该因为「当前账号不是超管」
	// 而被判定为不可用——那是最容易误报的一种失败。
	var exists bool
	if err := db.Raw(
		"SELECT EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'pg_trgm')",
	).Scan(&exists).Error; err != nil {
		logger.Warn("pg_trgm 自检失败，关键词检索将降级为 ILIKE", zap.Error(err))
		trgmReady.Store(false)
		return false
	}

	if !exists {
		if err := db.Exec("CREATE EXTENSION IF NOT EXISTS pg_trgm").Error; err != nil {
			logger.Warn("pg_trgm 扩展创建失败，关键词检索将降级为 ILIKE（功能可用但效果差一档）",
				zap.Error(err))
			trgmReady.Store(false)
			return false
		}
	}

	trgmReady.Store(true)
	logger.Info("pg_trgm 扩展可用，关键词检索走 word_similarity")
	return true
}

// TrgmReady 关键词检索能否使用 pg_trgm。未自检或自检失败时为 false。
func TrgmReady() bool {
	return trgmReady.Load()
}
