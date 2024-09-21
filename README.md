# ZeroLeaks Helper

Websocket server required for DNS and BitTorrent leak tests.

## Installation guide

Download `zeroleaks-x86_64.deb` from the releases page, and install it:

```
$ wget https://github.com/ZeroLeaks-Lab/helper/releases/latest/download/zeroleaks-x86_64.deb
$ sudo dpkg -i zeroleaks-x86_64.deb
```

### Configuration

Rename `/etc/zeroleaks/config.example.toml` to `config.toml` and edit as needed.

`addr` parameters can be specified as `IPv4:PORT`, `[IPv6]:PORT`, or just `:PORT` to listen on all addresses.

The `DNS.domain` field must be set to a domain containing a NS record pointing to the ZeroLeaks helper host. This can be a subdomain of it.

For example, if the zeroleaks helper is hosted at zeroleaks.org, `DNS.domain` can be set to dns.zeroleaks.org, and you would add a record like:

```
dns.zeroleaks.org.  3600    IN  NS  zeroleaks.org.
```

If you want the websocket server to handle TLS by itself, just specify the paths to your TLS certificate and key in `Websocket.TLS`, and you're good to go.

If instead you want to run the websocket server behind a TLS reverse proxy, remove the `Websocket.TLS` fields and configure your reverse proxy to forward plain HTTP to it. Here is an example nginx configuration snippet to expose the websocket server under the `/helper` path:

```nginx
location /helper {
  rewrite ^/helper(.*)$ $1 break;
  proxy_pass http://127.0.0.1:8000;
  proxy_http_version 1.1;
  proxy_set_header Upgrade $http_upgrade;
  proxy_set_header Connection "upgrade";
}
```

Then `/helper` needs to be added at the end of the `HELPER_SERVER_URL` field from the zeroleaks-web's `Config.php`.

### Run

```
$ sudo systemctl start zeroleaks
```

## Build from source

Instead of downloading the `.deb` package, you can also build the binary from source by yourself with:

```
$ git clone --depth=1 https://github.com/ZeroLeaks-Lab/helper.git
$ cd helper
$ go build
```

Then, to build the `.deb` package:

```
$ ./packaging/package.sh
```
