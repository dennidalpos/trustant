import test from 'node:test'
import assert from 'node:assert/strict'
import { existsSync, mkdirSync, mkdtempSync, rmSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'

import { PiAcpAgent } from '../../src/acp/agent.js'
import { FakeAgentSideConnection, asAgentConn } from '../helpers/fakes.js'

function writeSession(root: string, sessionId: string): string {
  const dir = join(root, 'sessions', 'delete-tests')
  mkdirSync(dir, { recursive: true })
  const file = join(dir, `${sessionId}.jsonl`)
  writeFileSync(
    file,
    `${JSON.stringify({
      type: 'session',
      version: 3,
      id: sessionId,
      timestamp: '2026-07-23T10:00:00.000Z',
      cwd: '/tmp/delete-test'
    })}\n`,
    'utf8'
  )
  return file
}

test('PiAcpAgent: advertises and safely implements session/delete', async () => {
  const root = mkdtempSync(join(tmpdir(), 'pi-acp-delete-'))
  const oldAgentDir = process.env.PI_CODING_AGENT_DIR
  process.env.PI_CODING_AGENT_DIR = root

  try {
    const agent = new PiAcpAgent(asAgentConn(new FakeAgentSideConnection()))
    const initialized = await agent.initialize({ protocolVersion: 1 } as any)
    assert.ok(initialized.agentCapabilities)
    assert.deepEqual(initialized.agentCapabilities.sessionCapabilities?.delete, {})

    const deletableFile = writeSession(root, 'sess-delete')
    await agent.deleteSession({ sessionId: 'sess-delete' } as any)
    assert.equal(existsSync(deletableFile), false)

    await assert.rejects(() => agent.deleteSession({ sessionId: 'sess-missing' } as any), /Unknown sessionId/)

    const activeFile = writeSession(root, 'sess-active')
    // WHY: exercise the public deletion guard without spawning Pi; the
    // SessionManager registry is the authoritative active-session signal.
    ;(agent as any).sessions.sessions.set('sess-active', {})
    await assert.rejects(() => agent.deleteSession({ sessionId: 'sess-active' } as any), /Cannot delete active session/)
    assert.equal(existsSync(activeFile), true)
  } finally {
    if (oldAgentDir === undefined) delete process.env.PI_CODING_AGENT_DIR
    else process.env.PI_CODING_AGENT_DIR = oldAgentDir
    rmSync(root, { recursive: true, force: true })
  }
})
