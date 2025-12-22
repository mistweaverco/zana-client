<div align="center">

![Zana Logo](assets/logo.svg)

# zana-client

[![Made with love](assets/badge-made-with-love.svg)](https://github.com/mistweaverco/zana-client/graphs/contributors)
[![Go](assets/badge-golang.svg)](https://golang.org/)
[![GitHub release (latest by date)](https://img.shields.io/github/v/release/mistweaverco/zana-client?style=for-the-badge)](https://github.com/mistweaverco/zana-client/releases/latest)
[![Discord](assets/badge-discord.svg)](https://getzana.net/discord)

[Requirements](#requirements) • [Install](#install) • [Usage](#usage) • [What's working?](#whats-working)

<p></p>

![Zana Demo Gif](assets/demo.gif)

<p></p>

Zana 📦 aims to be like Mason.nvim 🧱,
but with the goal of supporting 🌈 not only Neovim,
but rather any other editor 🫶.

Zana is swahili for "tools" or "tooling".

A minimal package manager for Neovim (and other editors) which
uses the [Zana Registry][zana-registry] to install and manage packages.

Easily install and manage LSP servers, DAP servers, linters, and formatters.

<p></p>

Currently, Zana is in beta and under active development.

<p></p>

</div>

## Requirements

Zana is a TUI, therefore you need to have a terminal emulator available.

Besides that, we shell out to `npm`, `pip`, `cargo`, `go`, and `git` to install packages,
depending on the package type.

E.g. if you want to install `pkg:npm` packages, you need to have `npm` installed.

For the packages to work in Neovim, you need to
[Zana.nvim](https://github.com/mistweaverco/zana.nvim) installed.

## Install

Just head over to the [download page][download-page] or
grab it directtly from the [releases][releases-page].

## Usage

The heart of Zana is its `zana-lock.json` file.
This file is used to keep track of the installed packages and their versions.

You can tell Zana where to find the `zana-lock.json` and
the packages by setting the environment variable `ZANA_HOME`.

If `ZANA_HOME` is not set,
Zana will look for the `zana-lock.json` file in the default locations:

- Linux: `$XDG_CONFIG_HOME/zana/zana-lock.json` or `$HOME/.config/zana/zana-lock.json`
- macOS: `$HOME/Library/Application Support/zana/zana-lock.json`
- Windows: `%APPDATA%\zana\zana-lock.json`

If the file does not exist,
Zana will create it for you (when you install a package).

1. Start Zana by running `zana` in your terminal.
2. Use the arrow keys (or hjkl) to navigate the packages.
3. Use `enter` to install a package.
4. Use `enter` to update a package.
5. Use `backspace` to remove a package.
6. Use `/` to filter packages.
7. Use `q` to quit Zana.

It's advised to keep the `zana-lock.json` file in version control.

### Modify environment path

If you want the installed packages to be available in your path,
you can add the following to your shell configuration file:

#### bash environment setup

add to `~/.bashrc`:

```sh
source <(zana env)
```

#### zsh environment setup

add to `~/.zshrc`:

```sh
source <(zana env)
```

or with [evalcache](https://github.com/mroth/evalcache) for zsh,
add to `~/.zshrc`:

```sh
_evalcache zana env zsh
```

#### PowerShell environment setup

add to `profile`:

```sh
zana env powershell | Invoke-Expression
```

### CLI autocompletion

If you want autocompletion for the CLI commands,
you can add the following to your shell configuration file:

#### bash autocompletion setup

add to `~/.bashrc`:

```sh
source <(zana completion bash)
```

#### zsh autocompletion setup

add to `~/.zshrc`:

```sh
source <(zana completion zsh)
```

or with [evalcache](https://github.com/mroth/evalcache) for zsh,
add to `~/.zshrc`:

```sh
_evalcache zana completion zsh
```

#### fish autocompletion setup

add to `~/.config/fish/completions/zana.fish`:

```sh
zana completion fish > ~/.config/fish/completions/zana.fish
```

#### powershell autocompletion setup

add to `profile`:

```sh
zana completion powershell | Invoke-Expression
```

### CLI Options

You can run `zana --help` to see the available CLI options.

#### zana install

`install`/`add` install packages

```sh
zana install \
  npm:@mistweavercokulala-ls@0.1.0 \
  pypi:black \
  golang:golangci-lint
```

#### zana sync

`sync` syncs the installed packages or registry data.

For packages,
it will make sure exactly the same packages are installed
that are listed in the `zana-lock.json` file.

```sh
zana sync packages
```

For registry data,
it'll update the local registry cache
with the latest data from the Zana Registry.

```sh
zana sync registry
```

The registry data is cached locally,
but with the `sync registry` command you can force an update.

#### zana list

`list`/`ls` list all installed packages.

```sh
zana list
```

or with `--all`/`-A` flag all available packages.

```sh
zana list --all
```

You can also filter packages by
prefix of either the package id or name.

```sh
 # lists all available packages with "yaml" in the name
zana list -A yaml
```

#### zana update

`update`/`up` updates packages.

```sh
zana update \
  npm:@mistweavercokulala-ls \
  pypi:black@latest
```

You can also update all packages at once with the `--all`/`-A` flag.

```sh
zana update --all
```

or filter packages by
prefix of either the package id or name.

```sh
 # updates all installed packages with "yaml" in the name
zana update -A yaml
```

#### zana remove

`remove`/`rm` removes packages.

```sh
zana remove \
  npm:@mistweavercokulala-ls \
  pypi:black
```

or filter packages by
prefix of either the package id or name.

```sh
 # removes all installed packages with "yaml" in the name
zana remove -A yaml
```

#### zana health

- `health` checks for requirements
(for shelling out to install packages)

```sh
zana health
```

### Where are the packages?

Zana uses a basepath to install packages of different types.

If you have set the `ZANA_HOME` environment variable,
the basepath will be `$ZANA_HOME/packages`.

If `ZANA_HOME` isn't set, the basepath is:

- Linux: `$XDG_CONFIG_HOME/zana/packages` or `$HOME/.config/zana/packages`
- macOS: `$HOME/Library/Application Support/zana/packages`
- Windows: `%APPDATA%\zana\packages`

The packages are installed in the following directory structure:

```
$basepath/$provider/$package-name/
```


## Supported providers

- Providers
  - [x] `cargo`
  - [x] `codeberg`
  - [x] `composer`
  - [x] `gem`
  - [x] `generic` (shell commands)
  - [x] `github`
  - [x] `gitlab`
  - [x] `golang`
  - [x] `luarocks`
  - [x] `npm`
  - [x] `nuget`
  - [x] `opam`
  - [x] `openvsx`
  - [x] `pypi`



[download-page]: https://getzana.net/#download
[releases-page]: https://github.com/mistweaverco/zana-client/releases/latest
[zana-registry]: https://github.com/mistweaverco/zana-registry

