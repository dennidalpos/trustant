# Written offer of corresponding source (container image)

The Trustant container image (`ghcr.io/trustant/trustant`) bundles components
licensed under the GPL, LGPL and similar terms. Trustant's own code is
AGPL-3.0-or-later and its source is this repository. For the components below,
the corresponding source code is available as follows.

## Ubuntu 24.04 packages (dpkg)

The image includes Ubuntu packages (coreutils, bash, git, gnupg, curl and so
on) under the GPL, LGPL and MPL. Their source is published in the Ubuntu
archive. For a package `<pkg>` at the version recorded in
[IMAGE-v1.0.3.md](IMAGE-v1.0.3.md) or in `/var/lib/dpkg/status` inside the
image, obtain it with:

    apt-get source <pkg>=<version>

with a `deb-src` entry for Ubuntu 24.04 (noble) enabled, or from
<https://archive.ubuntu.com/ubuntu/pool/> and <https://launchpad.net>.

## Python components under GPL/LGPL

| Component | Version | License | Source |
|---|---|---|---|
| pglast (with libpg_query) | 7.21 | GPL-3.0-or-later | <https://github.com/lelit/pglast>, sdist `pglast==7.21` on PyPI |
| psycopg | as installed | LGPL-3.0-only | <https://github.com/psycopg/psycopg>, sdist on PyPI |
| psycopg-pool | 3.3.3 | LGPL-3.0-only | <https://github.com/psycopg/psycopg>, sdist `psycopg-pool==3.3.3` |
| docstring-to-markdown | 0.17 | LGPL-2.1-or-later | <https://github.com/python-lsp/docstring-to-markdown>, sdist on PyPI |

The sdist of any Python distribution is available with
`pip download --no-binary :all: --no-deps <name>==<version>`.

## Written offer

For at least three years from the date of each image release, and for as long
as the image is offered, anyone who received the image may request the
corresponding source of the components above, if the download locations here
are no longer available. Open an issue at
<https://github.com/trustant/trustant/issues> naming the image tag and the
component. The source is provided at no more than the cost of physically
performing the distribution.

This is a technical notice, not legal advice. Open points: nine apt packages
without a copyright file and the exact versions of the GPL/LGPL Python
components in the image.
