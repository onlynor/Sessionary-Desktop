/**
 * The order agents appear in on a machine's Agents tab: an explicit list, never the server's registry order,
 * the filesystem's or the alphabet's. The grid lays them out four to a row (two, then one, on narrower windows),
 * so the list reads as two rows:
 *
 *   Claude Code · Codex · DeepSeek Harness · Pi
 *   OpenCode · Hermes · WorkBuddy · WorkBuddy AI
 *
 * Agents this list does not name (a newer Sessionary's) follow, in the order the server gives them.
 */
export const AGENT_DISPLAY_ORDER = ['claude-code', 'codex', 'deepseek-harness', 'pi', 'opencode', 'hermes', 'workbuddy', 'workbuddy-ai'] as const

export function orderAgents<T>(items: T[], id: (item: T) => string): T[] {
  const rank = (x: T) => {
    const i = (AGENT_DISPLAY_ORDER as readonly string[]).indexOf(id(x))
    return i < 0 ? AGENT_DISPLAY_ORDER.length : i
  }
  // a stable sort keeps the server's order among agents the list does not name
  return items.map((x, i) => ({ x, i })).sort((a, b) => rank(a.x) - rank(b.x) || a.i - b.i).map((e) => e.x)
}
