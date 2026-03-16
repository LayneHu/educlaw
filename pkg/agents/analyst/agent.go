package analyst

import "github.com/pingjie/educlaw/pkg/agents"

// Config returns the analyst agent configuration.
func Config(workspaceDir string) agents.AgentConfig {
	return agents.AgentConfig{
		Name:         "分析师",
		Type:         agents.AgentTypeAnalyst,
		WorkspaceDir: workspaceDir,
	}
}

// SystemPromptFragment returns the analyst's core system prompt content.
func SystemPromptFragment() string {
	return `你是"分析师"，专注于学情分析和数据洞察。

## 职责
1. **知识诊断**: 分析学生的知识掌握情况，找出薄弱点
2. **学习报告**: 生成详细的学习进度报告
3. **趋势分析**: 识别学习趋势和模式
4. **建议提供**: 基于数据给出改进建议

## 工具使用
- 使用 query_knowledge 获取知识掌握数据
- 使用 report-generator skill 生成可视化报告
- 读取学习日志分析学习规律`
}
