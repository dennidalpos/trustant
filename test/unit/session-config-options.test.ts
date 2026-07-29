import test from 'node:test'
import assert from 'node:assert/strict'
import { PiAcpAgent } from '../../src/acp/agent.js'
import { FakeAgentSideConnection, asAgentConn } from '../helpers/fakes.js'

class FakeSessions {
  constructor(private readonly session: any) {}

  async create() {
    return this.session
  }

  maybeGet(sessionId: string) {
    if (sessionId !== this.session.sessionId) return undefined
    return this.session
  }

  get(sessionId: string) {
    if (sessionId !== this.session.sessionId) {
      throw new Error(`Unknown sessionId: ${sessionId}`)
    }
    return this.session
  }
}

const LEVEL_ORDER = ['off', 'minimal', 'low', 'medium', 'high', 'xhigh']

// Pi clamps set_thinking_level to the active model's capabilities and still
// reports success. The fakes below reproduce that so the adapter is exercised
// against the behavior that produced the false compatibility error, not an
// idealized process that honors every level.
function clamp(requested: string, available: string[]): string {
  if (available.includes(requested)) return requested
  const idx = LEVEL_ORDER.indexOf(requested)
  if (idx !== -1) {
    for (let i = idx; i < LEVEL_ORDER.length; i++) {
      if (available.includes(LEVEL_ORDER[i])) return LEVEL_ORDER[i]
    }
    for (let i = idx - 1; i >= 0; i--) {
      if (available.includes(LEVEL_ORDER[i])) return LEVEL_ORDER[i]
    }
  }
  return available[0] ?? 'off'
}

function makePiProc(args: {
  state: any
  models?: Array<{ provider: string; id: string; name: string }>
  levels?: string[] | null
  setThinkingLevelCalls?: string[]
}) {
  const models = args.models ?? [{ provider: 'test', id: 'alpha', name: 'Alpha' }]

  return {
    async getAvailableModels() {
      return { models }
    },
    async getState() {
      return args.state
    },
    async getAvailableThinkingLevels() {
      if (!args.levels) throw new Error('pi get_available_thinking_levels failed: Unknown command')
      return { levels: args.levels }
    },
    async setThinkingLevel(level: string) {
      args.setThinkingLevelCalls?.push(level)
      args.state.thinkingLevel = args.levels ? clamp(level, args.levels) : level
    }
  }
}

function thoughtLevelValues(configOptions: any[]): string[] {
  const option = configOptions.find(candidate => candidate.id === 'thought_level')
  return option.options.map((choice: any) => choice.value)
}

test('PiAcpAgent: newSession returns configOptions for model and thinking selectors', async () => {
  const realSetTimeout = globalThis.setTimeout
  ;(globalThis as any).setTimeout = () => 0 as any

  try {
    const conn = new FakeAgentSideConnection()
    const session = {
      sessionId: 's1',
      cwd: process.cwd(),
      proc: makePiProc({
        state: { thinkingLevel: 'high', model: { provider: 'test', id: 'beta' } },
        models: [
          { provider: 'test', id: 'alpha', name: 'Alpha' },
          { provider: 'test', id: 'beta', name: 'Beta' }
        ],
        levels: ['off', 'minimal', 'low', 'medium', 'high']
      }),
      setStartupInfo() {},
      sendStartupInfoIfPending() {}
    }

    const agent = new PiAcpAgent(asAgentConn(conn), {} as any)
    ;(agent as any).sessions = new FakeSessions(session) as any

    const result = await agent.newSession({ cwd: process.cwd(), mcpServers: [] } as any)

    assert.equal(result.models?.currentModelId, 'test/beta')
    assert.equal(result.modes?.currentModeId, 'high')
    assert.deepEqual(result.configOptions, [
      {
        type: 'select',
        id: 'model',
        category: 'model',
        name: 'Model',
        description: 'Select the model for this session',
        currentValue: 'test/beta',
        options: [
          { value: 'test/alpha', name: 'test/Alpha', description: null },
          { value: 'test/beta', name: 'test/Beta', description: null }
        ]
      },
      {
        type: 'select',
        id: 'thought_level',
        category: 'thought_level',
        name: 'Thinking',
        description: 'Set the reasoning effort for this session',
        currentValue: 'high',
        options: [
          { value: 'off', name: 'Thinking: off', description: null },
          { value: 'minimal', name: 'Thinking: minimal', description: null },
          { value: 'low', name: 'Thinking: low', description: null },
          { value: 'medium', name: 'Thinking: medium', description: null },
          { value: 'high', name: 'Thinking: high', description: null }
        ]
      }
    ])
  } finally {
    ;(globalThis as any).setTimeout = realSetTimeout
  }
})

