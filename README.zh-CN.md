# Sessionary Desktop

[![Release](https://img.shields.io/github/v/release/onlynor/Sessionary-Desktop)](https://github.com/onlynor/Sessionary-Desktop/releases)
[![CI](https://github.com/onlynor/Sessionary-Desktop/actions/workflows/ci.yml/badge.svg)](https://github.com/onlynor/Sessionary-Desktop/actions/workflows/ci.yml)
[![License: Apache-2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

[English](README.md) | **简体中文** | [繁體中文](README.zh-TW.md) | [日本語](README.ja.md)

**Sessionary Desktop** 是一款开源的桌面控制中心,用于管理 AI 编程 Agent——Claude Code、Codex、OpenCode、Pi、DeepSeek
Harness、Hermes、WorkBuddy 和 WorkBuddy AI。它把 [Sessionary](https://github.com/onlynor/Sessionary) 打包成一个
Linux、macOS 和 Windows 上的原生应用,让你在一个窗口里管理所有 Agent 的项目、会话、模型路由和 token 用量,而不必再开着浏览器标签页。

与一般的 AI 聊天应用不同,Sessionary Desktop 本身不运行聊天引擎,也不保存自己的对话记录:它读取的是每个 Agent CLI 本来就
保存在磁盘上的真实会话历史和文件——无论是在本机,还是通过 SSH 连接的远程机器——然后在它们之上提供一套一致的界面:Agent
卡片、对话、终端、模型路由、用量统计。

## 功能特性

- **AI 编程 Agent 管理** —— 自动检测已安装的 Agent CLI(Claude Code、Codex、OpenCode、Pi、Hermes、WorkBuddy、WorkBuddy AI,
  以及若已安装的 [DeepSeek Harness](docs/ARCHITECTURE.md#deepseek-harness-integration)),并展示每个 Agent 的会话,省去为
  每个 Agent 分别维护终端和配置文件的麻烦。
- **项目与会话** —— 按项目目录浏览某个 Agent 过去的会话,恢复其中一个,或在你选择的目录里开始新会话。
- **本地与远程机器** —— 既可以操作本机上的 Agent,也可以通过 **SSH**(基于密钥,不保存任何密码或令牌)添加另一台机器来
  读取它的 Agent 历史;远程机器上不会安装任何东西。
- **模型提供方与路由** —— 一次性配置模型提供方和 API 密钥,再到 Routing 页面为每个 Agent 指定使用的模型,统一经由
  Sessionary 自己的网关。
- **Token 用量与分析** —— 在 Usage 页面查看按 Agent、按机器划分的用量与 token 消耗。
- **终端与监控** —— 在某个项目里打开一个真正的终端,并在一个标签页里查看某台机器上 Agent 的实时活动。

以上只列出目前已经实现的功能。已知的缺口见[尚未完成](docs/ARCHITECTURE.md#not-yet-done)(英文)。

## 安装与快速开始

从 **[GitHub Releases](https://github.com/onlynor/Sessionary-Desktop/releases)** 下载最新版本。每次发布包含:

| 平台 | 下载内容 | 说明 |
|---|---|---|
| Linux(amd64、arm64) | `.deb`,或带 `install.sh` 的 `.tar.gz` | |
| macOS(Apple 芯片) | `.dmg` | 目前还没有 Intel Mac 的构建 |
| Windows(amd64) | `Sessionary-Setup-<version>-amd64.exe` | 目前还没有 arm64 的构建 |

安装包**目前还没有代码签名或公证**:macOS 的 Gatekeeper 和 Windows 的 SmartScreen 会在首次运行时发出警告(见下方
[故障排查](#故障排查))。可以用发布包里的 `SHA256SUMS` 通过 `sha256sum -c SHA256SUMS` 校验下载文件。

**安装:**

```sh
# Linux,.deb
sudo apt install ./sessionary_<version>_amd64.deb   # 或 _arm64.deb

# Linux,tarball
tar xf sessionary-<version>-linux-amd64.tar.gz && ./install.sh

# macOS
# 打开 .dmg,将 Sessionary 拖到 Applications

# Windows
# 运行 Sessionary-Setup-<version>-amd64.exe
```

**前提条件:** Sessionary Desktop 本身不内置任何 Agent CLI。请先安装它支持的至少一个 Agent(例如
[Claude Code](https://claude.com/claude-code)、[Codex](https://github.com/openai/codex)、OpenCode 或 Pi),并确保它能在你
平时的登录 shell 中直接调用。[DeepSeek Harness](docs/ARCHITECTURE.md#deepseek-harness-integration) 不需要任何配置——如果机器
上已经装好了受支持的版本,Sessionary 会自动找到它。

**第一个会话:** 启动应用 → 它会自动检测机器上已安装的 Agent → 在侧边栏打开 **Agents**,选择一个已检测到的 Agent → 选择一个
项目目录 → 开始(或恢复)一个会话。不需要账号或登录步骤。

## 使用方法

侧边栏分为三组:

- **Home** 和 **Machines** —— 总览,以及机器列表(本机,加上你通过 SSH 添加的任何机器)。
- **Workspace**(作用于当前选中的机器)—— **Agents**(每个 Agent 的卡片和会话)、**Projects**、**Sessions**、
  **Terminal**,以及 **Monitor**(该机器上 Agent 的实时活动)。
- **Model Control** —— **Models** 和 **Gateway**(配置提供方和 API 密钥)、**Routing**(为每个 Agent 选择使用的模型),
  以及 **Usage**(随时间变化的 token 消耗)。

**常规流程:** 在侧边栏选一台机器(默认是本机)→ 打开 **Agents** 并选择一个已检测到的 → 选择一个项目目录 → 在
**Sessions** 里开始新会话或恢复已有会话。对话发生在 Sessionary 自己的聊天界面里——流式回复、工具调用、diff、模型选择器
——而不是一个纯终端;不过为同一个项目打开一个真正的终端,随时只需一次点击。

**Agent 检测与 PATH。** 当某个 Agent 的 CLI 能在 `PATH` 中找到时,它才会被标记为"已检测到"。如果你在启动应用之后才安装了
某个 Agent,或者把它装到了 `~/.local/bin`、npm/pnpm 的全局目录,或某个版本管理器(nvm、volta、fnm、mise、asdf)里,请重启
应用,让它重新读取登录 shell 的 `PATH`。如果某个确实已安装的 Agent 仍然没有出现,参见[故障排查](#故障排查)。

**模型提供方与路由。** 在 **Gateway**/**Models** 页面一次性添加提供方和 API 密钥,再用 **Routing** 为每个 Agent 选择
它使用的模型;Agent 都通过 Sessionary 自己的网关访问你配置好的提供方,密钥只需保存在一处。**Usage** 按 Agent 和机器给出
token 消耗的明细。

**通过 SSH 连接远程机器。** 在 **Machines** 页面,按主机(或 `~/.ssh/config` 里的别名)添加一个节点,并可选填写端口和
身份文件。Sessionary 使用你自己的 SSH 密钥连接——不保存任何密码或令牌——并且只读取该机器上已有的 Agent 会话历史
(Claude Code、OpenCode、Pi、Hermes);不会在上面安装任何东西。DeepSeek Harness 仅支持本机,无法在远程机器上使用。添加
节点前,先确认普通的 `ssh user@host true` 已经能在不提示输入密码的情况下成功执行。

超出日常使用范围的实现细节,见 [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) 和
[docs/DESKTOP_RUNTIME.md](docs/DESKTOP_RUNTIME.md)(英文)。

## 故障排查

- **已安装的 Agent 没有出现。** 应用读取的是登录 shell 的 `PATH`,而不是它被启动时所处的环境。先在一个新开的终端里确认该
  Agent 可以运行(`which <agent>`),然后完全退出并重新启动 Sessionary Desktop,让它重新读取环境变量。
- **macOS 提示应用已损坏或来自身份不明的开发者。** 构建目前还没有经过公证。右键点击应用 → **打开**,在弹出的对话框中确认。
- **Windows SmartScreen 拦截了安装程序。** 点击**更多信息** → **仍要运行**。在构建被签名之前,这是预期行为。
- **添加远程机器失败或卡住。** 先手动测试连接:用你填写的同一组 host/user/port/identity 执行 `ssh user@host true`,
  必须能在不提示密码的情况下成功。修复 SSH 访问后再重新添加节点。
- **关闭窗口后应用消失了(Linux)。** 在 Linux 上,只有在运行着 StatusNotifierItem host(大多数带 AppIndicator 扩展/模块
  的状态栏)时,关闭窗口才会把它隐藏到托盘;没有这个 host 时,关闭窗口就是直接退出应用——这是预期行为,不是缺陷。从你的
  应用启动器里重新打开它即可。
- **应用无法启动 / 显示"Backend unavailable"。** 在该界面点击 **Show Details** 查看内置服务器的输出,或者查看
  `~/.config/Sessionary/logs/backend.log`(Linux/macOS 路径;Windows 使用对应的每用户配置目录)。**Try Again** 会重新
  启动它;如果一直失败,请附上这份日志提交 issue。

## 项目结构与技术概览

Sessionary Desktop 是用 [MyGo](https://github.com/egoist/mygo) 构建的单一 Go 可执行文件,它内嵌了 Sessionary 自己的
Web 客户端(从同级的 [`Sessionary`](https://github.com/onlynor/Sessionary) 代码检出构建而来),并通过本机 WebKit/
WebView2 窗口里的一个回环端口来提供服务。一个受监督的后台进程运行内置的 Sessionary 服务器(连 Node.js 都是内置的,机器
上无需额外安装任何东西);窗口与它通信的方式,和浏览器访问一个自托管的 Sessionary 完全一样。

```
cmd/sessionary-desktop/   应用本体:生命周期、窗口、托盘、菜单
internal/{backend,uiserver,assets}/
frontend/                 原地构建的 Sessionary Web 客户端,外加一层桌面 UI
scripts/, .github/        构建/发布脚本与 CI
```

完整细节(英文):[架构](docs/ARCHITECTURE.md) · [桌面运行时(后端生命周期、窗口、托盘)](docs/DESKTOP_RUNTIME.md) ·
[发布与版本管理](docs/RELEASES.md)。

## 开发与贡献

依赖要求:Go 1.27+、[Bun](https://bun.sh)、Node.js、WebKitGTK 4.1(Linux),以及一个同级的
[Sessionary](https://github.com/onlynor/Sessionary) 代码检出,并在其中执行过 `pnpm install`。

```sh
bun install --cwd frontend
bun run dev         # 支持热重载的开发构建
bun run build       # 生产构建 → build/<os>-<arch>/
bun run typecheck
go test ./... && bun run --cwd frontend test
```

欢迎提交 issue 和 pull request。完整的构建、测试与 CI 配置见 [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md)(英文),版本
发布流程见 [docs/RELEASES.md](docs/RELEASES.md)(英文)。

## 许可证与链接

Sessionary Desktop 基于 [Apache License 2.0](LICENSE) 开源。本程序还打包了 Node.js 运行时,并静态链接了几个使用各自许可证的
Go 库;详见 [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md)(英文,每个安装包中都会附带在可执行文件旁)。

- [GitHub Releases](https://github.com/onlynor/Sessionary-Desktop/releases)
- [Sessionary](https://github.com/onlynor/Sessionary) —— 本应用打包的 Web 客户端与服务器
- [Issues](https://github.com/onlynor/Sessionary-Desktop/issues)
