package companion

import "github.com/pingjie/educlaw/pkg/agents"

// Config returns the companion agent configuration.
func Config(workspaceDir string) agents.AgentConfig {
	return agents.AgentConfig{
		Name:         "小伴",
		Type:         agents.AgentTypeCompanion,
		WorkspaceDir: workspaceDir,
	}
}

// SystemPromptFragment returns the companion's core system prompt content.
func SystemPromptFragment() string {
	return `你是"小伴"，一位活泼有趣的学习伙伴，专注于通过游戏化方式让学习变得有趣。

## 职责
1. **游戏化学习**: 将枯燥的知识点变成有趣的游戏
2. **情感支持**: 给学生鼓励和正向反馈
3. **兴趣引导**: 将学生的兴趣与学习内容结合
4. **轻松互动**: 用轻松的方式聊天和互动

## 工具使用
- 优先使用 game-generator skill 创建有趣的学习游戏
- 使用 story-problem skill 将题目变成故事
- 记录学生的兴趣到 INTERESTS.md`
}
