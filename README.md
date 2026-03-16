# EduClaw

EduClaw 是一个面向教育场景的轻量 AI Agent 平台，服务学生、家长、教师三类角色。

它借鉴 OpenClaw 的核心思路，用 Go 重新实现了一套更轻、更易部署、更适合教育场景的数据与工作流内核。

EduClaw 的目标不是成为另一个通用 AI 框架，而是提供一个真正可以落地到教育环境中的 **Agent Runtime**。

---

## 核心特点

- 学生端、家长端、教师端 Web 入口
- 基于 workspace 的角色长期记忆
- 基于 SQLite 的本地会话、知识状态与渲染内容存储
- 基于技能目录的可扩展能力加载
- 面向教育场景的 Agent + Tool + Web 服务一体化运行方式
- 单一 Go 二进制部署
- 支持多模型 provider 接入

EduClaw 强调：

**可控、可审计、可部署、可扩展**

而不是单纯的聊天产品。

---

## Why EduClaw

通用 Agent 框架往往更适合通用 AI 应用，而教育场景有一些独特需求：

- 长期记忆
- 多角色协同
- 教育数据安全
- 私有化部署
- 教师与家庭参与

EduClaw 的设计重点是：

**长期记忆 + 多角色协同 + 可扩展能力**

如果说 OpenClaw 更像通用 Agent 基础设施，

EduClaw 更强调：

**教育场景下的可控、可审计、可落地。**

---

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

---

### 2. 教育场景里，数据安全优先于炫技

EduClaw 把教育数据安全放在架构中心，而不是后补：

- 以本地 workspace 文件保存长期角色信息，便于审计和人工修订
- 以本地 SQLite 保存会话、知识状态、学习事件和渲染结果
- 支持私有化配置模型提供方，避免把教育数据强绑定到单一云平台
- 学生、家庭、教师三类数据天然分目录隔离
- 配置、技能、记忆和会话边界清晰，便于后续做权限控制和部署裁剪

教育不是普通聊天产品。学生画像、家庭背景、教师工作内容都属于高敏感信息，因此系统设计必须优先考虑可控存储、清晰边界和最小依赖。

---

### 3. 设计重点是“长期记忆 + 多角色协同 + 可扩展能力”

EduClaw 当前代码已经体现出几个非常明确的设计方向：

- 三类角色入口：`/student`、`/parent`、`/teacher`
- 统一 Agent Loop：消息进入后，按角色和意图路由，再进入工具调用循环
- 基于 workspace 的上下文装配：学生、家庭、教师各自拥有独立目录和模板
- 基于技能系统的扩展：内置技能、本地技能、工作区技能可以统一发现和加载
- 基于命令系统的操作入口：`/help`、`/list`、`/show`、`/clear`
- 基于健康检查与 setup 向导的可运维能力：`/health`、`/ready`、`/setup`

这意味着 EduClaw 不是“一个聊天页面”，而是在向“教育领域的 Agent Runtime”演进。

---

## 架构概览

当前仓库按职责分为以下层次（与源码一致）：

| 层次 | 包/目录 | 职责 |
|------|---------|------|
| **CLI 入口** | `cmd/educlaw` | 根命令；子命令 `serve`、`onboard`、`version`；`internal/serve` 负责配置加载、DB/工作区/LLM/技能/AgentLoop/Web 一站式初始化 |
| **Web 服务层** | `pkg/web` | Gin 路由；静态页 `/student`、`/parent`、`/teacher`、`/setup`；API：`/api/chat`、`/api/chat/stream/:session_id`、学生/家长/教师摘要与报告、onboard、setup、actors；健康检查 `/health`、`/ready` |
| **Agent 编排层** | `pkg/agents` | `AgentLoop`：会话获取/历史清洗、`orchestrator.Route` 按角色与意图选 Agent、`ContextBuilder` 装配系统提示、ReAct 循环（LLM 流式 + 工具调用）、会话落库与总线推送；子包 `orchestrator`（路由）、`tutor`/`companion`/`planner`/`analyst`/`teacher`/`parent` 提供各角色/场景的 prompt |
| **能力扩展层** | `pkg/tools` | Agent 可调用工具：工作区读写、每日笔记、学习事件、知识查询、内容渲染、技能读取/发现/安装、Web 抓取与搜索、定时提醒（依赖 `pkg/cron`） |
| | `pkg/skills` | 技能发现与加载：workspace / global / builtin 三层目录扫描，`Loader`、`RegistryManager`、`Installer` |
| | `pkg/commands` | Slash 命令：`/help`、`/list`、`/show`、`/clear`；`Executor` + `Runtime` 对接模型信息、技能列表、清空历史 |
| **数据与工作区层** | `pkg/storage` | SQLite 初始化与表结构：`actors`、`sessions`、`learning_events`、`knowledge_states`、`rendered_contents` |
| | `pkg/memory` | 基于 SQLite 的会话存储（`SQLiteStore`）：GetOrCreateSession、SetHistory、消息序列化 |
| | `pkg/workspace` | 工作区管理：`students/<id>`、`families/<id>`、`teachers/<id>`、`agents`；读写/追加文件、模板初始化 |
| **支撑** | `pkg/config` | 配置加载、`model_list` 解析与主/备模型选择、工作区与服务器地址 |
| | `pkg/llm` | 多 Provider 抽象；HTTP 客户端；流式完成；可选 fallback、disabled provider |
| | `pkg/bus` | 消息总线：InboundMessage / OutboundMessage，供 Web 与 AgentLoop 解耦 |
| | `pkg/health` | 就绪/存活检查注册（database、llm、skills） |
| | `pkg/cron`、`pkg/heartbeat` | 定时任务与心跳，可向 AgentLoop 投递消息 |

