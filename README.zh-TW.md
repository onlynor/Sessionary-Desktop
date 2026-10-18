# Sessionary Desktop

[![Release](https://img.shields.io/github/v/release/onlynor/Sessionary-Desktop)](https://github.com/onlynor/Sessionary-Desktop/releases)
[![CI](https://github.com/onlynor/Sessionary-Desktop/actions/workflows/ci.yml/badge.svg)](https://github.com/onlynor/Sessionary-Desktop/actions/workflows/ci.yml)
[![License: Apache-2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

[English](README.md) | [简体中文](README.zh-CN.md) | **繁體中文** | [日本語](README.ja.md)

**Sessionary Desktop** 是一款開源的桌面控制中心,用來管理 AI 程式設計代理(Agent)——Claude Code、Codex、OpenCode、
Pi、DeepSeek Harness、Hermes、WorkBuddy 與 WorkBuddy AI。它將 [Sessionary](https://github.com/onlynor/Sessionary) 打包成
一個在 Linux、macOS 與 Windows 上執行的原生應用程式,讓你在同一個視窗裡管理所有代理的專案、工作階段(session)、模型
路由與 token 用量,不必再開著瀏覽器分頁。

和一般的 AI 聊天應用程式不同,Sessionary Desktop 本身不執行聊天引擎,也不儲存自己的對話記錄:它讀取的是每個代理 CLI
原本就存在磁碟上的真實工作階段歷史與檔案——無論是在這台電腦上,還是透過 SSH 連線的遠端機器——再於其上提供一套一致的
介面:代理卡片、對話、終端機、模型路由、用量統計。

## 功能特色

- **AI 程式設計代理管理** —— 自動偵測已安裝的代理 CLI(Claude Code、Codex、OpenCode、Pi、Hermes、WorkBuddy、
  WorkBuddy AI,以及若已安裝的 [DeepSeek Harness](docs/ARCHITECTURE.md#deepseek-harness-integration)),並顯示每個代理
  的工作階段,不必再為每個代理分別維護終端機與設定檔。
- **專案與工作階段** —— 依專案資料夾瀏覽某個代理過去的工作階段、繼續其中一個,或在你選擇的資料夾裡開始新的工作階段。
- **本機與遠端機器** —— 可操作這台電腦上的代理,也可以透過 **SSH**(以金鑰驗證,不會儲存任何密碼或權杖)新增另一台
  機器來讀取它的代理歷史;遠端機器上不會安裝任何東西。
- **模型提供者與路由** —— 設定一次模型提供者與 API 金鑰後,就能在 Routing 頁面指定每個代理要使用的模型,全部經由
  Sessionary 自己的閘道(gateway)。
- **Token 用量與分析** —— 在 Usage 頁面查看依代理、依機器劃分的用量與 token 消耗。
- **終端機與監控** —— 在某個專案裡開啟一個真正的終端機,並在一個分頁裡查看某台機器上代理的即時活動。

以上僅列出目前已經實作的功能。已知的缺口請見[尚未完成](docs/ARCHITECTURE.md#not-yet-done)(英文)。

## 安裝與快速開始

從 **[GitHub Releases](https://github.com/onlynor/Sessionary-Desktop/releases)** 下載最新版本。每次發布包含:

| 平台 | 下載內容 | 備註 |
|---|---|---|
| Linux(amd64、arm64) | `.deb`,或附 `install.sh` 的 `.tar.gz` | |
| macOS(Apple 晶片) | `.dmg` | 目前還沒有 Intel Mac 的建置 |
| Windows(amd64) | `Sessionary-Setup-<version>-amd64.exe` | 目前還沒有 arm64 的建置 |

安裝檔**目前尚未簽署或公證**:macOS 的 Gatekeeper 與 Windows 的 SmartScreen 會在第一次執行時發出警告(見下方
[疑難排解](#疑難排解))。可以用發布頁裡的 `SHA256SUMS` 搭配 `sha256sum -c SHA256SUMS` 驗證下載檔案。

**安裝方式:**

```sh
# Linux,.deb
sudo apt install ./sessionary_<version>_amd64.deb   # 或 _arm64.deb

# Linux,tarball
tar xf sessionary-<version>-linux-amd64.tar.gz && ./install.sh

# macOS
# 開啟 .dmg,將 Sessionary 拖曳到 Applications

# Windows
# 執行 Sessionary-Setup-<version>-amd64.exe
```

**先決條件:** Sessionary Desktop 本身不附帶任何代理 CLI。請先安裝它支援的至少一個代理(例如
[Claude Code](https://claude.com/claude-code)、[Codex](https://github.com/openai/codex)、OpenCode 或 Pi),並確保它能在
你平常使用的登入 shell 中直接執行。[DeepSeek Harness](docs/ARCHITECTURE.md#deepseek-harness-integration) 不需要任何設定
——只要機器上已經裝好受支援的版本,Sessionary 會自動找到它。

**第一個工作階段:** 啟動應用程式 → 它會自動偵測機器上已安裝的代理 → 在側邊欄開啟 **Agents**,選擇一個已偵測到的代理
→ 選擇一個專案資料夾 → 開始(或繼續)一個工作階段。不需要帳號或登入步驟。

## 使用方式

側邊欄分為三組:

- **Home** 與 **Machines** —— 總覽,以及機器清單(這台電腦,加上你透過 SSH 新增的任何機器)。
- **Workspace**(作用於目前選取的機器)—— **Agents**(每個代理的卡片與工作階段)、**Projects**、**Sessions**、
  **Terminal**,以及 **Monitor**(該機器上代理的即時活動)。
- **Model Control** —— **Models** 與 **Gateway**(設定提供者與 API 金鑰)、**Routing**(為每個代理選擇要使用的模型),
  以及 **Usage**(隨時間變化的 token 消耗)。

**一般流程:** 在側邊欄選擇一台機器(預設為這台電腦)→ 開啟 **Agents** 並選擇一個已偵測到的代理 → 選擇一個專案資料夾
→ 在 **Sessions** 裡開始新工作階段或繼續既有的。對話發生在 Sessionary 自己的聊天介面裡——串流回覆、工具呼叫、diff、
模型選擇器——而不是單純的終端機;不過要為同一個專案開啟一個真正的終端機,隨時只要點一下。

**代理偵測與 PATH。** 只有當某個代理的 CLI 能在 `PATH` 中找到時,才會被標示為「已偵測到」。如果你是在啟動應用程式之後
才安裝某個代理,或把它裝到 `~/.local/bin`、npm/pnpm 的全域目錄,或某個版本管理工具(nvm、volta、fnm、mise、asdf)裡,
請重新啟動應用程式,讓它重新讀取登入 shell 的 `PATH`。如果某個確實已安裝的代理仍然沒有出現,請參見
[疑難排解](#疑難排解)。

**模型提供者與路由。** 在 **Gateway**/**Models** 頁面一次性新增提供者與 API 金鑰,再用 **Routing** 為每個代理選擇要
使用的模型;所有代理都透過 Sessionary 自己的閘道存取你設定好的提供者,金鑰只需保存在一處。**Usage** 會依代理與機器
列出 token 消耗的明細。

**透過 SSH 連接遠端機器。** 在 **Machines** 頁面,依主機(或 `~/.ssh/config` 裡的別名)新增一個節點,並可選擇填寫埠號
與金鑰檔案。Sessionary 使用你自己的 SSH 金鑰連線——不會儲存任何密碼或權杖——並且只讀取該機器上既有的代理工作階段歷史
(Claude Code、OpenCode、Pi、Hermes);不會在上面安裝任何東西。DeepSeek Harness 僅支援本機,無法用於遠端機器。新增
節點前,請先確認用同一組 host/user/port/identity 執行 `ssh user@host true` 已經能在不要求輸入密碼的情況下成功。

超出日常使用範圍的實作細節,請見 [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) 與
[docs/DESKTOP_RUNTIME.md](docs/DESKTOP_RUNTIME.md)(英文)。

## 疑難排解

- **已安裝的代理沒有出現。** 應用程式讀取的是登入 shell 的 `PATH`,而不是它被啟動時所在的環境。請先在一個新開的終端機
  裡確認該代理可以執行(`which <agent>`),然後完全結束並重新啟動 Sessionary Desktop,讓它重新讀取環境變數。
- **macOS 顯示應用程式已損毀或來自不明開發者。** 目前的建置尚未經過公證。請以滑鼠右鍵點選應用程式 → **打開**,並在
  彈出的對話框中確認。
- **Windows SmartScreen 擋下安裝程式。** 點選**其他資訊** → **仍要執行**。在建置完成簽署之前,這是預期行為。
- **新增遠端機器失敗或卡住。** 請先手動測試連線:用你填寫的同一組 host/user/port/identity 執行 `ssh user@host true`,
  必須能在不要求輸入密碼的情況下成功。修好 SSH 連線後再重新新增節點。
- **關閉視窗後應用程式就消失了(Linux)。** 在 Linux 上,只有在系統執行著 StatusNotifierItem host(大多數具備
  AppIndicator 擴充套件/模組的狀態列)時,關閉視窗才會把它隱藏到系統匣;沒有這個 host 時,關閉視窗就等於直接結束
  應用程式——這是預期行為,不是錯誤。從你的應用程式啟動器重新開啟即可。
- **應用程式無法啟動 / 顯示「Backend unavailable」。** 在該畫面點選 **Show Details** 查看內建伺服器的輸出,或查看
  `~/.config/Sessionary/logs/backend.log`(Linux/macOS 路徑;Windows 使用對應的每個使用者設定目錄)。**Try Again**
  會重新啟動它;如果一直失敗,請附上這份記錄提交 issue。

## 專案結構與技術概覽

Sessionary Desktop 是用 [MyGo](https://github.com/egoist/mygo) 建置的單一 Go 執行檔,內嵌了 Sessionary 自己的 Web
客戶端(從同層級的 [`Sessionary`](https://github.com/onlynor/Sessionary) 原始碼建置而來),並透過本機 WebKit/WebView2
視窗裡的一個回送(loopback)連接埠來提供服務。一個受監督的背景程序執行內建的 Sessionary 伺服器(連 Node.js 都是內建
的,機器上不需要額外安裝任何東西);視窗與它溝通的方式,和瀏覽器連到一個自架的 Sessionary 完全相同。

```
cmd/sessionary-desktop/   應用程式本體:生命週期、視窗、系統匣、選單
internal/{backend,uiserver,assets}/
frontend/                 原地建置的 Sessionary Web 客戶端,加上一層桌面 UI
scripts/, .github/        建置/發布腳本與 CI
```

完整細節(英文):[架構](docs/ARCHITECTURE.md) · [桌面執行時期(後端生命週期、視窗、系統匣)](docs/DESKTOP_RUNTIME.md) ·
[發布與版本管理](docs/RELEASES.md)。

## 開發與貢獻

需求:Go 1.27+、[Bun](https://bun.sh)、Node.js、WebKitGTK 4.1(Linux),以及一個同層級的
[Sessionary](https://github.com/onlynor/Sessionary) 原始碼,並已在其中執行過 `pnpm install`。

```sh
bun install --cwd frontend
bun run dev         # 支援熱重載的開發建置
bun run build       # 正式建置 → build/<os>-<arch>/
bun run typecheck
go test ./... && bun run --cwd frontend test
```

歡迎提交 issue 與 pull request。完整的建置、測試與 CI 設定請見 [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md)(英文),
版本發布流程請見 [docs/RELEASES.md](docs/RELEASES.md)(英文)。

## 授權與連結

Sessionary Desktop 以 [Apache License 2.0](LICENSE) 授權開源。本程式還內建了 Node.js 執行環境,並靜態連結了幾個各自使用
不同授權的 Go 函式庫;詳見 [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md)(英文,每個安裝包都會隨附在可執行檔旁)。

- [GitHub Releases](https://github.com/onlynor/Sessionary-Desktop/releases)
- [Sessionary](https://github.com/onlynor/Sessionary) —— 本應用程式打包的 Web 客戶端與伺服器
- [Issues](https://github.com/onlynor/Sessionary-Desktop/issues)
