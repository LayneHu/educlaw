package tools

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pingjie/educlaw/pkg/bus"
	"github.com/pingjie/educlaw/pkg/skills"
	"github.com/pingjie/educlaw/pkg/storage"
	"github.com/pingjie/educlaw/pkg/workspace"
)

// ---- ReadWorkspaceTool ----

// ReadWorkspaceTool reads a file from a workspace directory.
type ReadWorkspaceTool struct {
	wm  *workspace.Manager
	dir string
}

// NewReadWorkspaceTool creates a new ReadWorkspaceTool for a specific directory.
func NewReadWorkspaceTool(wm *workspace.Manager, dir string) *ReadWorkspaceTool {
	return &ReadWorkspaceTool{wm: wm, dir: dir}
}

func (t *ReadWorkspaceTool) Name() string { return "read_workspace_file" }
func (t *ReadWorkspaceTool) Description() string {
	return "Read a file from the student's workspace directory. Use to access PROFILE.md, KNOWLEDGE.md, INTERESTS.md, ERRORS.md, etc."
}
func (t *ReadWorkspaceTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"filename": map[string]any{
				"type":        "string",
				"description": "The filename to read (e.g., KNOWLEDGE.md, PROFILE.md)",
			},
		},
		"required": []string{"filename"},
	}
}
func (t *ReadWorkspaceTool) Execute(_ context.Context, args map[string]any) (string, error) {
	filename, _ := args["filename"].(string)
	if filename == "" {
		return "", fmt.Errorf("filename is required")
	}
	content := t.wm.ReadFile(t.dir, filename)
	if content == "" {
		return fmt.Sprintf("File %s not found or empty.", filename), nil
	}
	return content, nil
}

// ---- WriteWorkspaceTool ----

// WriteWorkspaceTool writes a file to a workspace directory.
type WriteWorkspaceTool struct {
	wm  *workspace.Manager
	dir string
}

// NewWriteWorkspaceTool creates a new WriteWorkspaceTool for a specific directory.
func NewWriteWorkspaceTool(wm *workspace.Manager, dir string) *WriteWorkspaceTool {
	return &WriteWorkspaceTool{wm: wm, dir: dir}
}

func (t *WriteWorkspaceTool) Name() string { return "write_workspace_file" }
func (t *WriteWorkspaceTool) Description() string {
	return "Write or update a file in the student's workspace directory. Use to update KNOWLEDGE.md, ERRORS.md, etc."
}
func (t *WriteWorkspaceTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"filename": map[string]any{
				"type":        "string",
				"description": "The filename to write",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "The content to write to the file",
			},
		},
		"required": []string{"filename", "content"},
	}
}
func (t *WriteWorkspaceTool) Execute(_ context.Context, args map[string]any) (string, error) {
	filename, _ := args["filename"].(string)
	content, _ := args["content"].(string)
	if filename == "" {
		return "", fmt.Errorf("filename is required")
	}
	if err := t.wm.WriteFile(t.dir, filename, content); err != nil {
		return "", fmt.Errorf("writing file: %w", err)
	}
	return fmt.Sprintf("Successfully wrote %s", filename), nil
}

// ---- AppendDailyTool ----

// AppendDailyTool appends a note to today's daily log.
type AppendDailyTool struct {
	wm  *workspace.Manager
	dir string
}

// NewAppendDailyTool creates a new AppendDailyTool.
func NewAppendDailyTool(wm *workspace.Manager, dir string) *AppendDailyTool {
	return &AppendDailyTool{wm: wm, dir: dir}
}

func (t *AppendDailyTool) Name() string { return "add_daily_note" }
func (t *AppendDailyTool) Description() string {
	return "Append a note to today's learning journal. Use at the end of each conversation to record what was learned."
}
func (t *AppendDailyTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"note": map[string]any{
				"type":        "string",
				"description": "The note to append to today's log",
			},
		},
		"required": []string{"note"},
	}
}
func (t *AppendDailyTool) Execute(_ context.Context, args map[string]any) (string, error) {
	note, _ := args["note"].(string)
	if note == "" {
		return "", fmt.Errorf("note is required")
	}
	if err := t.wm.AppendDailyLog(t.dir, note); err != nil {
		return "", fmt.Errorf("appending daily log: %w", err)
	}
	return "Daily note recorded.", nil
}

// ---- RecordEventTool ----

// RecordEventTool records a learning event and updates knowledge state.
type RecordEventTool struct {
	db        *sql.DB
	studentID string
}

