# git-ssh (ssh-profile)

[![CI](https://github.com/kuyacarlo/ssh-profile/actions/workflows/ci.yml/badge.svg)](https://github.com/kuyacarlo/ssh-profile/actions/workflows/ci.yml)
[![MIT License](https://img.shields.io/badge/license-MIT-blue.svg)](./LICENSE)

Per-repo SSH identity profiles for GitHub **without** `~/.ssh/config` Host aliases (Orca-safe). The CLI binary is **`git-ssh`**.

Companion to **[git-profile](https://github.com/kuyacarlo/git-profile)**: use the same profile name for Git identity and SSH key.

**Install both tools:** [git-profile docs/INSTALL.md](https://github.com/kuyacarlo/git-profile/blob/master/docs/INSTALL.md)

## Install

### Install script

```bash
curl -fsSL https://raw.githubusercontent.com/kuyacarlo/ssh-profile/main/install.sh | sh
```

### Fedora Copr

```bash
sudo dnf copr enable kuya-carlo/git-ssh
sudo dnf install git-ssh
```

### Prebuilt binaries

Download from [Releases](https://github.com/kuyacarlo/ssh-profile/releases) and place `git-ssh` on `$PATH`.

### Build from source

```bash
git clone https://github.com/kuyacarlo/ssh-profile.git
cd ssh-profile
make build
```

Confirm:

```bash
git-ssh version
```

## Quick start

```bash
git-ssh add alice
# add the printed public key to GitHub → SSH keys

git-ssh clone alice private-repo
# or, in an existing repo:
git-ssh use alice

# pair with git-profile under the same name
git-profile use alice
```

Layout:

- `~/.config/git-ssh/config.json` — profile store
- `~/.ssh/git-ssh/<profile>/id_ed25519` — managed keys

Markers in a repo: `current-profile.ssh` (git-ssh) pairs with `current-profile.name` (git-profile).

## Usage

| Command | Purpose |
| --- | --- |
| `git-ssh add [profile]` | Add or update a profile (auto key + `github_user`) |
| `git-ssh list` | List profiles |
| `git-ssh show [profile]` | Show a profile |
| `git-ssh del [profile]` | Delete a profile |
| `git-ssh use [profile] [repo\|owner/repo]` | Apply key + origin in this repo |
| `git-ssh clone [profile] [repo\|…] [dir]` | Clone with the profile key |
| `git-ssh unuse` | Clear git-ssh settings from this repo |
| `git-ssh current` | Show active profile (`default` if none) |
| `git-ssh export` / `import` | Profile JSON |
| `git-ssh backup` / `restore` | Sidecar backup |
| `git-ssh completion …` | Shell completion |
| `git-ssh version` | Print version |

`add` defaults: `github_user` = profile name; identity = managed ed25519 under `~/.ssh/git-ssh/<profile>/`. Override with `--github-user`, `--remote-host`, `--identity`, or `--set Key=Value`.

See `git-ssh --help` and each subcommand’s `--help` for flags and examples.

## License

[MIT](./LICENSE)
