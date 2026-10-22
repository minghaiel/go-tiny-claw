// cmd/claw/main.go
package main

import (
	"context"
	"log"
	"os"

	"github.com/minghaiel/go-tiny-claw/internal/engine"
	"github.com/minghaiel/go-tiny-claw/internal/provider"
	"github.com/minghaiel/go-tiny-claw/internal/tools"
)

func main() {
    // 确保设置了 ZHIPU_API_KEY
    if os.Getenv("ZHIPU_API_KEY") == "" {
        log.Fatal("请先导出 ZHIPU_API_KEY 环境变量")
    }

    // 1. 获取工作区物理边界
    workDir, _ := os.Getwd()

    // 2. 初始化真实的大脑 (指向智谱 GLM-4.5，使用上一讲的 OpenAI 适配器)
    llmProvider := provider.NewZhipuOpenAIProvider("glm-4.5-air")

    // 3. 初始化真实的 Tool Registry
    registry := tools.NewRegistry()

    // 4. 挂载极简工具集
    registry.Register(tools.NewReadFileTool(workDir))
    registry.Register(tools.NewWriteFileTool(workDir))
    registry.Register(tools.NewBashTool(workDir))
    registry.Register(tools.NewEditFileTool(workDir))

    // 5. 实例化核心引擎，由于任务简单，关闭思考阶段 (EnableThinking = false) 以加快速度
    eng := engine.NewAgentEngine(llmProvider, registry, workDir, false)

   // 6. 发起一个需要局部修改的指令
    // prompt := `
    // 我当前目录下有一个 server.go 文件。
    // 请帮我把里面 "TODO: 增加鉴权逻辑" 下面的那个 if 语句，整个替换为：
    // if user == nil {
    //     fmt.Println("Forbidden!")
    //     return
    // }
    // `
    prompt := `读取cmd/claw/main.go文件，并分析main.go的逻辑`


    err := eng.Run(context.Background(), prompt)
    if err != nil {
        log.Fatalf("引擎运行崩溃: %v", err)
    }
}