package tools

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pingjie/educlaw/pkg/cron"
)

// ---- ScheduleReminderTool ----

// ScheduleReminderTool lets agents schedule future messages for an actor.
type ScheduleReminderTool struct {
	cronSvc   *cron.Service
	actorID   string
	actorType string
}

// NewScheduleReminderTool creates a ScheduleReminderTool.
func NewScheduleReminderTool(cronSvc *cron.Service, actorID, actorType string) *ScheduleReminderTool {
	return &ScheduleReminderTool{cronSvc: cronSvc, actorID: actorID, actorType: actorType}
}

func (t *ScheduleReminderTool) Name() string { return "schedule_reminder" }
func (t *ScheduleReminderTool) Description() string {
	return `Schedule a future reminder message for this student/user.
Schedule types:
- "at" + run_at (e.g. "2026-03-15 16:00") — one-time reminder
- "every" + interval_minutes (e.g. 1440 for daily) — repeating reminder
- "cron" + cron_expr (e.g. "0 16 * * *" for daily at 4pm) — cron schedule`
}
func (t *ScheduleReminderTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{
				"type":        "string",
				"description": "Short name for this reminder (e.g. '每日数学练习')",
			},
			"message": map[string]any{
				"type":        "string",
				"description": "The reminder message to send",
			},
			"schedule_type": map[string]any{
				"type":        "string",
				"enum":        []string{"at", "every", "cron"},
				"description": "How to schedule: 'at' (one-time), 'every' (interval), 'cron' (expression)",
			},
			"run_at": map[string]any{
				"type":        "string",
				"description": "For 'at': datetime string like '2026-03-15 16:00' (local time)",
			},
			"interval_minutes": map[string]any{
				"type":        "integer",
				"description": "For 'every': interval in minutes (e.g. 1440 = daily, 60 = hourly)",
			},
			"cron_expr": map[string]any{
				"type":        "string",
				"description": "For 'cron': standard 5-field cron expression (e.g. '0 16 * * *' = daily 4pm)",
			},
		},
		"required": []string{"name", "message", "schedule_type"},
	}
}

func (t *ScheduleReminderTool) Execute(_ context.Context, args map[string]any) (string, error) {
	name, _ := args["name"].(string)
	message, _ := args["message"].(string)
	schedType, _ := args["schedule_type"].(string)

	if name == "" || message == "" || schedType == "" {
		return "", fmt.Errorf("name, message, and schedule_type are required")
	}

	var sch cron.Schedule
	sch.Kind = schedType

	switch schedType {
	case cron.KindAt:
		runAt, _ := args["run_at"].(string)
		if runAt == "" {
			return "", fmt.Errorf("run_at is required for schedule_type 'at'")
		}
		t, err := parseDateTime(runAt)
		if err != nil {
			return "", fmt.Errorf("invalid run_at %q: %w", runAt, err)
		}
		ms := t.UnixMilli()
		sch.AtMS = &ms

	case cron.KindEvery:
		minutes, ok := args["interval_minutes"].(float64)
		if !ok || minutes <= 0 {
			return "", fmt.Errorf("interval_minutes must be a positive integer for 'every'")
		}
		ms := int64(minutes) * 60 * 1000
		sch.EveryMS = &ms

	case cron.KindCron:
		expr, _ := args["cron_expr"].(string)
		if expr == "" {
			return "", fmt.Errorf("cron_expr is required for schedule_type 'cron'")
		}
		sch.Expr = expr

	default:
		return "", fmt.Errorf("unknown schedule_type %q", schedType)
	}

	payload := cron.Payload{
		Message:   message,
		ActorID:   t.actorID,
		ActorType: t.actorType,
	}

	job, err := t.cronSvc.AddJob(name, sch, payload)
	if err != nil {
		return "", fmt.Errorf("scheduling reminder: %w", err)
	}

	desc := schedDesc(job)
	return fmt.Sprintf("已设置提醒「%s」(%s)，ID: %s", name, desc, job.ID), nil
}

// ---- ListRemindersTool ----

// ListRemindersTool shows the actor's scheduled reminders.
type ListRemindersTool struct {
	cronSvc *cron.Service
	actorID string
}

// NewListRemindersTool creates a ListRemindersTool.
func NewListRemindersTool(cronSvc *cron.Service, actorID string) *ListRemindersTool {
	return &ListRemindersTool{cronSvc: cronSvc, actorID: actorID}
}

func (t *ListRemindersTool) Name() string        { return "list_reminders" }
func (t *ListRemindersTool) Description() string  { return "List all scheduled reminders for this user." }
func (t *ListRemindersTool) Parameters() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{}}
}

func (t *ListRemindersTool) Execute(_ context.Context, _ map[string]any) (string, error) {
	jobs := t.cronSvc.ListJobsForActor(t.actorID)
	if len(jobs) == 0 {
		return "没有设置任何提醒。", nil
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("已设置 %d 个提醒：\n\n", len(jobs)))
	for _, j := range jobs {
		status := "✓ 启用"
		if !j.Enabled {
			status = "✗ 已停用"
		}
		fmt.Fprintf(&sb, "- **%s** (%s) %s\n  消息: %s\n  ID: %s\n\n",
			j.Name, schedDesc(&j), status, j.Payload.Message, j.ID)
	}
	return sb.String(), nil
}

// ---- CancelReminderTool ----

// CancelReminderTool removes a scheduled reminder.
type CancelReminderTool struct {
	cronSvc *cron.Service
}

// NewCancelReminderTool creates a CancelReminderTool.
func NewCancelReminderTool(cronSvc *cron.Service) *CancelReminderTool {
	return &CancelReminderTool{cronSvc: cronSvc}
}

func (t *CancelReminderTool) Name() string        { return "cancel_reminder" }
func (t *CancelReminderTool) Description() string  { return "Cancel a scheduled reminder by its ID." }
func (t *CancelReminderTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"id": map[string]any{
				"type":        "string",
				"description": "The reminder ID to cancel",
			},
		},
		"required": []string{"id"},
	}
}

func (t *CancelReminderTool) Execute(_ context.Context, args map[string]any) (string, error) {
	id, _ := args["id"].(string)
	if id == "" {
		return "", fmt.Errorf("id is required")
	}
	if t.cronSvc.RemoveJob(id) {
		return fmt.Sprintf("提醒 %s 已取消。", id), nil
	}
	return fmt.Sprintf("未找到提醒 %s。", id), nil
}

// helpers

func schedDesc(j *cron.Job) string {
	switch j.Schedule.Kind {
	case cron.KindAt:
		if j.Schedule.AtMS != nil {
			return "一次性 @ " + time.UnixMilli(*j.Schedule.AtMS).Format("2006-01-02 15:04")
		}
	case cron.KindEvery:
		if j.Schedule.EveryMS != nil {
			d := time.Duration(*j.Schedule.EveryMS) * time.Millisecond
			return "每 " + d.String()
		}
	case cron.KindCron:
		return "定时 " + j.Schedule.Expr
	}
	return j.Schedule.Kind
}

var dateLayouts = []string{
	"2006-01-02 15:04",
	"2006-01-02 15:04:05",
	"2006-01-02T15:04",
	"2006-01-02T15:04:05",
}

func parseDateTime(s string) (time.Time, error) {
	for _, layout := range dateLayouts {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse %q, use format 'YYYY-MM-DD HH:MM'", s)
}
