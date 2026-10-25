// Package engine 实现 go-tiny-claw 的核心执行引擎：驱动 LLM 完成思考与行动阶段，
// 并以受限的并发度执行模型请求的工具调用。
package engine

import (
	"context"
	"fmt"
	"log"
	"sync"

	ctxpkg "github.com/minghaiel/go-tiny-claw/internal/context"
	"github.com/minghaiel/go-tiny-claw/internal/provider"
	schema "github.com/minghaiel/go-tiny-claw/internal/scheme"
	"github.com/minghaiel/go-tiny-claw/internal/tools"
)

// maxToolConcurrency 限制单轮工具调用的最大并行度
const maxToolConcurrency = 20

// AgentEngine 是微型 OS 的核心驱动
type AgentEngine struct {
	provider       provider.LLMProvider
	registry       tools.Registry
	EnableThinking bool // 慢思考模式开关

	compactor *ctxpkg.Compactor // 压缩器实例
}

func NewAgentEngine(p provider.LLMProvider, r tools.Registry, enableThinking bool) *AgentEngine {
	return &AgentEngine{
		provider:       p,
		registry:       r,
		EnableThinking: enableThinking,
		// 【初始化压缩器】：为了便于今天的极端测试，我们将水位线阈值设积极（例如 3000 字符），
		// 并保护最近的 6 条消息（大约两轮 Turn 的交互）
		compactor: ctxpkg.NewCompactor(3000, 6),
	}
}

// Run 启动 Agent 的生命周期
func (e *AgentEngine) Run(ctx context.Context, session *ctxpkg.Session, reporter Reporter) error {
	log.Printf("[Engine] 唤醒会话 [%s]，锁定工作区: %s\n", session.ID, session.WorkDir)

	// 动态组装 System Prompt
	composer := ctxpkg.NewPromptComposer(session.WorkDir)
	systemMsg := composer.Build()

	for {
		// 获取当前挂载的所有工具定义
		availableTools := e.registry.GetAvailableTools()

		// 1. 从 Session 提取出近期的 Working Memory
		workingMemory := session.GetWorkingMemory(20)

		var contextHistory []schema.Message
		contextHistory = append(contextHistory, systemMsg)
		contextHistory = append(contextHistory, workingMemory...)

		// 在向 Provider 发起推理前，过一遍内存压缩器！
		// 无论你带出了多少上下文，如果字符总数超标，早期日志将被掩码化，超大日志将被掐头去尾
		compactedContext := e.compactor.Compact(contextHistory)

		// ====================================================================
		// Phase 2: 慢思考阶段 (Thinking) - 剥夺工具，强制规划
		// ====================================================================
		if e.EnableThinking {
			if reporter != nil {
				reporter.OnThinking(ctx)
			}

			// 核心机制：传入的 availableTools 为 nil！
			// 大模型看不到任何 JSON Schema，被迫只能输出纯文本的思考过程。
			thinkResp, err := e.provider.Generate(ctx, compactedContext, nil)
			if err != nil {
				return fmt.Errorf("Thinking 阶段生成失败: %w\n", err)
			}

			// 如果模型输出了思考过程，我们将其作为 Assistant 消息追加到上下文中
			if thinkResp.Content != "" {
				// 将思考持久化到 Session 中
				session.Append(*thinkResp)
				compactedContext = append(compactedContext, *thinkResp)
			}
		}

		// ====================================================================
		// Phase 3: 行动阶段 (Action) - 恢复工具，顺着规划执行
		// ====================================================================
		actionResp, err := e.provider.Generate(ctx, compactedContext, availableTools)
		if err != nil {
			return fmt.Errorf("Action 阶段生成失败: %w\n", err)
		}

		// 将大模型的行动响应持久化到 Session 中
		session.Append(*actionResp)
		compactedContext = append(compactedContext, *actionResp)

		if actionResp.Content != "" && reporter != nil {
			reporter.OnMessage(ctx, actionResp.Content)
		}

		if len(actionResp.ToolCalls) == 0 {
			// 如果没有工具调用，说明本次任务已完成，打破 ReAct 循环，挂起等待下一条指令
			break
		}

		// ====================================================================
		// Phase 4: 并发执行底层工具
		// ====================================================================
		observationMsgs := make([]schema.Message, len(actionResp.ToolCalls))
		var wg sync.WaitGroup
		sem := make(chan struct{}, maxToolConcurrency)
		for i, toolCall := range actionResp.ToolCalls {
			sem <- struct{}{} // 获取令牌；在途协程达到上限时此处阻塞
			wg.Add(1)         // 增加计数器

			// 开启协程。注意：一定要将索引 i 和 toolCall 作为参数传入匿名函数，防止闭包变量捕获陷阱！
			go func(idx int, call schema.ToolCall) {
				defer wg.Done()          // 协程结束时计数器减一
				defer func() { <-sem }() // 释放令牌

				if reporter != nil {
					reporter.OnToolCall(ctx, call.Name, string(call.Arguments))
				}

				result := e.registry.Execute(ctx, call)

				if reporter != nil {
					displayOutput := result.Output
					if len(displayOutput) > 200 {
						displayOutput = displayOutput[:200] + "... (已截断)"
					}
					reporter.OnToolResult(ctx, call.Name, displayOutput, result.IsError)
				}

				observationMsgs[idx] = schema.Message{
					Role:       schema.RoleUser,
					Content:    result.Output,
					ToolCallID: call.ID,
				}

			}(i, toolCall)
		}

		wg.Wait()
		close(sem)

		// 将所有的工具执行结果（Observation）持久化到 Session 中，开启下一轮的复盘与推理
		session.Append(observationMsgs...)
	}

	return nil
}
