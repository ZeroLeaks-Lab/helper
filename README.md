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

If you want to run the websocket server behind a TLS reverse proxy, remove the `Websocket.TLS` fields and configure your reverse proxy to forward plain HTTP to it. Here is an example nginx configuration snippet to expose the websocket server under the `/helper` path:

```nginx
location /helper {
  rewrite ^/helper(.*)$ $1 break;
  proxy_pass http://127.0.0.1:8000;
  proxy_http_version 1.1;
  proxy_set_header Upgrade $http_upgrade;
  proxy_set_header Connection "upgrade";
}
```

Then `/helper` needs to be added at the end of the `HELPER_SERVER_URL` field from the web `Config.php`.

## Run

```
$ ./zeroleaks -config <CONFIG PATH>
```
