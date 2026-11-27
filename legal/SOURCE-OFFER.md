# Written offer of corresponding source (container image)

The Trustant container image (`ghcr.io/trustant/trustant`) bundles components
licensed under the GPL, LGPL and similar terms. Trustant's own code is
AGPL-3.0-or-later and its source is this repository. For the components below,
the corresponding source code is available as follows.

## Ubuntu 24.04 packages (dpkg)

The image includes Ubuntu packages (coreutils, bash, git, gnupg, curl and so
on) under the GPL, LGPL and MPL. Their source is published in the Ubuntu
archive. In the image, query the source package and source version for an
installed binary package `<pkg>`:

    dpkg-query -W -f='${source:Package}=${source:Version}\n' '<pkg>'

On a system with Ubuntu 24.04 (noble) source repositories enabled, pass the
result to `apt-get source --only-source '<source-package>=<source-version>'`.
The [Ubuntu archive](https://archive.ubuntu.com/ubuntu/pool/) and
[Launchpad](https://launchpad.net/ubuntu/+source) also list source packages.

## Python components under GPL/LGPL

| Component | Version | License | Source |
|---|---|---|---|
| pglast (with libpg_query) | 7.21 | GPL-3.0-or-later | [PyPI source distribution](https://pypi.org/project/pglast/7.21/#files) |
| psycopg | 3.3.9 | LGPL-3.0-only | [PyPI source distribution](https://pypi.org/project/psycopg/3.3.9/#files) |
| psycopg-binary | 3.3.9 | LGPL-3.0-only | [Tagged source](https://github.com/psycopg/psycopg/tree/3.3.9), [wheel build workflow](https://github.com/psycopg/psycopg/blob/3.3.9/.github/workflows/packages-bin.yml), and [bundled libpq build script](https://github.com/psycopg/psycopg/blob/3.3.9/tools/ci/build_libpq.sh) |
| psycopg-pool | 3.3.3 | LGPL-3.0-only | [PyPI source distribution](https://pypi.org/project/psycopg-pool/3.3.3/#files) |
| docstring-to-markdown | 0.17 | LGPL-2.1-or-later | [PyPI source distribution](https://pypi.org/project/docstring-to-markdown/0.17/#files) |

For releases that publish a source distribution, use
`pip download --no-binary :all: --no-deps <name>==<version>`.
The `psycopg-binary` 3.3.9 release on PyPI contains wheels only.

## Written offer

For at least three years from the date of each image release, and for as long
as the image is offered, anyone who received the image may request the
corresponding source of the components above, if the download locations here
are no longer available. Open an issue at
<https://github.com/trustant/trustant/issues> naming the image tag and the
component. The source is provided at no more than the cost of physically
performing the distribution.

This is a technical notice, not legal advice.
