import test from 'node:test'
import assert from 'node:assert/strict'
import { parseTrustablePiLaunch, piLaunchArgs } from '../../src/pi-rpc/launch-config.js'

test('Trustable Pi launch config defaults preserve normal discovery', () => {
  const parsed = parseTrustablePiLaunch(undefined, '/workspace/app')
  assert.deepEqual(parsed, {
    version: 1,
    extensions: { discover: true, paths: [] },
    skills: { discover: true, paths: [] },
    promptTemplates: { paths: [] }
  })
  assert.deepEqual(piLaunchArgs(parsed), [])
})

test('Trustable Pi launch config produces discrete validated argv entries', () => {
  const parsed = parseTrustablePiLaunch(
    {
      trustable: {
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

test('Trustable Pi launch config rejects unknown versions', () => {
  assert.throws(
    () => parseTrustablePiLaunch({ trustable: { piLaunch: { version: 2 } } }, '/workspace/app'),
    /Unsupported trustable\.piLaunch version/
  )
})
