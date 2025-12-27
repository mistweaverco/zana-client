<div align="center">

![Zana logo][logo]

# zana-client

[![Made with love][badge-made-with-love]][contributors]
[![Go][badge-golang]][golang-website]
[![Development status][badge-development-status]][development-status]
[![Our manifesto][badge-our-manifesto]][our-manifesto]
[![Latest release][badge-latest-release]][latest-release]

[Terms used](#requirements) •
[Requirements](#requirements) •
[Install](#install) •
[Usage](#usage) •
[Supported providers](#supported-providers)

<p></p>

![Zana Demo Gif](assets/demo.gif)

<p></p>

Zana 📦 aims to be like Mason.nvim 🧱,
but with the goal of supporting 🌈 not only Neovim,
but rather any other editor 🫶.

Zana is [swahili] for "tools" or "tooling."

<p></p>

Currently, Zana is in early development,
but should be considered stable enough for general use.

<p></p>

</div>

## Terms used

- *Language Server Protocol* (LSP): A protocol that defines
  how code editors and IDEs communicate with language servers.
- *Debug Adapter Protocol* (DAP): A protocol that defines
  how code editors and IDEs communicate with debuggers.
- *Package*: A package is a LSP server, DAP server, formatter
  or linter that can be installed via Zana.
- *Provider*: A provider is a package source,
    e.g., `npm`, `pypi`, `golang`, etc.
- *Package ID*: A package ID is a unique identifier for a package,
    e.g., `npm:@mistweavercokulala-ls@0.1.0`.
- *Zana Registry*: The Zana Registry is a registry of
    available packages that can be installed via Zana.
- *Terminal User Interface* (TUI): A text-based user interface
  that runs in a terminal emulator.

A minimal LSP/DAP/formatter/linter package manager
uses the [Zana Registry][zana-registry] to install and manage packages.

Easily install and manage LSP servers, DAP servers, linters, and formatters.

## Requirements

Zana is a CLI, therefore you need to have a terminal emulator available.

Besides that, we shell out a lot to install packages.

E.g. if you want to install `npm` packages,
you need to have `npm` installed.

For the packages to work in Neovim, you either need to
[zana.nvim] installed,
or source the environment setup in your shell.

```sh
source <(zana env)
```

## Install

Just head over to the [download page][download-website] or
grab it directtly from the [releases][latest-release].

## Usage

The heart of Zana is its `zana-lock.json` file.
This file is used to keep track of the installed packages and their versions.

You can tell Zana where to find the `zana-lock.json` and
the packages by setting the environment variable `ZANA_HOME`.

If `ZANA_HOME` isn't set,
Zana will look for the `zana-lock.json` file in the default locations:

- Linux: `$XDG_CONFIG_HOME/zana/zana-lock.json` or
  `$HOME/.config/zana/zana-lock.json`
- macOS: `$HOME/Library/Application Support/zana/zana-lock.json`
- Windows: `%APPDATA%\zana\zana-lock.json`

If the file doesn't exist,
Zana will create it for you (when you install a package).

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
source <(zana env zsh)
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

#### zana show

`show/info/details` shows information about one or more packages.

```sh
zana show \
  npm:@mistweavercokulala-ls@0.1.0 \
  pypi:black \
  golang:golangci-lint
```

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
it'll make sure exactly the same packages are installed
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

Zana can also update itself with:

```sh
zana update --self
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

- Linux: `$XDG_DATA_HOME/zana/packages` or `$HOME/.local/share/zana/packages`
- macOS: `$HOME/Library/Application Support/zana/packages`
- Windows: `%APPDATA%\zana\packages`

The packages are installed in the following directory structure:

```
$basepath/$provider/$package-name/
```

## Supported providers

- `cargo`
- `codeberg`
- `composer`
- `gem`
- `generic` (shell commands)
- `github`
- `gitlab`
- `golang`
- `luarocks`
- `npm`
- `nuget`
- `opam`
- `openvsx`
- `pypi`



[logo]: assets/logo.svg
[badge-made-with-love]: assets/badge-made-with-love.svg
[badge-golang]: assets/badge-golang.svg
[badge-development-status]: assets/badge-development-status.svg
[badge-our-manifesto]: assets/badge-our-manifesto.svg
[badge-latest-release]: https://img.shields.io/github/v/release/mistweaverco/zana-client?style=for-the-badge
[our-manifesto]: https://mistweaverco.com/manifesto
[development-status]: https://github.com/orgs/mistweaverco/projects/5/views/1?filterQuery=repo%3Amistweaverco%2Fzana-client.nvim
[registry-website]: https://registry.getzana.net
[golang-website]: https://golang.org
[website]: https://getzana.net
[contributors]: https://github.com/mistweaverco/zana-client/graphs/contributors
[swahili]: https://en.wikipedia.org/wiki/Swahili_language
[latest-release]: https://github.com/mistweaverco/zana-client/releases/latest
[download-website]: https://getzana.net/#download
[zana-registry]: https://github.com/mistweaverco/zana-registry
[zana.nvim]: https://github.com/mistweaverco/zana-nvim
