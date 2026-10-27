package main

import (
	"blog/internal/app"
	"blog/internal/migrate"
	"fmt"
	"os"
)

func main() {
	// 子命令：把 MySQL 数据整体迁移到 PostgreSQL。
	// 做进主程序而不是单独的二进制，是为了服务器上能直接复用已部署的镜像跑迁移，
	// 不必为了搬一次数据往服务器装 Go 工具链。
	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		if err := migrate.Run(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "\n迁移失败: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// 创建应用实例
	application := app.NewApp()

	// 初始化应用
	if err := application.Initialize(); err != nil {
		panic(fmt.Sprintf("应用初始化失败: %v", err))
	}

	// 运行应用
	application.Run()
}
