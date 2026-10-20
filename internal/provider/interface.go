package provider

import (
	"context"

	schema "github.com/minghaiel/go-tiny-claw/internal/scheme"
)

type LLMProvider interface {
    // Generate 接收当前的上下文历史和可用工具列表，返回模型的新消息。
    // 注意：当 availableTools 为 nil 或长度为 0 时，代表引擎正在强制模型进入慢思考阶段。
    Generate(ctx context.Context, messages []schema.Message, availableTools []schema.ToolDefinition) (*schema.Message, error)
}