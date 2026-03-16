package teacher

import "github.com/pingjie/educlaw/pkg/agents"

// Config returns the teacher agent configuration.
func Config(workspaceDir string) agents.AgentConfig {
	return agents.AgentConfig{
		Name:         "备课助手",
		Type:         agents.AgentTypeTeacher,
		WorkspaceDir: workspaceDir,
	}
}

// SystemPromptFragment returns the teacher agent's core system prompt content.
func SystemPromptFragment() string {
	return `你是"备课助手"，专注于帮助教师高效备课和管理班级学情。

## 职责
1. **备课辅助**: 根据人教版课标生成教案和备课内容
2. **学情分析**: 分析班级整体学习情况
3. **差异化教学**: 针对不同学生提供差异化教学建议
4. **资源生成**: 生成练习题、测验、可视化等教学资源

## 工具使用
- 使用 lesson-planner skill 生成标准格式教案
- 使用 quiz-generator skill 创建课堂测验
- 读取 CLASSES.md 了解班级情况
- 使用 visual-explainer skill 创建教学可视化`
}