// NewRecordEventTool creates a new RecordEventTool.
func NewRecordEventTool(db *sql.DB, studentID string) *RecordEventTool {
	return &RecordEventTool{db: db, studentID: studentID}
}

func (t *RecordEventTool) Name() string { return "record_answer" }
func (t *RecordEventTool) Description() string {
	return "Record a student's answer to a knowledge point question. Updates learning history and knowledge mastery."
}
func (t *RecordEventTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"subject": map[string]any{
				"type":        "string",
				"description": "Subject area (e.g., math, chinese, english, science)",
			},
			"kp_id": map[string]any{
				"type":        "string",
				"description": "Knowledge point ID (e.g., math_fractions_addition)",
			},
			"kp_name": map[string]any{
				"type":        "string",
				"description": "Human-readable knowledge point name",
			},
			"is_correct": map[string]any{
				"type":        "boolean",
				"description": "Whether the student answered correctly",
			},
			"note": map[string]any{
				"type":        "string",
				"description": "Optional note about the answer",
			},
		},
		"required": []string{"subject", "kp_id", "kp_name", "is_correct"},
	}
}
func (t *RecordEventTool) Execute(_ context.Context, args map[string]any) (string, error) {
	subject, _ := args["subject"].(string)
	kpID, _ := args["kp_id"].(string)
	kpName, _ := args["kp_name"].(string)
	isCorrect, _ := args["is_correct"].(bool)
	note, _ := args["note"].(string)

	event := storage.LearningEvent{
		StudentID: t.studentID,
		Subject:   subject,
		KpID:      kpID,
		KpName:    kpName,
		IsCorrect: isCorrect,
		Note:      note,
	}
	if err := storage.RecordEvent(t.db, event); err != nil {
		return "", fmt.Errorf("recording event: %w", err)
	}

	// Update knowledge state
	correctDelta := 0
	if isCorrect {
		correctDelta = 1
	}
	ks := storage.KnowledgeState{
		StudentID:    t.studentID,
		Subject:      subject,
		KpID:         kpID,
		KpName:       kpName,
		CorrectCount: correctDelta,
		TotalCount:   1,
	}
	if err := storage.UpsertKnowledge(t.db, ks); err != nil {
		return "", fmt.Errorf("updating knowledge: %w", err)
	}

	result := "incorrect"
	if isCorrect {
		result = "correct"
	}
	return fmt.Sprintf("Recorded: %s - %s (%s)", subject, kpName, result), nil
}

// ---- QueryKnowledgeTool ----

// QueryKnowledgeTool retrieves knowledge mastery for a student.
type QueryKnowledgeTool struct {
	db        *sql.DB
	studentID string
}

// NewQueryKnowledgeTool creates a new QueryKnowledgeTool.
func NewQueryKnowledgeTool(db *sql.DB, studentID string) *QueryKnowledgeTool {
	return &QueryKnowledgeTool{db: db, studentID: studentID}
}

func (t *QueryKnowledgeTool) Name() string { return "query_knowledge" }
func (t *QueryKnowledgeTool) Description() string {
	return "Query the student's knowledge mastery levels for all subjects or a specific subject."
}
func (t *QueryKnowledgeTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"subject": map[string]any{
				"type":        "string",
				"description": "Optional: filter by subject (e.g., math, chinese)",
			},
		},
	}
}
func (t *QueryKnowledgeTool) Execute(_ context.Context, args map[string]any) (string, error) {
	states, err := storage.GetKnowledgeStates(t.db, t.studentID)
	if err != nil {
		return "", fmt.Errorf("querying knowledge: %w", err)
	}

	subject, _ := args["subject"].(string)

	var sb strings.Builder
	sb.WriteString("# Knowledge Mastery\n\n")

	currentSubject := ""
	for _, ks := range states {
		if subject != "" && ks.Subject != subject {
			continue
		}
		if ks.Subject != currentSubject {
			currentSubject = ks.Subject
			sb.WriteString(fmt.Sprintf("## %s\n", currentSubject))
		}
		mastery := ks.MasteryPercent()
		bar := strings.Repeat("█", mastery/10) + strings.Repeat("░", 10-mastery/10)
		sb.WriteString(fmt.Sprintf("- %s: %s %d%% (%d/%d)\n",
			ks.KpName, bar, mastery, ks.CorrectCount, ks.TotalCount))
	}

	if sb.Len() == 20 {
		return "No knowledge data recorded yet.", nil
	}
	return sb.String(), nil
}

// ---- RenderContentTool ----

// RenderContentTool saves rendered content and publishes it via the message bus.
type RenderContentTool struct {
	db        *sql.DB
	msgBus    *bus.MessageBus
	sessionID string
	actorID   string
}

