# EduClaw

EduClaw 是一个面向教育场景的轻量 AI Agent 平台，服务学生、家长、教师三类角色。它借鉴 OpenClaw 的核心思路，用 Go 重新实现了一套更轻、更易部署、更适合教育场景的数据与工作流内核。

项目当前提供：

- 学生端、家长端、教师端 Web 入口
- 基于 workspace 的角色长期记忆
- 基于 SQLite 的本地会话、知识状态与渲染内容存储
- 基于技能目录的可扩展能力加载
- 面向教育场景的 Agent + Tool + Web 服务一体化运行方式

## 项目设计核心

### 1. 借鉴 OpenClaw，但不照搬

EduClaw 参考了 OpenClaw 的几项关键设计：

- workspace 即长期记忆载体
- agent 通过工具而不是硬编码能力工作
- skill 以目录和 Markdown 指令组织
- 系统通过统一入口路由不同角色与任务

但它不是 OpenClaw 的直接移植，而是一次 Go 重写后的教育场景变体：

- 单一 Go 二进制，部署和运行成本更低
- 运行时依赖更少，资源占用更小
- 更适合本地部署、私有部署和学校内部环境
- 用更简单直接的 Web、SQLite、文件系统组合承载核心能力

如果说 OpenClaw 更像通用 Agent 基础设施，EduClaw 更强调教育场景下的可控、可审计、可落地。

### 2. 教育场景里，数据安全优先于炫技

EduClaw 把教育数据安全放在架构中心，而不是后补：

- 以本地 workspace 文件保存长期角色信息，便于审计和人工修订
- 以本地 SQLite 保存会话、知识状态、学习事件和渲染结果
- 支持私有化配置模型提供方，避免把教育数据强绑定到单一云平台
- 学生、家庭、教师三类数据天然分目录隔离
- 配置、技能、记忆和会话边界清晰，便于后续做权限控制和部署裁剪

教育不是普通聊天产品。学生画像、家庭背景、教师工作内容都属于高敏感信息，因此系统设计必须优先考虑可控存储、清晰边界和最小依赖。

### 3. 设计重点是“长期记忆 + 多角色协同 + 可扩展能力”

EduClaw 当前代码已经体现出几个非常明确的设计方向：

- 三类角色入口：`/student`、`/parent`、`/teacher`
- 统一 Agent Loop：消息进入后，按角色和意图路由，再进入工具调用循环
- 基于 workspace 的上下文装配：学生、家庭、教师各自拥有独立目录和模板
- 基于技能系统的扩展：内置技能、本地技能、工作区技能可以统一发现和加载
- 基于命令系统的操作入口：`/help`、`/list`、`/show`、`/clear`
- 基于健康检查与 setup 向导的可运维能力：`/health`、`/ready`、`/setup`

这意味着 EduClaw 不是“一个聊天页面”，而是在向“教育领域的 Agent Runtime”演进。

## 架构概览

当前仓库可以概括为 5 层：

1. CLI 入口
   `cmd/educlaw`

2. Web 服务层
   `pkg/web`

3. Agent 编排层
   `pkg/agents`

4. 能力扩展层
   `pkg/tools`、`pkg/skills`、`pkg/commands`

5. 数据与工作区层
   `pkg/storage`、`pkg/memory`、`pkg/workspace`

核心运行流程：

1. `educlaw serve` 启动服务
2. Web 端请求进入统一 API
3. Agent Loop 根据角色选择合适的上下文与工具
4. 模型输出触发工具调用或直接响应
5. 会话、知识状态、渲染结果写入本地存储
6. 前端通过三端页面消费结果

## 目录说明

```text
.
├── cmd/educlaw                  # CLI 入口，包含 serve / onboard / version
├── pkg/agents                   # Agent loop、角色路由、上下文构建
├── pkg/web                      # HTTP 服务、三端页面与 API
├── pkg/config                   # 配置加载与模型选择
├── pkg/skills                   # 技能发现、加载与安装
├── pkg/tools                    # Agent 可调用工具
├── pkg/commands                 # slash commands
├── pkg/storage                  # SQLite 持久化
├── pkg/memory                   # 会话记忆存储
├── pkg/workspace                # 角色工作区与模板初始化
├── skills                       # 内置技能
└── workspace_templates          # 学生、家庭、教师工作区模板
```

## 当前能力

### 学生端

- 对话式学习入口
- 会话流式输出
- 知识摘要与学习内容渲染
- 可结合技能生成练习、可视化内容和互动素材

