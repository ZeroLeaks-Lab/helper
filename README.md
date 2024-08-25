# ZeroLeaks Helper

Websocket server required for DNS and Bittorrent leak tests.

## Build

```
$ git clone --depth=1 https://github.com/ZeroLeaks-Lab/helper
$ cd helper
$ go build
```


## Setup

Copy `config.example.toml` to `config.toml` and edit as needed.

The `DNS.domain` field must be set to a domain containing a NS record pointing to the zeroleaks helper host. This can be a subdomain of it.

For example, if the zeroleaks helper is hosted at zeroleaks.org, `DNS.domain` can be set to dns.zeroleaks.org, and you would add a record like:

```
dns.zeroleaks.org.  3600    IN  NS  zeroleaks.org.
```

## Run

```
$ ./zeroleaks -config <CONFIG PATH>
```