// NewRenderContentTool creates a new RenderContentTool.
func NewRenderContentTool(db *sql.DB, msgBus *bus.MessageBus, sessionID, actorID string) *RenderContentTool {
	return &RenderContentTool{
		db:        db,
		msgBus:    msgBus,
		sessionID: sessionID,
		actorID:   actorID,
	}
}

func (t *RenderContentTool) Name() string { return "render_content" }
func (t *RenderContentTool) Description() string {
	return "Render interactive content (game, quiz, visual, embed, video, report) in the student's browser. The content will appear as an interactive card in the chat."
}
func (t *RenderContentTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"type": map[string]any{
				"type":        "string",
				"enum":        []string{"game", "quiz", "visual", "embed", "video", "report"},
				"description": "Type of content to render",
			},
			"title": map[string]any{
				"type":        "string",
				"description": "Title for the rendered content",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "Complete HTML content to render in an iframe",
			},
		},
		"required": []string{"type", "title", "content"},
	}
}
func (t *RenderContentTool) Execute(_ context.Context, args map[string]any) (string, error) {
	contentType, _ := args["type"].(string)
	title, _ := args["title"].(string)
	content, _ := args["content"].(string)

	if contentType == "" || title == "" || content == "" {
		return "", fmt.Errorf("type, title, and content are required")
	}

	rc := storage.RenderedContentRow{
		ContentType: contentType,
		Title:       title,
		Content:     content,
	}
	id, err := storage.SaveRenderedContent(t.db, t.sessionID, t.actorID, rc)
	if err != nil {
		return "", fmt.Errorf("saving rendered content: %w", err)
	}

	// Publish rendered content to the frontend via message bus
	rendered := bus.RenderedContent{
		ID:      id,
		Type:    contentType,
		Title:   title,
		Content: content,
	}
	renderedJSON, _ := json.Marshal(rendered)

	t.msgBus.Publish(bus.OutboundMessage{
		SessionID:   t.sessionID,
		ActorID:     t.actorID,
		Content:     string(renderedJSON),
		ContentType: "rendered",
		Done:        false,
	})

	return fmt.Sprintf("Rendered %s '%s' (id: %s)", contentType, title, id), nil
}

// ---- ListWorkspaceFilesTool ----

// ListWorkspaceFilesTool lists files in the actor's workspace directory.
type ListWorkspaceFilesTool struct {
	wm  *workspace.Manager
	dir string
}

// NewListWorkspaceFilesTool creates a new ListWorkspaceFilesTool.
func NewListWorkspaceFilesTool(wm *workspace.Manager, dir string) *ListWorkspaceFilesTool {
	return &ListWorkspaceFilesTool{wm: wm, dir: dir}
}

func (t *ListWorkspaceFilesTool) Name() string { return "list_workspace_files" }
func (t *ListWorkspaceFilesTool) Description() string {
	return "List files and subdirectories in the workspace. Use before read_workspace_file to discover what files are available. Essential for 'analyze my files' tasks."
}
func (t *ListWorkspaceFilesTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"subdir": map[string]any{
				"type":        "string",
				"description": "Optional subdirectory to list (e.g., 'memory/202603'). Leave empty to list the workspace root.",
			},
		},
	}
}
func (t *ListWorkspaceFilesTool) Execute(_ context.Context, args map[string]any) (string, error) {
	subdir, _ := args["subdir"].(string)
	targetDir := t.dir
	if subdir != "" {
		clean := filepath.Clean(subdir)
		if strings.Contains(clean, "..") {
			return "", fmt.Errorf("invalid path: directory traversal not allowed")
		}
		targetDir = filepath.Join(t.dir, clean)
	}

	entries, err := os.ReadDir(targetDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "Directory does not exist or is empty.", nil
		}
		return "", fmt.Errorf("reading directory: %w", err)
	}

	if len(entries) == 0 {
		return "Directory is empty.", nil
	}

	var sb strings.Builder
	label := "workspace root"
	if subdir != "" {
		label = subdir
	}
	sb.WriteString(fmt.Sprintf("Files in %s:\n", label))
	for _, e := range entries {
		if e.IsDir() {
			sb.WriteString("  [dir]  " + e.Name() + "/\n")
		} else {
			info, _ := e.Info()
			size := int64(0)
			if info != nil {
				size = info.Size()
			}
			sb.WriteString(fmt.Sprintf("  [file] %s (%d bytes)\n", e.Name(), size))
		}
	}
	return sb.String(), nil
}

