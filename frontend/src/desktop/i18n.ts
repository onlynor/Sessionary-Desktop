import { DICT } from '@web/locales'

// The desktop layer's own strings, merged into Sessionary's dictionary so that `t()` translates them like its own.
// Each row: English key, 简体中文, 繁體中文, 日本語.
const ROWS: [string, string, string, string][] = [
  ['Starting…', '正在启动…', '正在啟動…', '起動しています…'],
  ['Starting Sessionary…', '正在启动 Sessionary…', '正在啟動 Sessionary…', 'Sessionary を起動しています…'],
  ['Reading your agents’ sessions…', '正在读取 Agent 的会话…', '正在讀取 Agent 的工作階段…', 'エージェントのセッションを読み込んでいます…'],
  ['Almost ready…', '马上就好…', '馬上就好…', 'もうすぐです…'],
  ['Sessionary stopped — restarting…', 'Sessionary 已停止 — 正在重新启动…', 'Sessionary 已停止 — 正在重新啟動…', 'Sessionary が停止しました — 再起動しています…'],
  ['the Node.js runtime is missing from this installation; reinstall Sessionary', '此安装缺少 Node.js 运行时，请重新安装 Sessionary', '此安裝缺少 Node.js 執行環境，請重新安裝 Sessionary', 'このインストールには Node.js ランタイムがありません。Sessionary を再インストールしてください'],
  ['Sessionary couldn’t start', 'Sessionary 无法启动', 'Sessionary 無法啟動', 'Sessionary を起動できませんでした'],
  ['Try Again', '重试', '重試', '再試行'],
  ['Show Details', '显示详细信息', '顯示詳細資訊', '詳細を表示'],
  ['Hide Details', '隐藏详细信息', '隱藏詳細資訊', '詳細を隠す'],
  ['Nothing was printed.', '没有任何输出。', '沒有任何輸出。', '出力はありません。'],
  ['Back', '后退', '返回', '戻る'],
  ['Connecting…', '正在连接…', '正在連線…', '接続しています…'],
  ['Reconnecting', '正在重新连接', '正在重新連線', '再接続しています'],
  ['Reconnecting…', '正在重新连接…', '正在重新連線…', '再接続しています…'],
  ['Sessionary isn’t responding…', 'Sessionary 没有响应…', 'Sessionary 沒有回應…', 'Sessionary が応答していません…'],
  ['Restarting Sessionary', '正在重新启动 Sessionary', '正在重新啟動 Sessionary', 'Sessionary を再起動しています'],
  ['Restarting Sessionary…', '正在重新启动 Sessionary…', '正在重新啟動 Sessionary…', 'Sessionary を再起動しています…'],
  ['Sessionary stopped. Starting it again…', 'Sessionary 已停止，正在重新启动…', 'Sessionary 已停止，正在重新啟動…', 'Sessionary が停止しました。再起動しています…'],
  ['Sessionary stopped. Starting it again (attempt {n})…', 'Sessionary 已停止，正在重新启动（第 {n} 次）…', 'Sessionary 已停止，正在重新啟動（第 {n} 次）…', 'Sessionary が停止しました。再起動しています（{n} 回目）…'],
  ['Sessionary is unavailable', 'Sessionary 不可用', 'Sessionary 無法使用', 'Sessionary を利用できません'],
  ['Quit', '退出', '結束', '終了'],
  ['Quitting…', '正在退出…', '正在結束…', '終了しています…'],
  ['The full log is in {path}', '完整日志位于 {path}', '完整記錄位於 {path}', '完全なログ: {path}'],
  ['Window', '窗口', '視窗', 'ウインドウ'],
  ['Another Sessionary is using the same data (for example `sessionary` running in a terminal). Quit it, then try again.', '另一个 Sessionary 正在使用同一份数据（例如在终端中运行的 `sessionary`）。请先退出它，然后重试。', '另一個 Sessionary 正在使用同一份資料（例如在終端機中執行的 `sessionary`）。請先結束它，然後重試。', '別の Sessionary が同じデータを使っています（ターミナルで実行中の `sessionary` など）。それを終了してから再試行してください。'],
  ['Waiting for the development server on port {port}…', '正在等待端口 {port} 上的开发服务器…', '正在等待連接埠 {port} 上的開發伺服器…', 'ポート {port} の開発サーバーを待っています…'],
  ['Close', '关闭', '關閉', '閉じる'],
  ['Minimize', '最小化', '最小化', '最小化'],
  ['Maximize', '最大化', '最大化', '最大化'],
  ['Restore', '还原', '還原', '元に戻す'],
  ['Forward', '前进', '前進', '進む'],
  ['Show Sessionary', '显示 Sessionary', '顯示 Sessionary', 'Sessionary を表示'],
  ['Hide Sessionary', '隐藏 Sessionary', '隱藏 Sessionary', 'Sessionary を隠す'],
  ['Recent sessions', '最近的会话', '最近的工作階段', '最近のセッション'],
  ['About Sessionary', '关于 Sessionary', '關於 Sessionary', 'Sessionary について'],
  ['Quit Sessionary', '退出 Sessionary', '結束 Sessionary', 'Sessionary を終了'],
  ['Running', '运行中', '執行中', '実行中'],
]

const COLS = ['zh-Hans', 'zh-Hant', 'ja'] as const

export function registerStrings() {
  const dict = DICT as Record<string, Record<string, string>>
  for (const [en, ...tr] of ROWS) {
    COLS.forEach((lang, i) => {
      dict[lang] ??= {}
      dict[lang]![en] ??= tr[i]!
    })
  }
}
