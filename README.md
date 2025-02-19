<div align="center">

![Zana Logo](assets/logo.svg)

# zana-client

[![Made with love](assets/badge-made-with-love.svg)](https://github.com/mistweaverco/zana-client/graphs/contributors)
[![Go](assets/badge-golang.svg)](https://golang.org/)
[![GitHub release (latest by date)](https://img.shields.io/github/v/release/mistweaverco/zana-client?style=for-the-badge)](https://github.com/mistweaverco/zana-client/releases/latest)
[![Discord](assets/badge-discord.svg)](https://getzana.net/discord)

[Requirements](#requirements) • [Install](#install) • [What is working?](#what-is-working)

<p></p>

![Zana Demo Gif](assets/demo.gif)

<p></p>

Zana GUI 🕹️. Zana 📦 aims to be like Mason.nvim 🧱, but maintained by the community 🌈.

Zana is swahili for "tools" or "tooling".

A minimal package manager for Neovim which
uses the [Zana Registry][zana-registry] to install and manage packages.

Easily install and manage LSP servers, DAP servers, linters, and formatters.

<p></p>

Currently, Zana is in pre-alpha and under active development.

<p></p>

</div>

## Requirements

Because Zana is a package manager for Neovim, you need to have Neovim installed.
Also, because Zana is a TUI, you need to have a terminal emulator installed.

Besides that, we shell out to `npm`, `pip`, `cargo`, `go`, and `git` to install packages,
depending on the package type.

E.g. if you want to install `pkg:npm` packages, you need to have `npm` installed.

For the packages to work in Neovim, you need to
[Zana.nvim](https://github.com/mistweaverco/zana.nvim) installed.

## Install

Just head over to the [download page][download-page] or
grab it directtly from the [releases][releases-page].

## What is working?

- [x] [registry](https://github.com/mistweaverco/zana-registry) updates on startup
- [x] Updates available for installed packages?
- [x] Filtering packages
- [x] Vim keymaps
- [x] Install `pkg:npm` packages
- [x] Update `pkg:npm` packages
- [x] Remove `pkg:npm` packages
- [ ] Install `pkg:github` packages
- [ ] Update `pkg:github` packages
- [ ] Remove `pkg:github` packages
- [x] Install `pkg:pypi` packages
- [x] Update `pkg:pypi` packages
- [x] Remove `pkg:pypi` packages
- [ ] Install `pkg:golang` packages
- [ ] Update `pkg:golang` packages
- [ ] Remove `pkg:golang` packages
- [ ] Install `pkg:cargo` packages
- [ ] Update `pkg:cargo` packages
- [ ] Remove `pkg:cargo` packages


[download-page]: https://getzana.net/#download
[releases-page]: https://github.com/mistweaverco/zana-client/releases/latest
[zana-registry]: https://github.com/mistweaverco/zana-registry