test('PiAcpAgent: thought level choices never include a level pi reports outside the ACP surface', async () => {
  const conn = new FakeAgentSideConnection()
  const session = {
    sessionId: 's1',
    cwd: process.cwd(),
    proc: makePiProc({
      state: { thinkingLevel: 'high', model: { provider: 'test', id: 'alpha' } },
      levels: ['off', 'minimal', 'low', 'medium', 'high', 'xhigh', 'max']
    })
  }

  const agent = new PiAcpAgent(asAgentConn(conn), {} as any)
  ;(agent as any).sessions = new FakeSessions(session) as any

  const result = await agent.setSessionConfigOption({
    sessionId: 's1',
    configId: 'thought_level',
    value: 'xhigh'
  } as any)

  assert.deepEqual(thoughtLevelValues(result.configOptions), ['off', 'minimal', 'low', 'medium', 'high', 'xhigh'])
})

test('PiAcpAgent: thought level choices fall back to model metadata when pi lacks the levels command', async () => {
  const conn = new FakeAgentSideConnection()
  const session = {
    sessionId: 's1',
    cwd: process.cwd(),
    proc: makePiProc({
      state: {
        thinkingLevel: 'high',
        model: { provider: 'test', id: 'alpha', reasoning: true, thinkingLevelMap: { xhigh: 'high' } }
      },
      levels: null
    })
  }

  const agent = new PiAcpAgent(asAgentConn(conn), {} as any)
  ;(agent as any).sessions = new FakeSessions(session) as any

  const result = await agent.setSessionConfigOption({
    sessionId: 's1',
    configId: 'thought_level',
    value: 'high'
  } as any)

  assert.deepEqual(thoughtLevelValues(result.configOptions), ['off', 'minimal', 'low', 'medium', 'high', 'xhigh'])
})

test('PiAcpAgent: thought level choices drop a level the model metadata disables', async () => {
  const conn = new FakeAgentSideConnection()
  const session = {
    sessionId: 's1',
    cwd: process.cwd(),
    proc: makePiProc({
      state: {
        thinkingLevel: 'high',
        model: { provider: 'test', id: 'alpha', reasoning: true, thinkingLevelMap: { minimal: null } }
      },
      levels: null
    })
  }

  const agent = new PiAcpAgent(asAgentConn(conn), {} as any)
  ;(agent as any).sessions = new FakeSessions(session) as any

  const result = await agent.setSessionConfigOption({
    sessionId: 's1',
    configId: 'thought_level',
    value: 'high'
  } as any)

  assert.deepEqual(thoughtLevelValues(result.configOptions), ['off', 'low', 'medium', 'high'])
})

test('PiAcpAgent: a non-reasoning model advertises only off and reports off as current', async () => {
  // Regression: the adapter used to advertise high for a model pi collapses to
  // off, so a client verifying the echoed value saw a false compatibility error.
  const conn = new FakeAgentSideConnection()
  const session = {
    sessionId: 's1',
    cwd: process.cwd(),
    proc: makePiProc({
      state: { thinkingLevel: 'high', model: { provider: 'test', id: 'alpha', reasoning: false } },
      levels: null
    })
  }

  const agent = new PiAcpAgent(asAgentConn(conn), {} as any)
  ;(agent as any).sessions = new FakeSessions(session) as any

  const result = await agent.setSessionConfigOption({
    sessionId: 's1',
    configId: 'thought_level',
    value: 'high'
  } as any)

  assert.deepEqual(thoughtLevelValues(result.configOptions), ['off'])
  assert.equal(result.configOptions.find(option => option.id === 'thought_level')?.currentValue, 'off')
})

test('PiAcpAgent: thought level choices fall back to the full list when pi reports nothing usable', async () => {
  const conn = new FakeAgentSideConnection()
  const session = {
    sessionId: 's1',
    cwd: process.cwd(),
    proc: {
      async getAvailableModels() {
        return { models: [{ provider: 'test', id: 'alpha', name: 'Alpha' }] }
      },
      async getState(): Promise<unknown> {
        throw new Error('pi get_state failed')
      },
      async getAvailableThinkingLevels(): Promise<unknown> {
        throw new Error('pi get_available_thinking_levels failed: Unknown command')
      },
      async setThinkingLevel() {}
    }
  }

  const agent = new PiAcpAgent(asAgentConn(conn), {} as any)
  ;(agent as any).sessions = new FakeSessions(session) as any

  const result = await agent.setSessionConfigOption({
    sessionId: 's1',
    configId: 'thought_level',
    value: 'high'
  } as any)

  assert.deepEqual(thoughtLevelValues(result.configOptions), ['off', 'minimal', 'low', 'medium', 'high', 'xhigh'])
  assert.equal(result.configOptions.find(option => option.id === 'thought_level')?.currentValue, 'medium')
})

