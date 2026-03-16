package orchestrator

import "strings"

// Route determines which agent should handle the message based on actor type and content.
func Route(actorType, content string) string {
	lower := strings.ToLower(content)

	switch actorType {
	case "teacher":
		if contains(lower, "lesson", "plan", "curriculum", "备课", "教案") {
			return "planner"
		}
		if contains(lower, "class", "student", "grade", "学情", "成绩") {
			return "analyst"
		}
		return "teacher"

	case "family", "parent":
		if contains(lower, "progress", "report", "score", "进度", "报告", "成绩") {
			return "analyst"
		}
		if contains(lower, "plan", "goal", "schedule", "计划", "目标") {
			return "planner"
		}
		return "parent"

	case "student":
		if contains(lower, "plan", "schedule", "today", "计划", "今天", "安排") {
			return "planner"
		}
		if contains(lower, "game", "play", "fun", "游戏", "玩", "有趣") {
			return "companion"
		}
		if contains(lower, "quiz", "test", "exam", "测验", "考试") {
			return "tutor"
		}
		return "tutor"
	}

	return "tutor"
}

func contains(s string, keywords ...string) bool {
	for _, kw := range keywords {
		if strings.Contains(s, kw) {
			return true
		}
	}
	return false
}
