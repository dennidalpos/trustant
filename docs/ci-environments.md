# Testing other operating systems in CI

The workflow [`.github/workflows/environments.yml`](../.github/workflows/environments.yml) checks the start scripts and the install path on systems a maintainer may not have at hand. It runs only on demand: open **Actions → Environments → Run workflow**.

| Job | Runs on | What it proves | What it does not prove |
|---|---|---|---|
| **Scripts** | Ubuntu 24.04 (amd64, arm64), macOS 15 (arm64, Intel), Windows | Every `*.sh` parses with the system bash (macOS still ships bash 3.2), `shellcheck` finds no errors, `start.ps1` parses with Windows PowerShell 5.1 and PowerShell 7 | That the scripts do the right thing when run |
| **Ubuntu install** | Clean Ubuntu 24.04, amd64 and arm64 | The real `./start.sh -n`: preflight, the Trustant package with k3s, Ollama, kubefwd, `gh`, `setup.sh`; then `./run.sh` starts and `/api/version` answers on port 8910 | The browser UI, a real model answering, publishing; a machine with other software on ports 80/443 |
| **macOS Lima** | macOS 15 (arm64, Intel) | That Lima installs, and (optional) whether a VM can boot at all | The macOS install: hosted macOS runners are virtual machines and usually cannot run another one, so the boot is reported as a skip with the reason |
| *(none)* | Windows + WSL2 | Nothing beyond the `start.ps1` parse | Hosted Windows runners cannot run WSL2 distributions |

## Inputs

- `ubuntu_install` (default on): the Ubuntu install job downloads about 5 GB and can take up to 90 minutes. Turn it off for a quick script check.
- `macos_lima` (default off): attempt the Lima VM boot. Leave it off unless you want to see whether the runner image gained nested virtualization.

## Reading the results

- A red **Scripts** job is a real defect in a script on that OS. Fix it before anything else.
- A red **Ubuntu install** job: open the job log, the last 120 lines of `start.log` and `run.log` are printed at the end.
- **macOS Lima** green with a notice "cannot boot" means nothing was proven about macOS; test `start.sh` on a real Mac.

## Limits to know

- First run (2026-11-25, default inputs): all nine jobs passed. The two `macos-15` jobs were cancelled on the first attempt, without an error in the logs, and passed on a re-run; if a macOS job is cancelled, re-run it before suspecting the scripts. The Lima VM boot was not attempted (`macos_lima` off); on `macos-15` (arm64) `kern.hv_support` is not reported, on `macos-15-intel` it is 1, so only a run with `macos_lima` on shows whether a VM can boot.
- Decision: no Docker job is added to this workflow, so it never triggers the tag and `opsroot.json` effects of `build.sh`.
- Not covered by any job yet: building the container image (`build.sh` changes local tags and `oplugins-truinst/opsroot.json`, so it is not run here), the full macOS install, and the Windows + WSL2 install.
- The full macOS and Windows installs cannot be automated on hosted runners. A self-hosted Mac or Windows machine would be needed; the jobs are written so another `runs-on` can be added.
- `ubuntu-24.04-arm` and `macos-15-intel` depend on the runner labels GitHub offers to the repository (arm64 runners are free only for public repositories).