// ---- ReadSkillTool ----

// ReadSkillTool reads a skill's content or asset file.
type ReadSkillTool struct {
	loader *skills.Loader
}

// NewReadSkillTool creates a new ReadSkillTool.
func NewReadSkillTool(loader *skills.Loader) *ReadSkillTool {
	return &ReadSkillTool{loader: loader}
}

func (t *ReadSkillTool) Name() string { return "read_skill" }
func (t *ReadSkillTool) Description() string {
	return "Read a skill's SKILL.md content or one of its asset files. Use to get templates for generating games, quizzes, etc."
}
func (t *ReadSkillTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"skill_name": map[string]any{
				"type":        "string",
				"description": "Name of the skill to read (e.g., game-generator, quiz-generator)",
			},
			"asset_file": map[string]any{
				"type":        "string",
				"description": "Optional: specific asset file to read (e.g., assets/game-template.html)",
			},
		},
		"required": []string{"skill_name"},
	}
}
func (t *ReadSkillTool) Execute(_ context.Context, args map[string]any) (string, error) {
	skillName, _ := args["skill_name"].(string)
	assetFile, _ := args["asset_file"].(string)

	if skillName == "" {
		return "", fmt.Errorf("skill_name is required")
	}

	if assetFile != "" {
		content, ok := t.loader.LoadSkillAsset(skillName, assetFile)
		if !ok {
			return fmt.Sprintf("Asset %s not found in skill %s", assetFile, skillName), nil
		}
		return content, nil
	}

	content, ok := t.loader.LoadSkill(skillName)
	if !ok {
		return fmt.Sprintf("Skill %s not found", skillName), nil
	}
	return content, nil
}

// ---- FindSkillsTool ----

// FindSkillsTool searches visible skills by capability keywords.
type FindSkillsTool struct {
	registry *skills.RegistryManager
}

// NewFindSkillsTool creates a new FindSkillsTool.
func NewFindSkillsTool(registry *skills.RegistryManager) *FindSkillsTool {
	return &FindSkillsTool{registry: registry}
}

func (t *FindSkillsTool) Name() string { return "find_skills" }
func (t *FindSkillsTool) Description() string {
	return "Search available skills from workspace, global, and builtin skill directories by capability or topic."
}
func (t *FindSkillsTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "What kind of skill you need, such as game, quiz, report, comic, lesson plan",
			},
			"limit": map[string]any{
				"type":        "integer",
				"description": "Maximum number of results to return. Default 5.",
			},
		},
		"required": []string{"query"},
	}
}
func (t *FindSkillsTool) Execute(_ context.Context, args map[string]any) (string, error) {
	query, _ := args["query"].(string)
	if strings.TrimSpace(query) == "" {
		return "", fmt.Errorf("query is required")
	}
	limit := 5
	if raw, ok := args["limit"].(float64); ok && int(raw) > 0 {
		limit = int(raw)
	}
	results := t.registry.SearchLocal(query, limit)
	if len(results) == 0 {
		return fmt.Sprintf("No skills found for query %q.", query), nil
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d skills for %q:\n", len(results), query))
	for _, item := range results {
		sb.WriteString(fmt.Sprintf("- %s [%s]: %s\n", item.Name, item.Source, item.Summary))
	}
	return sb.String(), nil
}

// ---- InstallSkillTool ----

// InstallSkillTool installs a skill from a GitHub repository.
type InstallSkillTool struct {
	installer *skills.Installer
}

// NewInstallSkillTool creates a new InstallSkillTool.
func NewInstallSkillTool(installer *skills.Installer) *InstallSkillTool {
	return &InstallSkillTool{installer: installer}
}

func (t *InstallSkillTool) Name() string { return "install_skill" }
func (t *InstallSkillTool) Description() string {
	return "Install a skill from a GitHub repository path in owner/repo format into the workspace skills directory."
}
func (t *InstallSkillTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"repo": map[string]any{
				"type":        "string",
				"description": "GitHub repo path in owner/repo format",
			},
			"force": map[string]any{
				"type":        "boolean",
				"description": "Whether to overwrite an existing installed skill with the same name",
			},
		},
		"required": []string{"repo"},
	}
}
func (t *InstallSkillTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	repo, _ := args["repo"].(string)
	force, _ := args["force"].(bool)
	if strings.TrimSpace(repo) == "" {
		return "", fmt.Errorf("repo is required")
	}
	name, err := t.installer.InstallFromGitHub(ctx, repo, force)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Installed skill %s from %s.", name, repo), nil
}
