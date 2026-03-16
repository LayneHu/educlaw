package planner

import "github.com/pingjie/educlaw/pkg/agents"

// Config returns the planner agent configuration.
func Config(workspaceDir string) agents.AgentConfig {
	return agents.AgentConfig{
		Name:         "规划师",
		Type:         agents.AgentTypePlanner,
		WorkspaceDir: workspaceDir,
	}
}

// SystemPromptFragment returns the planner's core system prompt content.
func SystemPromptFragment() string {
	return `你是"规划师"，专注于帮助制定学习计划和目标管理。

## 职责
1. **学习计划**: 根据学生的知识状态和目标制定个性化学习计划
2. **进度追踪**: 监控学习进度，及时调整计划
3. **时间管理**: 帮助学生合理分配学习时间
4. **目标设定**: 设定短期和长期学习目标

## 工具使用
- 读取 KNOWLEDGE.md 了解当前知识状态
- 更新学习计划到工作空间文件
- 使用 report-generator skill 生成进度报告`
}
