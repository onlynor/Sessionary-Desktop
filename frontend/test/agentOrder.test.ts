import { describe, expect, test } from 'bun:test'
import { AGENT_DISPLAY_ORDER, orderAgents } from '../src/desktop/agentOrder'

const ids = (xs: { id: string }[]) => xs.map((x) => x.id)
// the server's registry order (src/adapters/index.ts), plus the desktop's own DeepSeek Harness card at the end
const fromServer = ['claude-code', 'codex', 'opencode', 'pi', 'hermes', 'workbuddy', 'workbuddy-ai', 'deepseek-harness'].map((id) => ({ id }))

describe('agent display order', () => {
  test('two rows of four: Claude Code, Codex, DeepSeek Harness, Pi / OpenCode, Hermes, WorkBuddy, WorkBuddy AI', () => {
    const order = ids(orderAgents(fromServer, (a) => a.id))
    expect(order.slice(0, 4)).toEqual(['claude-code', 'codex', 'deepseek-harness', 'pi'])
    expect(order.slice(4, 8)).toEqual(['opencode', 'hermes', 'workbuddy', 'workbuddy-ai'])
  })

  test('OpenCode starts the second row', () => {
    expect(ids(orderAgents(fromServer, (a) => a.id)).indexOf('opencode')).toBe(4)
  })

  test('does not depend on the order it is given', () => {
    const shuffled = [...fromServer].reverse()
    expect(ids(orderAgents(shuffled, (a) => a.id))).toEqual([...AGENT_DISPLAY_ORDER])
  })

  test('agents the list does not name follow, in the order given', () => {
    const more = [{ id: 'zeta' }, ...fromServer, { id: 'alpha' }]
    expect(ids(orderAgents(more, (a) => a.id)).slice(-2)).toEqual(['zeta', 'alpha'])
  })

  test('missing agents leave no gaps', () => {
    const some = ['pi', 'workbuddy', 'claude-code'].map((id) => ({ id }))
    expect(ids(orderAgents(some, (a) => a.id))).toEqual(['claude-code', 'pi', 'workbuddy'])
  })

  test('does not change its input', () => {
    const input = [...fromServer]
    orderAgents(input, (a) => a.id)
    expect(input).toEqual(fromServer)
  })
})
