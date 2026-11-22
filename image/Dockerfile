ARG OLLAMA_VERSION=0.40.16
# NOTE: there is deliberately no OPS_REPO/OPS_BRANCH here. `ops` resolves its
# task source from the binary (the Trustant fork embeds
# trustable-ai/openserverless-task). Both variables still override that default,
# so declaring them is how an image ends up on the wrong fork. See spec/start.md.
# WHY: VM setup reads these exact values too, so image and development installs
# cannot drift back to the upstream Milvus MCP or a floating branch.
ARG MILVUS_MCP_REPO=https://github.com/trustable-ai/mcp-server-milvus.git
ARG MILVUS_MCP_REF=a7e624f3057a0d739528bca3ed92504943224ceb
ARG GH_VERSION=2.105.0
ARG GH_SHA_AMD64=79e7ab5db738c48777b696e374f8b428f0647debfb424f44d2f426370407fe2f
ARG GH_SHA_ARM64=b1278c75107423d818f1937fd92cafc9e5e9b606631568f68d7fb816f5885a02

########################################################## ops-build
FROM --platform=$BUILDPLATFORM golang:1.27.4@sha256:205c7636d74e0ffbe3d007162df7d74f859f5a8c8322dfb9f907977686c126cf AS ops-build
ARG TARGETARCH
WORKDIR /src/ops
SHELL ["/bin/bash", "-o", "pipefail", "-c"]
COPY ops-prereq-error.patch /tmp/ops-prereq-error.patch
# Rebuild the fork's release snapshot with the corrected go-git dependency.
# The source hash covers the embedded task source, branch and release version.
RUN set -eu ;\
    curl -fsSL -o /tmp/ops.tar.gz \
        https://codeload.github.com/trustable-ai/openserverless-cli/tar.gz/bcb9d40abe909747c2c39531f1fbc59923745192 ;\
    echo 'df51b6b240b1f2684252988b6c5bd95d691c7c1bee2e37e8226b275391ae1644  /tmp/ops.tar.gz' | sha256sum -c - ;\
    tar -xzf /tmp/ops.tar.gz --strip-components=1 ;\
    git apply /tmp/ops-prereq-error.patch ;\
    GOTOOLCHAIN=local go get github.com/go-git/go-git/v5@v5.19.3 ;\
    printf '%s\n' \
        '31bc7cba6a248f9ce01fd682f8091a6bfb0c3730f31639e6245ab3385ac1cfba  go.mod' \
        'bd055ef89df24a25610edcb65fcf4d8868650a288cd9c7e648e044d01f914889  go.sum' | sha256sum -c - ;\
    GOTOOLCHAIN=local go mod verify ;\
    GOTOOLCHAIN=local go test -mod=readonly -vet=off -run '^TestFailedPrerequisiteKeepsInstalledVersion$' . ;\
    GOTOOLCHAIN=local CGO_ENABLED=0 GOOS=linux GOARCH="${TARGETARCH}" \
        go build -trimpath -buildvcs=false -mod=readonly -o /out/ops ./cmd/ops/

########################################################## loader
FROM ubuntu:24.04 AS loader
ARG TARGETARCH
ARG OLLAMA_VERSION

# install ollama
USER root

RUN apt-get update && apt-get install -y curl zstd

RUN curl -fsSL https://ollama.com/install.sh >ollama.sh
RUN env OLLAMA_VERSION=${OLLAMA_VERSION} bash ollama.sh

RUN \
    echo "OLLAMA_VERSION=${OLLAMA_VERSION}" >>/etc/environment

########################################################## stage1-root
FROM ubuntu:24.04 AS stage1-root
ARG TARGETARCH
ARG MILVUS_MCP_REPO
ARG MILVUS_MCP_REF
ARG GH_VERSION
ARG GH_SHA_AMD64
ARG GH_SHA_ARM64
USER root