### 家长端

- 家庭角色对话入口
- 报告查看与渲染内容展示
- 面向家庭语境的学习反馈承载

### 教师端

- 教师角色对话入口
- 班级报告与学情分析接口
- 备课辅助与内容生成能力

## 工作区与记忆模型

EduClaw 的长期记忆建立在 workspace 目录结构上。默认工作区是 `~/.educlaw`。

其中包括：

- `students/<id>`
- `families/<id>`
- `teachers/<id>`
- `agents`

每类角色都可以从模板初始化，例如：

- 学生：`PROFILE.md`、`SOUL.md`、`KNOWLEDGE.md`、`ERRORS.md`、`INTERESTS.md`
- 家庭：`CONTEXT.md`、`GOALS.md`、`REGION.md`
- 教师：`PROFILE.md`、`CLASSES.md`

这种设计有几个好处：

- 记忆是显式文件，不是黑箱
- 可以人工编辑、审阅和版本化
- 不同角色天然隔离
- 很适合教育场景里长期陪伴、长期积累的需求

## 模型配置

EduClaw 使用新的 `model_list` 机制管理模型，不再介绍旧格式。

最小配置示例：

```json
{
  "llm": {
    "primary": {
      "model": "minimax-default"
    }
  },
  "model_list": [
    {
      "model_name": "minimax-default",
      "provider": "minimax",
      "model": "MiniMax-M2.5",
      "api_key": "your-api-key",
      "api_base": "https://api.minimaxi.com/v1"
    }
  ],
  "workspace": "~/.educlaw",
  "server": {
    "host": "127.0.0.1",
    "port": 18080
  }
}
```

也可以直接使用仓库里的 [config.example.json](./config.example.json) 作为起点。

当前代码支持的 provider 包括：

- `minimax`
- `openai`
- `deepseek`
- `gemini`
- `anthropic`
- `openrouter`
- `groq`
- `ollama`
- `zhipu`
- `mistral`
- `moonshot`
- `nvidia`
- `litellm`

## 快速开始

### 1. 准备配置

复制并修改配置文件：

```bash
cp config.example.json config.json
```

Windows PowerShell：

```powershell
Copy-Item config.example.json config.json
```

### 2. 启动服务

```bash
go run ./cmd/educlaw serve -c ./config.json
```

### 3. 打开页面

- 学生端：`http://127.0.0.1:18080/student`
- 家长端：`http://127.0.0.1:18080/parent`
- 教师端：`http://127.0.0.1:18080/teacher`
- 初始化向导：`http://127.0.0.1:18080/setup`

如果系统尚未完成配置，根路径会自动跳转到 `/setup`。

## Commands

输入以 `/` 开头时，会优先进入命令执行而不是 Agent 对话。

当前内置命令：

- `/help`
- `/list`
- `/show model`
- `/show skill <name>`
- `/clear`

## Skills

技能系统是 EduClaw 的核心扩展机制之一。

当前支持三层技能目录：

- workspace skills
- global skills
- builtin skills

仓库内置了一批教育相关技能，例如：

- `quiz-generator`
- `lesson-planner`
- `report-generator`
- `visual-explainer`
- `story-problem`
- `game-generator`
- `comic-generator`

内置技能目录见 [skills](./skills)。

## 健康检查与运维

启用后可访问：

- `/health`
- `/ready`

当前 readiness 会检查：

- 数据库可用性
- 模型配置状态
- 技能目录状态

## 为什么开源

EduClaw 适合开源，不只是因为它能跑，而是因为它的设计天然需要透明：

- 教育数据如何存
- 角色记忆如何组织
- 技能如何扩展
- 模型如何接入
- 系统如何做到轻量可控

开源之后，开发者、学校和教育团队可以基于同一套核心能力做二次开发，而不是被绑定在封闭平台里。

## 路线方向

当前仓库已经具备基础可运行框架。后续演进方向可以继续围绕以下几条主线展开：

- 更完整的学生知识状态建模
- 更严格的权限控制与数据隔离
- 更稳定的教师端分析链路
- 更丰富的教育技能生态
- 更适合学校和家庭的私有化部署方案

## 开发原则

EduClaw 当前 README 所描述的是“当前推荐架构”：

- 使用 Go 作为统一实现语言
- 使用 `model_list` 作为唯一推荐模型配置方式
- 使用新的工作区、技能和三端结构继续演进

不再在 README 中保留旧路径、旧格式和兼容说明。