核心运行流程（对应 `serve` 启动与请求路径）：

1. `educlaw serve` → 加载配置、初始化 DB、工作区、LLM、技能 Loader、`AgentLoop`、可选 cron/heartbeat，创建 `web.Server` 并注册路由。  
2. 用户访问 `/student`/`/parent`/`/teacher` 或调用 `/api/chat` → 若为聊天则经 `HandleChat` 发到 `bus`，由消费端调用 `AgentLoop.Process`。  
3. `AgentLoop.Process`：取/建会话、按角色与内容 `orchestrator.Route` 选 agent 类型、用 `ContextBuilder` 拼系统提示、按会话与角色注册工具（工作区、知识、技能、渲染、cron 等）、执行 ReAct 循环（流式输出到 bus、工具调用结果写回）。  
4. 会话历史与学习事件/知识状态/渲染内容由 `memory` 与 `storage` 写入 SQLite；前端通过 `/api/chat/stream/:session_id` 等消费 SSE 与结果。

---

## 目录说明

以下为源码仓库实际目录与用途说明：

```
.
├── cmd/
│   └── educlaw/                    # CLI 入口
│       ├── main.go                 # 根命令，注册 serve / onboard / version
│       └── internal/
│           ├── helpers.go          # 配置加载等公共逻辑
│           ├── serve/              # serve 子命令：SetupServer，组装 DB/workspace/LLM/health/skills/agentLoop/cron/heartbeat/web
│           ├── onboard/            # onboard 子命令
│           └── version/            # version 子命令
│
├── pkg/
│   ├── agents/                     # Agent 编排：ReAct 循环、上下文构建、按角色路由
│   │   ├── loop.go                 # AgentLoop：会话、Route、ContextBuilder、工具注册、ReAct、落库与总线
│   │   ├── context.go              # ContextBuilder：装配 workspace + agent 目录下的 prompt
│   │   ├── types.go
│   │   ├── prompts.go
│   │   ├── orchestrator/           # 按 actor 类型与内容关键词路由到 tutor/companion/planner/analyst/teacher/parent
│   │   ├── tutor/                  # 学生默认 Agent（测验、学习）
│   │   ├── companion/              # 学生游戏/趣味场景
│   │   ├── planner/                # 计划、安排（学生/家长/教师）
│   │   ├── analyst/                # 学情、报告（家长/教师）
│   │   ├── teacher/                # 教师通用
│   │   └── parent/                 # 家长通用
│   │
│   ├── web/                        # HTTP 服务与三端页面
│   │   ├── server.go               # Server 构造、路由注册、commandRuntime
│   │   ├── routes.go               # 路由说明注释
│   │   ├── student_api.go
│   │   ├── parent_api.go
│   │   ├── teacher_api.go
│   │   ├── health_api.go
│   │   ├── setup_api.go
│   │   └── static/                 # 内嵌：student.html, parent.html, teacher.html, setup.html
│   │
│   ├── config/                     # 配置加载、model_list、工作区路径、server、health、cron、heartbeat
│   ├── llm/                        # Provider 接口、多 provider/fallback、HTTP 客户端、流式、disabled
│   ├── bus/                        # 消息总线：InboundMessage / OutboundMessage
│   ├── health/                     # 就绪/存活检查 Manager
│   │
│   ├── tools/                      # Agent 可调用工具
│   │   ├── registry.go
│   │   ├── tools.go                # 工作区读写、每日笔记、事件、知识、渲染、技能读/发现/安装、web 抓取/搜索
│   │   ├── cron_tool.go            # 定时提醒（依赖 pkg/cron）
│   │   └── web.go
│   ├── skills/                     # 技能发现与安装
│   │   ├── loader.go               # workspace / global / builtin 三层扫描
│   │   ├── registry.go
│   │   └── installer.go
│   ├── commands/                   # Slash 命令
│   │   ├── executor.go
│   │   ├── registry.go
│   │   ├── runtime.go
│   │   ├── definition.go
│   │   ├── builtin.go              # /help, /list, /show, /clear
│   │   └── cmd_help.go, cmd_list.go, cmd_show.go, cmd_clear.go
│   │
│   ├── storage/                    # SQLite：表 actors, sessions, learning_events, knowledge_states, rendered_contents
│   │   ├── db.go
│   │   ├── sessions.go
│   │   ├── events.go
│   │   └── knowledge.go
│   ├── memory/                     # 会话存储：SQLiteStore，GetOrCreateSession、SetHistory
│   │   ├── store.go
│   │   └── sqlite.go
│   ├── workspace/                  # 工作区管理：StudentDir/FamilyDir/TeacherDir/AgentDir，读写文件
│   │   └── manager.go
│   │
│   ├── cron/                      # 定时任务服务，可向 AgentLoop 投递消息
│   └── heartbeat/                 # 心跳服务，可触发 Agent 处理
│
├── skills/                         # 内置技能（SKILL.md + 资源）
│   ├── quiz-generator/
│   ├── lesson-planner/
│   ├── report-generator/
│   ├── visual-explainer/
│   ├── story-problem/
│   ├── game-generator/
│   ├── comic-generator/
│   ├── video-intro/
│   └── notebooklm-embed/
│
├── workspace_templates/            # 工作区模板（初始化学生/家庭/教师目录用）
│   ├── student/                   # PROFILE, SOUL, KNOWLEDGE, ERRORS, INTERESTS, AGENTS, memory/MEMORY
│   ├── family/                    # CONTEXT, GOALS, REGION
│   └── teacher/                   # PROFILE, CLASSES
│
├── config.example.json
├── go.mod
└── go.sum
```

