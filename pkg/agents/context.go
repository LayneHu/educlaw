package agents

import (
	"fmt"
	"strings"

	"github.com/pingjie/educlaw/pkg/skills"
	"github.com/pingjie/educlaw/pkg/workspace"
)

// ContextBuilder assembles system prompts for agents.
type ContextBuilder struct {
	wm           *workspace.Manager
	actorDir     string
	agentDir     string
	skillsLoader *skills.Loader
}

// NewContextBuilder creates a new ContextBuilder using the provided skills loader.
// The caller is responsible for the loader's lifecycle; the same loader instance
// should be shared with the tool registry to avoid duplicate directory scans.
func NewContextBuilder(wm *workspace.Manager, actorDir, agentDir string, skillsLoader *skills.Loader) *ContextBuilder {
	return &ContextBuilder{
		wm:           wm,
		actorDir:     actorDir,
		agentDir:     agentDir,
		skillsLoader: skillsLoader,
	}
}

const baseSystemPrompt = `# 角色定位
你是知知，一个专为中小学生设计的 AI 学习伙伴。你热情、耐心，善于用生动有趣的方式讲解知识，能够自主执行各种创作任务。

# 核心原则：收到指令后立即执行，不询问确认

# render_content 工具 —— 必须调用的场景
下列任何请求，**必须**调用 render_content，禁止用文字代替：
- 游戏、互动练习、动画
- 漫画、绘本、故事图文
- 视频推荐卡片
- 知识可视化、思维导图、图表
- 学习报告、分析报告
- 测验/小测试

render_content 参数：
- type: "game" | "quiz" | "visual" | "video" | "report" | "embed"
- title: 内容标题（中文）
- content: **完整可运行的 HTML**（含 <!DOCTYPE html> 到 </html>，内联所有 CSS/JS）

---

# 各类内容生成流程

## 流程1：游戏生成
1. read_skill(skill_name="game-generator", asset_file="assets/game-template.html")
2. 找到 GAME_CONFIG 对象，替换：title/subtitle/questions（每题：{q, options, answer:索引从0起}）
3. render_content(type="game", title="...", content=完整修改后HTML)

## 流程2：漫画生成
1. read_skill(skill_name="comic-generator", asset_file="assets/comic-template.html")
2. 找到 COMIC_CONFIG 对象，替换：title 和 panels 数组（每个panel：{bg, character, speech, caption}）
   - bg 可选值："sky"/"classroom"/"night"/"nature"/"space"
   - character 使用 emoji（如"🧒"/"🤖"/"👩‍🏫"/"🦁"）
   - speech 是对话气泡文字
3. render_content(type="visual", title="...", content=完整修改后HTML)

## 流程3：读取目录文件并生成分析报告
1. list_workspace_files() — 列出工作区所有文件
2. 逐个 read_workspace_file 读取相关文件（KNOWLEDGE.md, ERRORS.md, PROFILE.md 等）
3. query_knowledge() — 获取结构化知识掌握数据
4. 生成完整的 HTML 报告（包含各学科掌握度进度条、优势/薄弱点、建议）
5. render_content(type="report", title="学习分析报告", content=HTML)

## 流程4：视频推荐
1. 确定主题、年级
2. 直接生成精美 HTML 视频推荐卡片（包含 Bilibili/YouTube 搜索链接、视频标题建议、观看要点）
3. render_content(type="video", title="...", content=HTML)

## 流程5：知识可视化
1. 直接生成完整 HTML（思维导图/流程图/对比表格，用 CSS/SVG 实现）
2. render_content(type="visual", title="...", content=HTML)

---

# HTML 生成规范
- 完整独立可运行，不依赖外部文件
- 中文界面，颜色鲜艳，字体清晰
- 移动端友好（viewport meta 标签）
- 交互内容有清晰操作说明

# 其他工具
- list_workspace_files: 列出工作区文件（分析前先调用）
- read_workspace_file: 读取指定文件
- write_workspace_file: 更新学生档案
- add_daily_note: 对话结束记录学习日志
- record_answer: 记录答题结果
- query_knowledge: 查询知识掌握度
- read_skill: 读取技能模板（可用技能见 Available Skills 列表）
- web_search: 搜索网络获取最新教育内容、知识解释
- web_fetch: 获取指定网页内容（如维基百科、百科全书）
- schedule_reminder: 为学生设置定时提醒（如每日练习、周期复习）
- list_reminders: 查看已设置的提醒
- cancel_reminder: 取消某个提醒

# 执行规则
1. 收到任务立即按流程执行，无需反复确认
2. 游戏/漫画必须先 read_skill 获取模板再修改
3. 报告类必须先 list/read 文件获取真实数据
4. render_content 之后用一句话告知用户
5. 结束时调用 add_daily_note 记录本次学习`

