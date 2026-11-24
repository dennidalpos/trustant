Implement an image.sh script

cd to the directory of the script at the beginning

the <image> name is ghcr.io/trustant/trustant

if no arguments: delete all tags and generate one with date +%y.%j.%H%S
otherwise use the first argument as tag

Detects the architecture to build:
- if it is a Mac builds for arm64
- if it is windows builds for amd64
- if it is Linux uses arch to detect the arc to build for

- if it is a GITHUB_ACTION builds for both and use as action --push otherwise execute a --load

Selects the container runtime by sourcing `runtime.sh`: Docker when installed,
otherwise nerdctl against the k3s containerd socket in the `k8s.io` namespace,
starting buildkit if needed. Override with `TRUSTANT_CONTAINER_RUNTIME`.

Builds the whole Dockerfile in one pass, passing the target architecture. There
is no base/current split: buildkit cannot resolve `FROM` against an image that
was just loaded into containerd, and the builder's layer cache already avoids
rebuilding unchanged base stages.

The image build stages the vendored `mcp/` sources and builds runtime artifacts
from the vendored `acp/` and `acp/pi-acp/` sources before invoking Docker.

Log the content hashes of the pinned OpenServerless MCP and TruACP
revisions/content for build provenance.

The base image installs the staged TruACP JavaScript bundle and the built
`pi-acp` tarball. Its installer verifies the upstream Pi packages from
[`acp/pi.version`](../acp/pi.version) against
[`acp/pi.integrity`](../acp/pi.integrity) before installing them from npm.
