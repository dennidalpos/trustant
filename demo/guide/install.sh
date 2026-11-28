#!/bin/bash
# Copyright 2025-2026 Nuvolaris Inc
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU Affero General Public License as published
# by the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.
#
# This program is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU Affero General Public License for more details.
#
# You should have received a copy of the GNU Affero General Public License
# along with this program.  If not, see <https://www.gnu.org/licenses/>.

# install.sh — generate the creation-mode guides and copy them into the Demo workbenches.
#
#   ./install.sh                      # all guide apps, workbench in $HOME/trustant-state/workbench
#   WORKBENCH=/path ./install.sh      # another workbench root
#
# Static apps (guidastarter, provachecklist, guidashared, guidapublish) get the kit in web/.
# React apps (guidaprovider, guidatruchat, guidatemplate) get it in public/, which vite copies to web/.
# Run `ops ide login` then `ops ide deploy` in each workbench afterwards (see README.md).
set -e
KIT="$(cd "$(dirname "$0")" && pwd)"
WB="${WORKBENCH:-$HOME/trustant-state/workbench}"
GEN="$(mktemp -d)"
trap 'rm -rf "$GEN"' EXIT
python3 "$KIT/build_guides.py" "$GEN"

for app in guidastarter provachecklist guidashared guidapublish; do
  [ -d "$WB/$app/web" ] || { echo "skip $app: no workbench"; continue; }
  cp "$KIT/guide-kit.css" "$KIT/guide-kit.js" "$WB/$app/web/"
  # Keep the starter's own technical page once, as riferimento.html.
  if [ "$app" = guidastarter ] && [ -f "$WB/$app/web/guida.html" ] && [ ! -f "$WB/$app/web/riferimento.html" ]; then
    mv "$WB/$app/web/guida.html" "$WB/$app/web/riferimento.html"
  fi
  cp "$GEN/$app/guida.html" "$WB/$app/web/guida.html"
done
# Landing page = the guide for apps whose home was only a stub.
for app in guidastarter guidashared guidapublish; do
  [ -d "$WB/$app/web" ] && cp "$GEN/$app/guida.html" "$WB/$app/web/index.html"
done
for app in guidaprovider guidatruchat guidatemplate; do
  [ -d "$WB/$app" ] || { echo "skip $app: no workbench"; continue; }
  mkdir -p "$WB/$app/public"
  cp "$KIT/guide-kit.css" "$KIT/guide-kit.js" "$WB/$app/public/"
  cp "$GEN/$app/guida.html" "$WB/$app/public/guida.html"
  if [ "$app" = guidaprovider ]; then
    cp "$GEN/$app/trureact.html" "$WB/$app/public/"
  fi
done
echo "installed into $WB"
