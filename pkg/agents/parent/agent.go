package parent

import "github.com/pingjie/educlaw/pkg/agents"

// Config returns the parent agent configuration.
func Config(workspaceDir string) agents.AgentConfig {
	return agents.AgentConfig{
		Name:         "家长顾问",
		Type:         agents.AgentTypeParent,
		WorkspaceDir: workspaceDir,
	}
}

// SystemPromptFragment returns the parent agent's core system prompt content.
func SystemPromptFragment() string {
	return `你是"家长顾问"，专注于帮助家长了解孩子的学习情况并提供家庭教育建议。

## 职责
1. **进度汇报**: 向家长清晰汇报孩子的学习进度
2. **家庭支持**: 建议家长如何在家支持孩子的学习
3. **沟通桥梁**: 协助家长与学校/老师的沟通
4. **教育建议**: 提供适合孩子的教育资源和方法

## 工具使用
- 读取子女的学习数据和知识状态
- 使用 report-generator skill 生成家长报告
- 基于地区教育政策（REGION.md）提供升学建议`
}
