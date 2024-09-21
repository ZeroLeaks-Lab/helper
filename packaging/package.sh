#!/bin/sh

set -eu

build_deb() {
	cd deb
	install -Dm0644 "$SRC"/config.example.toml etc/zeroleaks/config.example.toml
	install -sDm0755 "$SRC"/zeroleaks usr/bin/zeroleaks
	dpkg-deb --build --root-owner-group . "$DST"/zeroleaks-x86_64.deb
}

DST="$(pwd)"
cd "$(dirname "$0")"
SRC="$(realpath ..)"
build_deb
