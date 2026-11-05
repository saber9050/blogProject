package chat

import (
	"os"
	"path/filepath"
	"testing"

	"blog/pkg/config"
	"blog/pkg/logger"
)

// TestMain 初始化日志。
//
// 不加这段，任何在测试路径上打日志的用例都会因为全局 logger 为 nil 而 panic
// ——被测代码里有大量「失败只 warn 不中断」的分支，那些分支正是最需要被覆盖的。
// 日志写到临时目录，且级别调到 error，避免污染测试输出。
func TestMain(m *testing.M) {
	logFile := filepath.Join(os.TempDir(), "blog-chat-test.log")
	_ = logger.Init(&config.LogConfig{
		Level:    "error",
		Filename: logFile,
		MaxSize:  1,
	})
	code := m.Run()
	_ = logger.Sync()
	_ = os.Remove(logFile)
	os.Exit(code)
}
