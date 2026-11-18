# Trustant fork

This fork starts from upstream
[`svkozak/pi-acp`](https://github.com/svkozak/pi-acp) commit
`9e857dcc05a057404eb1537e5f31e5aef88a5863`, published as `pi-acp@0.0.31`.
The `upstream` Git remote remains configured so upstream fixes can be reviewed
and merged deliberately.

## Maintained Trustant changes

- Negotiate a versioned `_meta.trustant.piLaunch` contract and translate only
  validated extension, skill, prompt-template, and session-directory fields
  into discrete Pi arguments. Arbitrary arguments and shell fragments are not
  accepted.
- Include Pi extension commands in the ACP command list and preserve their
  input hints.
- Publish versioned activity metadata for thinking, responding, tools, retries,
  compaction, stopping, and idle states. Retry and compaction status remains
  control-plane metadata instead of appearing as assistant prose.
- Bound Pi abort requests so Stop has a deterministic adapter-side timeout.
- Derive the advertised `thought_level` choices from Pi's
  `get_available_thinking_levels`, falling back to the active model metadata
  Pi already returns, and report the level Pi actually applied instead of the
  one requested. Pi clamps silently and still reports success, so a fixed list
  told ACP clients a reasoning effort was honored when it was not.
- Advertise standard ACP `session/delete`, resolve deletion by session ID
  inside Pi's configured session root, and reject deletion of a loaded session.
- Honor `PI_SKIP_VERSION_CHECK` and `PI_OFFLINE` when producing the adapter
  update notice.

The Trustant parent repositories package this source directly. There is no
fallback to the public `pi-acp` npm release because doing so would silently
remove these capabilities.

## Upstream synchronization

1. Fetch `upstream`.
2. Review upstream changes after the baseline above.
3. Rebase or merge on a dedicated branch.
4. Run `npm test`, `npm run typecheck`, `npm run lint`, and `npm run build`.
5. Revalidate Stop, activity, extension commands, session resume, and session
   deletion through Trustant before updating the parent submodule pointer.

## Licensing

The upstream MIT license and copyright notice are preserved in
[`LICENSE`](LICENSE). The Trustant changes remain distributed under the same
license.