// Build assembles the system prompt for the given actor and agent types.
// agentType selects the role-specific prompt fragment (e.g. "tutor", "analyst").
func (cb *ContextBuilder) Build(actorType, agentType string) string {
	var parts []string

	// 0. Always include base system prompt
	parts = append(parts, baseSystemPrompt)

	// 0b. Agent-specific role fragment (tutor / analyst / companion / etc.)
	if fragment := FragmentFor(agentType); fragment != "" {
		parts = append(parts, fragment)
	}

	// 1. Read agent principles (AGENTS.md)
	if agentsMD := cb.wm.ReadFile(cb.agentDir, "AGENTS.md"); agentsMD != "" {
		parts = append(parts, "# Teaching Principles\n"+agentsMD)
	}

	// 2. Read agent soul (SOUL.md)
	if soulMD := cb.wm.ReadFile(cb.agentDir, "SOUL.md"); soulMD != "" {
		parts = append(parts, "# Personality\n"+soulMD)
	}

	// 3. Actor-type-specific files
	switch actorType {
	case "student":
		parts = append(parts, cb.buildStudentContext()...)
	case "family", "parent":
		parts = append(parts, cb.buildFamilyContext()...)
	case "teacher":
		parts = append(parts, cb.buildTeacherContext()...)
	}

	// 4. Long-term memory (MEMORY.md)
	if memory := cb.wm.ReadMemory(cb.actorDir); memory != "" {
		parts = append(parts, "# Long-Term Memory\n"+memory)
	}

	// 5. Recent learning journal (last 3 days)
	if recentNotes := cb.wm.GetRecentDailyNotes(cb.actorDir, 3); recentNotes != "" {
		parts = append(parts, "# Recent Learning Journal\n"+recentNotes)
	}

	// 6. Skills summary
	if summary := cb.skillsLoader.BuildSummary(); summary != "" {
		parts = append(parts, fmt.Sprintf("# Available Skills\n%s", summary))
	}

	return strings.Join(parts, "\n\n---\n\n")
}

func (cb *ContextBuilder) buildStudentContext() []string {
	var parts []string
	files := []struct{ label, file string }{
		{"# Student Profile", "PROFILE.md"},
		{"# Knowledge States", "KNOWLEDGE.md"},
		{"# Student Interests", "INTERESTS.md"},
		{"# Common Errors", "ERRORS.md"},
		{"# Student Soul/Preferences", "SOUL.md"},
	}
	for _, f := range files {
		if content := cb.wm.ReadFile(cb.actorDir, f.file); content != "" {
			parts = append(parts, f.label+"\n"+content)
		}
	}
	return parts
}

func (cb *ContextBuilder) buildFamilyContext() []string {
	var parts []string
	files := []struct{ label, file string }{
		{"# Family Context", "CONTEXT.md"},
		{"# Educational Goals", "GOALS.md"},
	}
	for _, f := range files {
		if content := cb.wm.ReadFile(cb.actorDir, f.file); content != "" {
			parts = append(parts, f.label+"\n"+content)
		}
	}
	return parts
}

func (cb *ContextBuilder) buildTeacherContext() []string {
	var parts []string
	files := []struct{ label, file string }{
		{"# Teacher Profile", "PROFILE.md"},
		{"# Classes", "CLASSES.md"},
	}
	for _, f := range files {
		if content := cb.wm.ReadFile(cb.actorDir, f.file); content != "" {
			parts = append(parts, f.label+"\n"+content)
		}
	}
	return parts
}
