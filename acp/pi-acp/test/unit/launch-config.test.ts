import test from 'node:test'
import assert from 'node:assert/strict'
import { parseTrustantPiLaunch, piLaunchArgs } from '../../src/pi-rpc/launch-config.js'

test('Trustant Pi launch config defaults preserve normal discovery', () => {
  const parsed = parseTrustantPiLaunch(undefined, '/workspace/app')
  assert.deepEqual(parsed, {
    version: 1,
    extensions: { discover: true, paths: [] },
    skills: { discover: true, paths: [] },
    promptTemplates: { paths: [] }
  })
  assert.deepEqual(piLaunchArgs(parsed), [])
})

test('Trustant Pi launch config produces discrete validated argv entries', () => {
  const parsed = parseTrustantPiLaunch(
    {
      trustant: {
        piLaunch: {
          version: 1,
          extensions: { discover: false, paths: ['extensions/my extension.ts'] },
          skills: { discover: false, paths: ['/shared/skills'] },
          promptTemplates: { paths: ['prompts'] },
          sessionDir: '.pi-sessions'
        }
      }
    },
    '/workspace/app'
  )

  assert.deepEqual(piLaunchArgs(parsed), [
    '--no-extensions',
    '--extension',
    '/workspace/app/extensions/my extension.ts',
    '--no-skills',
    '--skill',
    '/shared/skills',
    '--prompt-template',
    '/workspace/app/prompts',
    '--session-dir',
    '/workspace/app/.pi-sessions'
  ])
})

test('Trustant Pi launch config forwards the MCP project server policy only as "allow"', () => {
  const parsed = parseTrustantPiLaunch({ trustant: { piLaunch: { version: 1, mcpProjectServers: 'allow' } } }, '/workspace/app')
  assert.deepEqual(piLaunchArgs(parsed), ['--mcp-project-servers', 'allow'])
  assert.throws(
    () => parseTrustantPiLaunch({ trustant: { piLaunch: { version: 1, mcpProjectServers: 'ask' } } }, '/workspace/app'),
    /mcpProjectServers must be "allow"/
  )
})

test('Trustant Pi launch config rejects unknown versions', () => {
  assert.throws(
    () => parseTrustantPiLaunch({ trustant: { piLaunch: { version: 2 } } }, '/workspace/app'),
    /Unsupported trustant\.piLaunch version/
  )
})
