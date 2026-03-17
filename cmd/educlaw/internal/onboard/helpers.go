package onboard

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"
	cmdinternal "github.com/pingjie/educlaw/cmd/educlaw/internal"
	"github.com/pingjie/educlaw/pkg/storage"
	"github.com/pingjie/educlaw/pkg/workspace"
	"github.com/spf13/cobra"
)

func runOnboard(_ *cobra.Command, _ []string) error {
	cfg, err := cmdinternal.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	db, err := storage.InitDB(cfg.DBPath())
	if err != nil {
		return fmt.Errorf("initializing database: %w", err)
	}
	defer db.Close()

	wm := workspace.NewManager(cfg.WorkspacePath())
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("\n=== EduClaw 用户注册 ===")
	fmt.Println("请选择用户类型:")
	fmt.Println("  1. 学生 (student)")
	fmt.Println("  2. 家长 (parent/family)")
	fmt.Println("  3. 教师 (teacher)")
	fmt.Print("\n请输入选择 (1/2/3): ")

	scanner.Scan()
	choice := strings.TrimSpace(scanner.Text())

	var actorType string
	switch choice {
	case "1":
		actorType = "student"
	case "2":
		actorType = "family"
	case "3":
		actorType = "teacher"
	default:
		return fmt.Errorf("无效选择: %s", choice)
	}

	fmt.Print("请输入姓名: ")
	scanner.Scan()
	name := strings.TrimSpace(scanner.Text())
	if name == "" {
		return fmt.Errorf("姓名不能为空")
	}

	var grade, subject, familyID string

	switch actorType {
	case "student":
		fmt.Print("请输入年级 (例如: 五年级, 可留空): ")
		scanner.Scan()
		grade = strings.TrimSpace(scanner.Text())

		fmt.Print("请输入家庭ID (可留空): ")
		scanner.Scan()
		familyID = strings.TrimSpace(scanner.Text())

	case "teacher":
		fmt.Print("请输入所教科目 (例如: 数学, 可留空): ")
		scanner.Scan()
		subject = strings.TrimSpace(scanner.Text())
	}

	// Generate ID
	id := uuid.New().String()

	// Save to DB
	if err := storage.SaveActor(db, id, actorType, name, grade, subject, familyID, ""); err != nil {
		return fmt.Errorf("saving actor: %w", err)
	}

	// Initialize workspace directory
	var actorDir string
	switch actorType {
	case "student":
		actorDir = wm.StudentDir(id)
	case "family":
		actorDir = wm.FamilyDir(id)
	case "teacher":
		actorDir = wm.TeacherDir(id)
	}

	// Copy embedded templates (works in single-binary deployments)
	if err := wm.InitFromEmbeddedTemplate(actorDir, actorType); err != nil {
		fmt.Printf("Warning: could not copy templates: %v\n", err)
	}

	// Write PROFILE.md
	profileContent := buildProfile(name, actorType, grade, subject)
	if err := wm.WriteFile(actorDir, "PROFILE.md", profileContent); err != nil {
		fmt.Printf("Warning: could not write PROFILE.md: %v\n", err)
	}

	fmt.Printf("\n✅ 注册成功!\n")
	fmt.Printf("   类型: %s\n", actorType)
	fmt.Printf("   姓名: %s\n", name)
	fmt.Printf("   ID: %s\n", id)
	fmt.Printf("   工作目录: %s\n", actorDir)
	fmt.Printf("\n启动服务器后可在以下地址使用:\n")

	switch actorType {
	case "student":
		fmt.Printf("   http://localhost:18080/student\n")
	case "family":
		fmt.Printf("   http://localhost:18080/parent\n")
	case "teacher":
		fmt.Printf("   http://localhost:18080/teacher\n")
	}

	return nil
}

func buildProfile(name, actorType, grade, subject string) string {
	switch actorType {
	case "student":
		return fmt.Sprintf(`# 学生档案

## 基本信息
- 姓名: %s
- 年级: %s
- 学习风格: (待填写)
- 兴趣爱好: (待填写)

## 学习目标
- (待填写)
`, name, grade)

	case "family":
		return fmt.Sprintf(`# 家庭档案

## 基本信息
- 家长姓名: %s
- 家庭情况: (待填写)

## 孩子信息
- (待关联)
`, name)

	case "teacher":
		return fmt.Sprintf(`# 教师档案

## 基本信息
- 姓名: %s
- 科目: %s
- 学校: (待填写)
- 年级: (待填写)

## 教学理念
- (待填写)
`, name, subject)
	}
	return "# Profile\n\nName: " + name + "\n"
}
