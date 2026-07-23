import { isAbsolute, resolve } from 'node:path'

export const TRUSTABLE_PI_LAUNCH_VERSION = 1 as const

export type EffectivePiLaunchConfig = {
  version: typeof TRUSTABLE_PI_LAUNCH_VERSION
  extensions: { discover: boolean; paths: string[] }
  skills: { discover: boolean; paths: string[] }
  promptTemplates: { paths: string[] }
  sessionDir?: string
}

function record(value: unknown): Record<string, unknown> | null {
  return value && typeof value === 'object' && !Array.isArray(value) ? (value as Record<string, unknown>) : null
}

function paths(value: unknown, cwd: string, field: string): string[] {
  if (value === undefined) return []
  if (!Array.isArray(value) || value.some(item => typeof item !== 'string' || !item.trim())) {
    throw new Error(`${field} must be an array of non-empty paths`)
  }
  return value.map(item => (isAbsolute(item) ? item : resolve(cwd, item)))
}

function optionalBoolean(value: unknown, field: string, fallback: boolean): boolean {
  if (value === undefined) return fallback
  if (typeof value !== 'boolean') throw new Error(`${field} must be a boolean`)
  return value
}

/**
 * Read the versioned Trustable launch extension without accepting raw argv.
 *
 * WHY: ACP clients must not replace adapter-owned flags such as `--mode rpc`;
 * converting a small typed object here keeps process launch deterministic and
 * prevents shell/argument injection.
 */
export function parseTrustablePiLaunch(meta: unknown, cwd: string): EffectivePiLaunchConfig {
  const root = record(meta)
  const trustable = record(root?.trustable)
  const raw = record(trustable?.piLaunch)

  if (!raw) {
    return {
      version: TRUSTABLE_PI_LAUNCH_VERSION,
      extensions: { discover: true, paths: [] },
      skills: { discover: true, paths: [] },
      promptTemplates: { paths: [] }
    }
  }

  if (raw.version !== TRUSTABLE_PI_LAUNCH_VERSION) {
    throw new Error(
      `Unsupported trustable.piLaunch version: ${String(raw.version)} (expected ${TRUSTABLE_PI_LAUNCH_VERSION})`
    )
  }

  const extensions = record(raw.extensions)
  const skills = record(raw.skills)
  const promptTemplates = record(raw.promptTemplates)
  const rawSessionDir = raw.sessionDir

  if (rawSessionDir !== undefined && (typeof rawSessionDir !== 'string' || !rawSessionDir.trim())) {
    throw new Error('trustable.piLaunch.sessionDir must be a non-empty path')
  }

  return {
    version: TRUSTABLE_PI_LAUNCH_VERSION,
    extensions: {
      discover: optionalBoolean(extensions?.discover, 'trustable.piLaunch.extensions.discover', true),
      paths: paths(extensions?.paths, cwd, 'trustable.piLaunch.extensions.paths')
    },
    skills: {
      discover: optionalBoolean(skills?.discover, 'trustable.piLaunch.skills.discover', true),
      paths: paths(skills?.paths, cwd, 'trustable.piLaunch.skills.paths')
    },
    promptTemplates: {
      paths: paths(promptTemplates?.paths, cwd, 'trustable.piLaunch.promptTemplates.paths')
    },
    ...(typeof rawSessionDir === 'string'
      ? { sessionDir: isAbsolute(rawSessionDir) ? rawSessionDir : resolve(cwd, rawSessionDir) }
      : {})
  }
}

/**
 * Convert the validated object into separate argv entries.
 *
 * WHY: keeping every value as its own spawn argument preserves spaces in paths
 * and deliberately offers no escape hatch for arbitrary Pi CLI flags.
 */
export function piLaunchArgs(config: EffectivePiLaunchConfig): string[] {
  const args: string[] = []
  if (!config.extensions.discover) args.push('--no-extensions')
  for (const path of config.extensions.paths) args.push('--extension', path)
  if (!config.skills.discover) args.push('--no-skills')
  for (const path of config.skills.paths) args.push('--skill', path)
  for (const path of config.promptTemplates.paths) args.push('--prompt-template', path)
  if (config.sessionDir) args.push('--session-dir', config.sessionDir)
  return args
}