test('PiAcpAgent: setSessionConfigOption maps model changes to pi and emits config_option_update', async () => {
  const conn = new FakeAgentSideConnection()
  const state = {
    thinkingLevel: 'medium',
    model: { provider: 'test', id: 'alpha' }
  }
  const setModelCalls: Array<{ provider: string; modelId: string }> = []

  const session = {
    sessionId: 's1',
    cwd: process.cwd(),
    proc: {
      async getAvailableModels() {
        return {
          models: [
            { provider: 'test', id: 'alpha', name: 'Alpha' },
            { provider: 'test', id: 'beta', name: 'Beta' }
          ]
        }
      },
      async getState() {
        return state
      },
      async getAvailableThinkingLevels() {
        return { levels: ['off', 'minimal', 'low', 'medium', 'high'] }
      },
      async setModel(provider: string, modelId: string) {
        setModelCalls.push({ provider, modelId })
        state.model = { provider, id: modelId }
      }
    }
  }

  const agent = new PiAcpAgent(asAgentConn(conn), {} as any)
  ;(agent as any).sessions = new FakeSessions(session) as any

  const result = await agent.setSessionConfigOption({
    sessionId: 's1',
    configId: 'model',
    value: 'test/beta'
  } as any)

  assert.deepEqual(setModelCalls, [{ provider: 'test', modelId: 'beta' }])
  assert.equal(result.configOptions.find(option => option.id === 'model')?.currentValue, 'test/beta')
  assert.deepEqual(conn.updates, [
    {
      sessionId: 's1',
      update: {
        sessionUpdate: 'config_option_update',
        configOptions: result.configOptions
      }
    }
  ])
})

test('PiAcpAgent: setSessionConfigOption maps thought level changes to pi and emits sync updates', async () => {
  const conn = new FakeAgentSideConnection()
  const thinkingLevels: string[] = []

  const session = {
    sessionId: 's1',
    cwd: process.cwd(),
    proc: makePiProc({
      state: { thinkingLevel: 'medium', model: { provider: 'test', id: 'alpha' } },
      levels: ['off', 'minimal', 'low', 'medium', 'high', 'xhigh'],
      setThinkingLevelCalls: thinkingLevels
    })
  }

  const agent = new PiAcpAgent(asAgentConn(conn), {} as any)
  ;(agent as any).sessions = new FakeSessions(session) as any

  const result = await agent.setSessionConfigOption({
    sessionId: 's1',
    configId: 'thought_level',
    value: 'xhigh'
  } as any)

  assert.deepEqual(thinkingLevels, ['xhigh'])
  assert.equal(result.configOptions.find(option => option.id === 'thought_level')?.currentValue, 'xhigh')
  assert.deepEqual(conn.updates, [
    {
      sessionId: 's1',
      update: {
        sessionUpdate: 'current_mode_update',
        currentModeId: 'xhigh'
      }
    },
    {
      sessionId: 's1',
      update: {
        sessionUpdate: 'config_option_update',
        configOptions: result.configOptions
      }
    }
  ])
})

test('PiAcpAgent: setSessionConfigOption reports the level pi applied when pi clamps the request', async () => {
  const conn = new FakeAgentSideConnection()
  const thinkingLevels: string[] = []

  const session = {
    sessionId: 's1',
    cwd: process.cwd(),
    proc: makePiProc({
      state: { thinkingLevel: 'medium', model: { provider: 'test', id: 'alpha' } },
      levels: ['off', 'minimal', 'low', 'medium', 'high'],
      setThinkingLevelCalls: thinkingLevels
    })
  }

  const agent = new PiAcpAgent(asAgentConn(conn), {} as any)
  ;(agent as any).sessions = new FakeSessions(session) as any

  const result = await agent.setSessionConfigOption({
    sessionId: 's1',
    configId: 'thought_level',
    value: 'xhigh'
  } as any)

  assert.deepEqual(thinkingLevels, ['xhigh'])
  assert.equal(result.configOptions.find(option => option.id === 'thought_level')?.currentValue, 'high')
  assert.deepEqual(conn.updates[0], {
    sessionId: 's1',
    update: {
      sessionUpdate: 'current_mode_update',
      currentModeId: 'high'
    }
  })
})

test('PiAcpAgent: setSessionMode reports the level pi applied when pi clamps the request', async () => {
  const conn = new FakeAgentSideConnection()
  const thinkingLevels: string[] = []

  const session = {
    sessionId: 's1',
    cwd: process.cwd(),
    proc: makePiProc({
      state: { thinkingLevel: 'medium', model: { provider: 'test', id: 'alpha' } },
      levels: ['off', 'minimal', 'low', 'medium', 'high'],
      setThinkingLevelCalls: thinkingLevels
    })
  }

  const agent = new PiAcpAgent(asAgentConn(conn), {} as any)
  ;(agent as any).sessions = new FakeSessions(session) as any

  await agent.setSessionMode({ sessionId: 's1', modeId: 'xhigh' } as any)

  assert.deepEqual(thinkingLevels, ['xhigh'])
  assert.deepEqual(conn.updates[0], {
    sessionId: 's1',
    update: {
      sessionUpdate: 'current_mode_update',
      currentModeId: 'high'
    }
  })
})
