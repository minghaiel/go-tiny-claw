package main

import (
	"context"
	"log"
	"os"

	ctxpkg "github.com/minghaiel/go-tiny-claw/internal/context"
	"github.com/minghaiel/go-tiny-claw/internal/engine"
	"github.com/minghaiel/go-tiny-claw/internal/provider"
	schema "github.com/minghaiel/go-tiny-claw/internal/scheme"
	"github.com/minghaiel/go-tiny-claw/internal/tools"
)

func main() {
	if os.Getenv("ZHIPU_API_KEY") == "" {
		log.Fatal("请先导出 ZHIPU_API_KEY 环境变量")
	}

	llmProvider := provider.NewZhipuOpenAIProvider("glm-4.5-air") // 智谱或 Claude

	workDir, _ := os.Getwd()
	registry := tools.NewRegistry()
	registry.Register(tools.NewReadFileTool(workDir))
	registry.Register(tools.NewWriteFileTool(workDir))
	registry.Register(tools.NewBashTool(workDir))

	// 实例化引擎 (关闭思考模式以提速)
	eng := engine.NewAgentEngine(llmProvider, registry, false)
	reporter := engine.NewTerminalReporter()

	sessionID := "test_oom_protection_001"
	sess := ctxpkg.GlobalSessionMgr.GetOrCreate(sessionID, workDir)

	// 发起一个会导致读取大文件的恶意任务
	prompt := `
    请帮我执行以下三个步骤：
    1. 使用 bash 执行 echo "开始排查日志"
    2. 使用 read_file 工具读取当前目录下的巨大文件 ./tmp/mock_log.txt
    3. 使用 bash 执行 date 命令获取当前时间，并告诉我任务全部完成。
    `

	sess.Append(schema.Message{Role: schema.RoleUser, Content: prompt})

	err := eng.Run(context.Background(), sess, reporter)
	if err != nil {
		log.Fatalf("引擎运行崩溃: %v", err)
	}

	// 2. 初始化飞书 Bot 调度器
	//bot := feishu.NewFeishuBot(eng)
	//handler := httpserverext.NewEventHandlerFunc(bot.GetEventDispatcher())

	//// 3. 注册路由并启动 HTTP 服务
	//http.HandleFunc("/webhook/event", func(w http.ResponseWriter, r *http.Request) {
	//	// 飞书验证回调地址时会 POST 一个 JSON：{"challenge":"xxx"}
	//	// 需要原样返回 challenge 才能通过校验
	//	body, _ := io.ReadAll(r.Body)
	//	r.Body.Close()
	//
	//	var req map[string]interface{}
	//	if err := json.Unmarshal(body, &req); err == nil {
	//		if ch, ok := req["challenge"]; ok {
	//			log.Println("[Feishu] 响应 Challenge 校验")
	//			w.Header().Set("Content-Type", "application/json")
	//			json.NewEncoder(w).Encode(map[string]interface{}{
	//				"challenge": ch,
	//			})
	//			return
	//		}
	//	}
	//
	//	// 正常事件回调，恢复 body 后交由 SDK handler 处理
	//	r.Body = io.NopCloser(bytes.NewReader(body))
	//	handler(w, r)
	//})

	//// 3.1 健康检查接口，供探活 / 负载均衡使用
	//http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
	//	w.Header().Set("Content-Type", "application/json")
	//	w.WriteHeader(http.StatusOK)
	//	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	//})
	//
	//// ListenAndServe 要求地址格式为 host:port；":8888" 表示监听本机所有网卡的 8888 端口
	//addr := ":8888"
	//log.Printf("🚀 go-tiny-claw 飞书服务端已启动，正在监听 %s\n", addr)
	//
	//err := http.ListenAndServe(addr, nil)
	//if err != nil {
	//	log.Fatalf("服务器启动失败: %v", err)
	//}
}