---

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

---

## 工作区与记忆模型

EduClaw 的长期记忆建立在 workspace 目录结构上。

默认工作区：

```
~/.educlaw
```

其中包括：

- `students/<id>`
- `families/<id>`
- `teachers/<id>`
- `agents`

学生 workspace 示例：

```
PROFILE.md
SOUL.md
KNOWLEDGE.md
ERRORS.md
INTERESTS.md
```

家庭 workspace 示例：

```
CONTEXT.md
GOALS.md
REGION.md
```

教师 workspace 示例：

```
PROFILE.md
CLASSES.md
```

优势：

- 记忆是显式文件
- 可以人工编辑
- 可以版本管理
- 角色天然隔离

---

## 模型配置

EduClaw 使用 `model_list` 管理模型。

示例：

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

支持 provider：

* minimax
* openai
* deepseek
* gemini
* anthropic
* openrouter
* groq
* ollama
* zhipu
* mistral
* moonshot
* nvidia
* litellm

---

## 快速开始

### 1 准备配置

```bash
cp config.example.json config.json
```

Windows：

```powershell
Copy-Item config.example.json config.json
```

### 2 启动服务

```bash
go run ./cmd/educlaw serve -c ./config.json
```

### 3 打开页面

学生端

```
http://127.0.0.1:18080/student
```

家长端

```
http://127.0.0.1:18080/parent
```

教师端

```
http://127.0.0.1:18080/teacher
```

初始化

```
http://127.0.0.1:18080/setup
```

---

## Commands

内置命令：

```
/help
/list
/show model
/show skill <name>
/clear
```

---

## Skills

技能系统是 EduClaw 的核心扩展机制。

三层目录：

* workspace skills
* global skills
* builtin skills

内置技能：

* quiz-generator
* lesson-planner
* report-generator
* visual-explainer
* story-problem
* game-generator
* comic-generator

---

## 健康检查

接口：

```
/health
/ready
```

检查：

* 数据库状态
* 模型配置
* 技能目录

---

## 为什么开源

EduClaw 适合开源，因为教育系统需要透明：

* 数据如何存储
* 角色记忆如何组织
* 技能如何扩展
* 模型如何接入

开源后，开发者和学校可以基于同一核心能力扩展。

---

## Roadmap

未来方向：

* 学生知识状态建模
* 权限与数据隔离
* 教师分析链路
* 教育技能生态

---

## Contributing

欢迎提交 Issue 或 Pull Request。

---

## License

Apache License 2.0
