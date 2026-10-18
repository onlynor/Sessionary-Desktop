# Sessionary Desktop

[![Release](https://img.shields.io/github/v/release/onlynor/Sessionary-Desktop)](https://github.com/onlynor/Sessionary-Desktop/releases)
[![CI](https://github.com/onlynor/Sessionary-Desktop/actions/workflows/ci.yml/badge.svg)](https://github.com/onlynor/Sessionary-Desktop/actions/workflows/ci.yml)
[![License: Apache-2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

[English](README.md) | [简体中文](README.zh-CN.md) | [繁體中文](README.zh-TW.md) | **日本語**

**Sessionary Desktop** は、Claude Code、Codex、OpenCode、Pi、DeepSeek Harness、Hermes、WorkBuddy、WorkBuddy AI
といった AI コーディングエージェントを一括管理するオープンソースのデスクトップ・コントロールセンターです。
[Sessionary](https://github.com/onlynor/Sessionary) を Linux・macOS・Windows 向けのネイティブアプリとしてパッケージ
化したもので、各エージェントのプロジェクト、セッション、モデルルーティング、トークン使用量を、ブラウザのタブでは
なく 1 つのウィンドウの中で管理できます。

一般的な AI チャットアプリとは異なり、Sessionary Desktop は自分自身のチャットエンジンを持たず、独自の会話履歴も保存
しません。各エージェント CLI がこのコンピュータ上、あるいは SSH 経由のリモートマシン上にすでに保持している実際の
セッション履歴とファイルを読み取り、その上にエージェントカード・チャット・ターミナル・モデルルーティング・使用量と
いった、一貫したインターフェースを提供します。

## 機能

- **AI コーディングエージェント管理** —— インストール済みのエージェント CLI(Claude Code、Codex、OpenCode、Pi、
  Hermes、WorkBuddy、WorkBuddy AI、インストールされていれば
  [DeepSeek Harness](docs/ARCHITECTURE.md#deepseek-harness-integration) も)を検出し、各エージェントのセッションを
  表示します。エージェントごとに別々のターミナルや設定ファイルを管理する必要はありません。
- **プロジェクトとセッション** —— あるエージェントの過去のセッションをプロジェクトのフォルダ単位で閲覧し、再開したり、
  選んだフォルダで新しいセッションを開始したりできます。
- **ローカルとリモートのマシン** —— このコンピュータ上のエージェントを操作するほか、**SSH**(鍵認証のみ、パスワードや
  トークンは保存しません)で別のマシンを追加し、そのエージェント履歴を読み取ることもできます。リモートマシン側には
  何もインストールされません。
- **モデルプロバイダーとルーティング** —— モデルプロバイダーと API キーを一度設定すれば、Routing ページで各エージェント
  が使うモデルを、Sessionary 自身のゲートウェイを通じて選択できます。
- **トークン使用量と分析** —— Usage ページで、エージェント別・マシン別の使用量とトークン消費を確認できます。
- **ターミナルとモニター** —— プロジェクト内で実際のターミナルを開き、1 つのタブでマシン上のエージェントの活動状況を
  確認できます。

上記は現時点で実装済みの機能のみです。既知の未実装項目は
[Not yet done](docs/ARCHITECTURE.md#not-yet-done)(英語)を参照してください。

## インストールと始め方

最新版は **[GitHub Releases](https://github.com/onlynor/Sessionary-Desktop/releases)** から入手できます。各リリースには
以下が含まれます。

| プラットフォーム | ダウンロード | 補足 |
|---|---|---|
| Linux(amd64、arm64) | `.deb`、または `install.sh` 付きの `.tar.gz` | |
| macOS(Apple シリコン) | `.dmg` | Intel Mac 向けビルドは未提供 |
| Windows(amd64) | `Sessionary-Setup-<version>-amd64.exe` | arm64 向けビルドは未提供 |

パッケージは**まだコード署名・ノータリゼーションを行っていません**。macOS の Gatekeeper と Windows の SmartScreen が
初回起動時に警告を表示します([トラブルシューティング](#トラブルシューティング)を参照)。ダウンロードしたファイルは
リリースに含まれる `SHA256SUMS` を使って `sha256sum -c SHA256SUMS` で検証できます。

**インストール:**

```sh
# Linux、.deb
sudo apt install ./sessionary_<version>_amd64.deb   # または _arm64.deb

# Linux、tarball
tar xf sessionary-<version>-linux-amd64.tar.gz && ./install.sh

# macOS
# .dmg を開き、Sessionary を Applications にドラッグ

# Windows
# Sessionary-Setup-<version>-amd64.exe を実行
```

**前提条件:** Sessionary Desktop 自体にはエージェント CLI は同梱されていません。対応するエージェント
(例: [Claude Code](https://claude.com/claude-code)、[Codex](https://github.com/openai/codex)、OpenCode、Pi)を少なく
とも 1 つインストールし、通常使っているログインシェルから実行できる状態にしてください。
[DeepSeek Harness](docs/ARCHITECTURE.md#deepseek-harness-integration) は特別な設定は不要で、対応バージョンがマシンに
あれば Sessionary が自動的に見つけます。

**最初のセッション:** アプリを起動する → マシンにすでにインストールされているエージェントが自動検出される → サイド
バーで **Agents** を開き、検出されたエージェントを選ぶ → プロジェクトのフォルダを選ぶ → セッションを開始(または再開)
する。アカウント登録やサインインは不要です。

## 使い方

サイドバーは 3 つのグループに分かれています。

- **Home** と **Machines** —— 概要、およびマシンの一覧(このコンピュータと、SSH で追加したマシン)。
- **Workspace**(選択中のマシンに対するスコープ)—— **Agents**(各エージェントのカードとセッション)、**Projects**、
  **Sessions**、**Terminal**、そして **Monitor**(そのマシン上のエージェントの活動状況)。
- **Model Control** —— **Models** と **Gateway**(プロバイダーと API キーの設定)、**Routing**(各エージェントが使う
  モデルの選択)、**Usage**(トークン消費の時系列)。

**基本的な流れ:** サイドバーでマシンを選ぶ(デフォルトはこのコンピュータ)→ **Agents** を開き、検出済みのエージェント
を選ぶ → プロジェクトのフォルダを選ぶ → **Sessions** から新しいセッションを開始するか、既存のセッションを再開する。
会話は単なるターミナルではなく、Sessionary 独自のチャットビュー(ストリーミング応答、ツール呼び出し、diff、モデル
選択)の中で行われます。同じプロジェクトで実際のターミナルを開くことも、いつでもワンクリックで可能です。

**エージェントの検出と PATH。** エージェントは、その CLI が `PATH` 上で見つかる場合に「検出済み」と表示されます。
アプリ起動後にエージェントをインストールした場合や、`~/.local/bin`、npm/pnpm のグローバルディレクトリ、バージョン
マネージャー(nvm、volta、fnm、mise、asdf)経由でインストールした場合は、アプリを再起動してログインシェルの `PATH`
を再読み込みしてください。インストール済みのはずのエージェントが表示されない場合は
[トラブルシューティング](#トラブルシューティング)を参照してください。

**モデルプロバイダーとルーティング。** **Gateway**/**Models** ページでプロバイダーと API キーを一度設定し、
**Routing** で各エージェントが使うモデルを選びます。すべてのエージェントは Sessionary 自身のゲートウェイを経由して
設定済みのプロバイダーにアクセスするため、キーは 1 か所にまとめて管理できます。**Usage** では、エージェント別・
マシン別のトークン消費の内訳を確認できます。

**SSH 経由のリモートマシン。** **Machines** ページで、ホスト(または `~/.ssh/config` のエイリアス)を指定してノードを
追加します。ポートや identity ファイルの指定は任意です。Sessionary は自分の SSH 鍵で接続し——パスワードやトークンは
保存しません——そのマシン上に既にあるエージェントのセッション履歴(Claude Code、OpenCode、Pi、Hermes)を読み取るだけ
です。何かをインストールすることはありません。DeepSeek Harness はローカル専用で、リモートマシンでは使用できません。
ノードを追加する前に、指定するものと同じ host/user/port/identity で `ssh user@host true` がパスワード入力なしで
成功することを確認してください。

日常的な使い方を超える実装の詳細は、[docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) や
[docs/DESKTOP_RUNTIME.md](docs/DESKTOP_RUNTIME.md)(英語)を参照してください。

## トラブルシューティング

- **インストール済みのエージェントが表示されない。** アプリが読み取るのはログインシェルの `PATH` であり、アプリが
  起動された方法の環境ではありません。まず新しいターミナルでそのエージェントが実行できることを確認し
  (`which <agent>`)、Sessionary Desktop を完全に終了して再起動し、環境を再読み込みさせてください。
- **macOS で「壊れている」または「開発元が未確認」と表示される。** 現在のビルドはまだノータリゼーションされていません。
  アプリを右クリックして **開く** を選び、表示されるダイアログで確認してください。
- **Windows SmartScreen がインストーラーをブロックする。** **詳細情報** → **実行** を選んでください。ビルドが署名
  されるまでは想定どおりの動作です。
- **リモートマシンの追加に失敗する、または止まる。** まず手動で接続を確認してください。入力する host/user/port/
  identity と同じ組み合わせで `ssh user@host true` を実行し、パスワード入力なしで成功する必要があります。SSH 接続を
  修正してから、ノードを追加し直してください。
- **ウィンドウを閉じるとアプリが消える(Linux)。** Linux では、StatusNotifierItem host(AppIndicator 拡張/モジュール
  を備えたほとんどのステータスバー)が動作している場合のみ、ウィンドウを閉じるとトレイに隠れます。host がない場合は、
  ウィンドウを閉じるとアプリも終了します——これは想定どおりの動作で、不具合ではありません。アプリランチャーから再度
  開いてください。
- **アプリが起動しない / 「Backend unavailable」と表示される。** その画面の **Show Details** で内蔵サーバーの出力を
  確認するか、`~/.config/Sessionary/logs/backend.log`(Linux/macOS のパス。Windows では対応するユーザーごとの設定
  ディレクトリ)を確認してください。**Try Again** で再起動できます。それでも失敗し続ける場合は、そのログを添えて
  issue を作成してください。

## プロジェクト構成と技術概要

Sessionary Desktop は [MyGo](https://github.com/egoist/mygo) でビルドされた単一の Go バイナリで、Sessionary 自身の
Web クライアント(隣接する [`Sessionary`](https://github.com/onlynor/Sessionary) のチェックアウトからビルド)を
組み込み、ネイティブの WebKit/WebView2 ウィンドウ内のループバックポートから提供します。監視対象のバックグラウンド
プロセスが、同梱の Sessionary サーバー(Node.js も同梱されているため、マシンに別途インストールする必要はありません)
を実行します。ウィンドウはブラウザがセルフホストの Sessionary と通信するのと同じ方法でサーバーと通信します。

```
cmd/sessionary-desktop/   アプリ本体: ライフサイクル、ウィンドウ、トレイ、メニュー
internal/{backend,uiserver,assets}/
frontend/                 その場でビルドされる Sessionary の Web クライアント + デスクトップ UI レイヤー
scripts/, .github/        ビルド/リリーススクリプトと CI
```

詳細(英語):[Architecture](docs/ARCHITECTURE.md) ·
[Desktop runtime(バックエンドのライフサイクル、ウィンドウ、トレイ)](docs/DESKTOP_RUNTIME.md) ·
[Releases and versioning](docs/RELEASES.md)。

## 開発とコントリビューション

必要なもの: Go 1.27+、[Bun](https://bun.sh)、Node.js、WebKitGTK 4.1(Linux)、そして `pnpm install` を実行済みの
隣接する [Sessionary](https://github.com/onlynor/Sessionary) のチェックアウト。

```sh
bun install --cwd frontend
bun run dev         # ホットリロード対応の開発ビルド
bun run build       # 本番ビルド → build/<os>-<arch>/
bun run typecheck
go test ./... && bun run --cwd frontend test
```

Issue や pull request を歓迎します。ビルド・テスト・CI の詳細は
[docs/DEVELOPMENT.md](docs/DEVELOPMENT.md)(英語)、リリースの切り方・公開手順は
[docs/RELEASES.md](docs/RELEASES.md)(英語)を参照してください。

## ライセンスとリンク

Sessionary Desktop は [Apache License 2.0](LICENSE) の下でオープンソースとして公開されています。このバイナリには
Node.js ランタイムが同梱されており、それぞれ別のライセンスを持つ Go のライブラリも静的リンクしています。詳細は
[THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md)(英語。各パッケージで実行ファイルと同じ場所に同梱されます)を
参照してください。

- [GitHub Releases](https://github.com/onlynor/Sessionary-Desktop/releases)
- [Sessionary](https://github.com/onlynor/Sessionary) —— 本アプリが同梱する Web クライアントとサーバー
- [Issues](https://github.com/onlynor/Sessionary-Desktop/issues)
