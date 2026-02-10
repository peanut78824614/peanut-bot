// 一次性测试：拉取 Kyber 池子数据并打印（不发送 Telegram）
// 用法：cd 项目根目录 && go run scripts/test_kyber_fetch.go
package main

import (
	"context"
	"data/internal/service"
	"fmt"
	"os"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"
)

func main() {
	ctx := gctx.New()
	// 确保从项目根目录运行，以便加载 config/config.yaml
	if _, err := os.Stat("config/config.yaml"); err != nil {
		fmt.Println("请从项目根目录运行: go run scripts/test_kyber_fetch.go")
		os.Exit(1)
	}

	kyber := service.KyberSwap()
	fmt.Println("正在拉取 Kyber 池子数据 (page=1, 仅含 WETH)...")
	pools, err := kyber.FetchAllPools(ctx)
	if err != nil {
		g.Log().Error(ctx, "拉取失败:", err)
		fmt.Println("拉取失败:", err)
		os.Exit(1)
	}

	fmt.Printf("获取到 %d 个池子（已过滤仅含 WETH）\n\n", len(pools))
	if len(pools) > 0 {
		fmt.Println("--- 第 1 条预览 ---")
		fmt.Println(service.FormatPoolMessage(pools[0]))
		if len(pools) > 1 {
			fmt.Println("--- 第 2 条预览 ---")
			fmt.Println(service.FormatPoolMessage(pools[1]))
		}
	}
	fmt.Println("测试完成")
}
