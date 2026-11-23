import { isAbsolute, resolve } from 'node:path'

export const TRUSTANT_PI_LAUNCH_VERSION = 1 as const

export type EffectivePiLaunchConfig = {
  version: typeof TRUSTANT_PI_LAUNCH_VERSION
  extensions: { discover: boolean; paths: string[] }
  skills: { discover: boolean; paths: string[] }
  promptTemplates: { paths: string[] }
  sessionDir?: string
  mcpProjectServers?: 'allow'
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
 * Read the versioned Trustant launch extension without accepting raw argv.
 *
 * WHY: ACP clients must not replace adapter-owned flags such as `--mode rpc`;
 * converting a small typed object here keeps process launch deterministic and
 * prevents shell/argument injection.
 */
export function parseTrustantPiLaunch(meta: unknown, cwd: string): EffectivePiLaunchConfig {
  const root = record(meta)
  const trustant = record(root?.trustant)
  const raw = record(trustant?.piLaunch)

  if (!raw) {
    return {
      version: TRUSTANT_PI_LAUNCH_VERSION,
      extensions: { discover: true, paths: [] },
      skills: { discover: true, paths: [] },
      promptTemplates: { paths: [] }
    }
  }

  if (raw.version !== TRUSTANT_PI_LAUNCH_VERSION) {
    throw new Error(
      `Unsupported trustant.piLaunch version: ${String(raw.version)} (expected ${TRUSTANT_PI_LAUNCH_VERSION})`
    )
  }

  const extensions = record(raw.extensions)
  const skills = record(raw.skills)
  const promptTemplates = record(raw.promptTemplates)
  const rawSessionDir = raw.sessionDir

  if (rawSessionDir !== undefined && (typeof rawSessionDir !== 'string' || !rawSessionDir.trim())) {
    throw new Error('trustant.piLaunch.sessionDir must be a non-empty path')
  }

  if (raw.mcpProjectServers !== undefined && raw.mcpProjectServers !== 'allow') {
    throw new Error('trustant.piLaunch.mcpProjectServers must be "allow"')
  }

  return {
    version: TRUSTANT_PI_LAUNCH_VERSION,
    ...(raw.mcpProjectServers === 'allow' ? { mcpProjectServers: 'allow' as const } : {}),
    extensions: {
      discover: optionalBoolean(extensions?.discover, 'trustant.piLaunch.extensions.discover', true),
      paths: paths(extensions?.paths, cwd, 'trustant.piLaunch.extensions.paths')
    },
    skills: {
      discover: optionalBoolean(skills?.discover, 'trustant.piLaunch.skills.discover', true),
      paths: paths(skills?.paths, cwd, 'trustant.piLaunch.skills.paths')
    },
    promptTemplates: {
      paths: paths(promptTemplates?.paths, cwd, 'trustant.piLaunch.promptTemplates.paths')
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
  // WHY: pi-mcp-adapter 7 skips unapproved project servers in RPC mode, and the
  // managed workbench declares its host-owned servers in the project .mcp.json.
  if (config.mcpProjectServers === 'allow') args.push('--mcp-project-servers', 'allow')
  return args
}
