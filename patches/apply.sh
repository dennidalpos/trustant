#!/usr/bin/env bash
# Apply the patches in this directory to the oplugins-truinst checkout.
#
# oplugins-truinst is a release repository that this fork never pushes to, so
# changes to it travel as patches kept here. Each patch is named
# <submodule>-<topic>.patch and is applied from inside that submodule.
# The script is idempotent: a patch that is already applied is skipped.
# It never commits.
set -euo pipefail

cd "$(dirname "$0")/.."

status=0
for patch in patches/oplugins-truinst-*.patch; do
    [ -e "$patch" ] || continue
    abs="$PWD/$patch"
    if git -C oplugins-truinst apply --reverse --check "$abs" >/dev/null 2>&1; then
        echo "patch already applied: $patch"
    elif git -C oplugins-truinst apply --check "$abs" >/dev/null 2>&1; then
        git -C oplugins-truinst apply "$abs"
        echo "patch applied: $patch"
    else
        echo "patch does not apply: $patch (oplugins-truinst changed?)" >&2
        status=1
    fi
done
exit "$status"
