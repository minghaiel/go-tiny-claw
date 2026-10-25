package main

import (
	"context"
	"log"
	"os"
	"sync"
	"time"

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

	registry := tools.NewRegistry()
	registry.Register(tools.NewReadFileTool("/tmp/project_front"))

	// 引擎本身变成无状态的，它不绑定 WorkDir（仅适用于本讲演示）
	eng := engine.NewAgentEngine(llmProvider, registry, false)
	reporter := engine.NewTerminalReporter()

	var wg sync.WaitGroup

	// ================= 模拟并发场景 1：飞书前端群 =================
	wg.Add(1)
	go func() {
		defer wg.Done()
		sessionA := ctxpkg.GlobalSessionMgr.GetOrCreate("chat_front_001", "/tmp/project_front")

		// 回合 1：获取机密
		log.Println("\n>>> 🙋‍♂️ [Session A / Turn 1]: 帮我看看 README.md 里记录了什么密钥？")
		sessionA.Append(schema.Message{Role: schema.RoleUser, Content: "帮我看看 README.md 里记录了什么密钥？"})
		_ = eng.Run(context.Background(), sessionA, reporter)

		// 故意制造大量“废话”对话，刷掉记忆 (假设 Working Memory Limit=6)
		for i := 0; i < 6; i++ {
			sessionA.Append(schema.Message{Role: schema.RoleUser, Content: "这只是一句闲聊占位符。"})
			sessionA.Append(schema.Message{Role: schema.RoleAssistant, Content: "好的，收到闲聊。"})
		}

		// 回合 2：验证记忆截断 (此时第一轮的密钥已经被挤出 Working Memory 了！)
		log.Println("\n>>> 🙋‍♂️ [Session A / Turn 2]: 请直接告诉我，刚才第一轮你查到的那个密钥是什么？")
		sessionA.Append(schema.Message{Role: schema.RoleUser, Content: "请直接告诉我，刚才第一轮你查到的那个密钥是什么？不准调用工具！"})
		_ = eng.Run(context.Background(), sessionA, reporter)
	}()

	// ================= 模拟并发场景 2：飞书后端群 =================
	wg.Add(1)
	go func() {
		defer wg.Done()
		// 稍微错开一点时间发起请求
		time.Sleep(1 * time.Second)

		sessionB := ctxpkg.GlobalSessionMgr.GetOrCreate("chat_back_002", "/tmp/project_back")

		log.Println("\n>>> 🙋‍♂️ [Session B]: 别人查到了一个密钥，你这里能看到吗？")
		sessionB.Append(schema.Message{Role: schema.RoleUser, Content: "别人查到了一个密钥，你这里能看到吗？不准调用工具！"})
		_ = eng.Run(context.Background(), sessionB, reporter)
	}()

	wg.Wait()

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
