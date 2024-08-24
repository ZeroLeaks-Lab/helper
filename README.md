# ZeroLeaks Helper

Websocket server required for DNS and Bittorrent leak tests.

## Setup

```
$ git clone --depth=1 https://github.com/ZeroLeaks-Lab/helper
$ cd helper
$ go build
```

Copy `config.example.toml` to `config.toml` and edit as needed.

## Run

```
$ ./zeroleaks -config <CONFIG PATH>
```