# collecting
COPY --from=loader /etc/environment /etc/environment
COPY --from=loader /usr/local/bin/ollama /usr/local/bin/ollama
COPY --from=loader /usr/local/lib/ollama/*.so* /usr/local/lib/ollama/
# Ollama starts this runner as a subprocess when loading models.
COPY --from=loader /usr/local/lib/ollama/llama-server /usr/local/lib/ollama/

ENV DEBIAN_FRONTEND=noninteractive
RUN \
    apt update && \
    apt install -y --no-install-recommends \
        locales ca-certificates git sudo curl jq less vim lsof libatomic1 \
        tini openssh-server supervisor iputils-ping inetutils-telnet ripgrep \
        python-is-python3 python3-pip python3-pytest python3-dotenv \
        postgresql-client-16 redis-tools rclone  && \
    curl -fsSL https://deb.nodesource.com/setup_24.x | bash ;\
    apt update && apt -y install nodejs ;\
    rm -rf /var/lib/apt/lists/*

RUN \
    echo "en_US.UTF-8 UTF-8" > /etc/locale.gen && \
    locale-gen en_US.UTF-8 && \
    update-locale LANG=en_US.UTF-8 LANGUAGE=en_US.UTF-8 LC_ALL=en_US.UTF-8

# Install the official GitHub CLI from a checked release. WHY: managed account
# state is isolated at runtime, but VM and pod must execute the same verified
# client without inheriting a floating distro package.
RUN \
    case "${TARGETARCH}" in \
        amd64) GH_SHA="${GH_SHA_AMD64}" ;; \
        arm64) GH_SHA="${GH_SHA_ARM64}" ;; \
        *) echo "unsupported GitHub CLI architecture: ${TARGETARCH}" >&2; exit 1 ;; \
    esac ;\
    GH_ARCHIVE="gh_${GH_VERSION}_linux_${TARGETARCH}.tar.gz" ;\
    curl -fsSL -o "/tmp/${GH_ARCHIVE}" \
        "https://github.com/cli/cli/releases/download/v${GH_VERSION}/${GH_ARCHIVE}" ;\
    echo "${GH_SHA}  /tmp/${GH_ARCHIVE}" | sha256sum -c - ;\
    tar -C /tmp -xzf "/tmp/${GH_ARCHIVE}" ;\
    install -m 0755 "/tmp/gh_${GH_VERSION}_linux_${TARGETARCH}/bin/gh" /usr/local/bin/gh ;\
    rm -rf "/tmp/${GH_ARCHIVE}" "/tmp/gh_${GH_VERSION}_linux_${TARGETARCH}" ;\
    gh --version


# Keep the upstream Milvus CLI pinned in /usr/local/bin. WHY: app launch replaces
# ~/.local/bin/milvus_cli with a configured wrapper and must not follow a
# floating package or overwrite the package-managed implementation.
# installing lsp: ts py, mcp: postgres, redis, milvus, mongodb, cli: milvus
# WHY the `mcp<2` constraint: postgres-mcp, redis-mcp-server, and mcp-server-milvus
# declare an open (or absent) upper bound on the MCP SDK (`mcp[cli]>=1.5.0` /
# `>=1.9.4`), but all three still import `mcp.server.fastmcp`, which mcp 2.x
# renamed to `mcp.server.mcpserver`. Left unpinned they resolve mcp 2.x and die
# at import, which surfaces only as a silent "7/9 servers connected" in
# OpenCode. Keep this in step with setup.sh.
# WHY the pglast override: postgres-mcp 0.3.0 pins pglast==7.2.0, which was
# deleted from PyPI (only 7.20+ remain); 7.21 keeps the same 7.x API (issue #24).
RUN set -eu ;\
    echo 'pglast==7.21' > /tmp/uv-overrides.txt ;\
    curl -LsSf https://astral.sh/uv/install.sh | \
    env UV_INSTALL_DIR="/usr/bin" INSTALLER_NO_MODIFY_PATH=1 sh ;\
    for tool in \
        milvus-cli==1.2.1 \
        'python-lsp-server[yapf]' \
        'postgres-mcp==0.3.0|--with|mcp<2|--overrides|/tmp/uv-overrides.txt' \
        'redis-mcp-server==0.5.1|--with|mcp<2' \
        "git+${MILVUS_MCP_REPO}@${MILVUS_MCP_REF}|--with|mcp<2" ;\
    do \
        spec="${tool%%|*}" ;\
        extra="" ;\
        if [ "$tool" != "$spec" ] ; then \
            extra="$(echo "$tool" | cut -d'|' -f2- | tr '|' ' ')" ;\
        fi ;\
        env \
        UV_TOOL_BIN_DIR=/usr/local/bin \
        UV_TOOL_DIR=/opt/uv/tools \
        UV_CACHE_DIR=/opt/uv/cache \
        UV_PYTHON_PREFERENCE=only-system \
        UV_LINK_MODE=hardlink \
        uv tool install --python /usr/bin/python3 $extra "$spec" ;\
    done ;\
    # Fail the build on an import-time break rather than shipping a server that
    # only fails once a user launches an app (see the constraint note above).
    for probe in "postgres-mcp:postgres_mcp" \
                 "redis-mcp-server:src.main" \
                 "mcp-server-milvus:mcp_server_milvus" ;\
    do \
        /opt/uv/tools/"${probe%%:*}"/bin/python -c "import ${probe#*:}" \
        || { echo "FATAL: ${probe%%:*} fails at import" >&2 ; exit 1 ; } ;\
    done ;\
    rm -f /tmp/uv-overrides.txt

COPY openserverless-mcp /tmp/openserverless-mcp
COPY trustant-react-mcp /tmp/trustant-react-mcp
# The language server needs the JavaScript tsserver shipped with TypeScript 6.
RUN set -eu ;\
    npm install -g typescript-language-server@6.1.0 typescript@6.0.3 tsx@4.23.15 mongodb-mcp-server@1.9.0 ;\
    cd /tmp/openserverless-mcp && npm pack --pack-destination /tmp ;\
    npm install -g /tmp/openserverless-mcp-*.tgz ;\
    cd /tmp/trustant-react-mcp && npm pack --pack-destination /tmp ;\
    npm install -g /tmp/trustant-react-mcp-*.tgz ;\
    cd / ;\
    rm -rf /tmp/openserverless-mcp /tmp/openserverless-mcp-*.tgz /tmp/trustant-react-mcp /tmp/trustant-react-mcp-*.tgz ;\
    rm -rf /var/lib/apt/lists/* ;\
    npm cache clean --force ;\
    export VER=1.3.0 ARCH="$(dpkg --print-architecture)" ;\
    curl -sL https://github.com/txn2/mcp-s3/releases/download/v${VER}/mcp-s3_${VER}_linux_${ARCH}.tar.gz |\
    tar -C /usr/local/bin -xzvf -  mcp-s3

ENTRYPOINT ["tini", "--"]

# build the local part
########################################################## stage2-user
FROM stage1-root AS stage2-user
ARG TARGETARCH

# install cli tools
USER root

RUN \
    useradd -d /home/trustant -u 769 -s /bin/bash -m trustant ;\
    echo 'trustant ALL=(ALL:ALL) NOPASSWD: ALL' >>/etc/sudoers

ENV HOME=/home/trustant
WORKDIR /home/trustant
USER trustant

COPY --from=ops-build --chown=trustant:trustant /out/ops /home/trustant/.local/bin/ops

RUN \
    sed -e 's/^\(.*\)=\(.*\)/export \1="\2"/'  -e 's/""/"/g' >>$HOME/.bashrc /etc/environment  ;\
    echo "export PATH=\"/home/trustant/.local/bin:/home/trustant/.ops/linux-${TARGETARCH}/bin:/usr/local/bin:/usr/bin:/bin\"" >>$HOME/.bashrc

RUN bash -ic "source $HOME/.bashrc ;  ops -t ; ops -info"

# TruACP serves the coding UI on :4096 and starts pi-acp/Pi. image.sh builds the
# self-contained JavaScript bundle and pi-acp package before Docker runs and
# stages only the runtime artifacts; source and node_modules must never enter an
# image layer. The separately loaded policy file is pinned with this artifact
# set so managed Pi cannot start without the issue #57 host contract.
COPY --chown=trustant:trustant truacp-runtime/setup.sh /tmp/truacp/setup.sh
COPY --chown=trustant:trustant truacp-runtime/pi.version /tmp/truacp/pi.version
COPY --chown=trustant:trustant truacp-runtime/pi.integrity /tmp/truacp/pi.integrity
COPY --chown=trustant:trustant truacp-runtime/pi-acp-package.tgz /tmp/truacp/pi-acp-package.tgz
COPY --chown=trustant:trustant truacp-runtime/dist-bin/truacp.cjs /tmp/truacp/dist-bin/truacp.cjs
COPY --chown=trustant:trustant truacp-runtime/extensions/trustant-runtime.ts /tmp/truacp/extensions/trustant-runtime.ts
COPY --chown=trustant:trustant truacp-runtime/extensions/requirements.txt /tmp/truacp/extensions/requirements.txt
RUN \
    cd /tmp/truacp ;\
    sh setup.sh ;\
    test -x "$HOME/.local/bin/truacp" ;\
    test -x "$HOME/.local/bin/trustant-acp" ;\
    rm -rf /tmp/truacp

# workspace
#
# Drop an interactive terminal into the currently-launched app's checkout.
# launch.go's writeCurrentApp rewrites $WORKBENCH_DIR/current on every launch, so
# that file -- not the shell's starting directory -- is the source of truth for
# which app is live. screenshot.sh reads it the same way.
#
# Written as a heredoc rather than an echo of one long line: the previous
# single-line form hid three faults at once (a relative `cat` path that only
# resolved when the shell started in $HOME, an `export PWD` that sets a variable
# the next cd overwrites instead of changing directory, and an error on a fresh
# pod where no app has been launched yet).
#
# .bashrc, not .profile: the terminal spawns `$SHELL -i` (see spec/12-terminal.md),
# which reads .bashrc and never sources the profile files.
RUN \
    mkdir -p /home/trustant/workspace ;\
    cat >>$HOME/.bashrc <<'BASHRC'

# Start in the launched app's checkout, if there is one. Silent otherwise: no
# current app is the normal state on a fresh pod, and `current` can still name an
# app whose checkout has since been removed.
if [ -r "$HOME/workbench/current" ]; then
    __trustant_app="$(tr -d '[:space:]' <"$HOME/workbench/current" 2>/dev/null || true)"
    if [ -n "$__trustant_app" ] && [ -d "$HOME/workbench/$__trustant_app" ]; then
        cd "$HOME/workbench/$__trustant_app" || true
    fi
    unset __trustant_app
fi
BASHRC

########################################################## stage3-app
# start
USER root
RUN \
    if [ -x /usr/local/bin/mcp-s3 ] && [ ! -x /usr/local/bin/mcp-s3-real ]; then \
        mv /usr/local/bin/mcp-s3 /usr/local/bin/mcp-s3-real; \
    fi
COPY mcp-s3 /usr/local/bin/mcp-s3
COPY redis-mcp /usr/local/bin/trustant-redis-mcp
RUN chmod 0755 /usr/local/bin/mcp-s3 /usr/local/bin/trustant-redis-mcp
COPY bin/trustant-$TARGETARCH /usr/local/bin/trustant
COPY supervisord.ini /etc/supervisord.ini
COPY start.sh /usr/local/bin/start.sh
CMD ["/usr/local/bin/start.sh"]

COPY --chown=trustant:trustant env .env
COPY --chown=trustant:trustant trustant.json trustant.json

# Trustant's license, third-party inventory and notices (generated by deps.sh,
# staged into image/ by build.sh / hotfix.sh). See spec/20-legal.md.
COPY LICENSE NOTICE DEPS /usr/share/doc/trustant/

# start.sh / supervisord must run as root: start.sh chowns $HOME and
# supervisord (user=root) drops privilege per-program. Final user MUST be root.
USER root
WORKDIR /home/trustant
