package main

import (
	"flag"
	"log"
	"net"
	"time"
	"zeroleaks/bittorrent"
	"zeroleaks/dns"

	"github.com/BurntSushi/toml"
	"github.com/coder/websocket"
)

type TLSConfig struct {
	Cert string
	Key  string
}

type Config struct {
	Websocket struct {
		Addr    string
		TLS     TLSConfig
		Origins []string
	}
	DNS struct {
		Addr    string
		Domain  string
		Timeout time.Duration
	}
	BitTorrent struct {
		Addr    string
		Timeout time.Duration
	}
}

type IPLogger[T any] interface {
	RegisterCallback(t T, f func(net.IP))
}

var conf Config

var dnsServer IPLogger[uint32]
var bittorrentTracker IPLogger[bittorrent.InfoHash]

func main() {
	configPath := flag.String("config", "config.toml", "Configuration file path. Defaults to \"config.toml\"")
	flag.Parse()
	if _, err := toml.DecodeFile(*configPath, &conf); err != nil {
		log.Fatalln("Failed to parse config file:", err)
	}
	websocketOptions := websocket.AcceptOptions{}
	if len(conf.Websocket.Origins) == 0 {
		websocketOptions.InsecureSkipVerify = true
	} else {
		websocketOptions.OriginPatterns = conf.Websocket.Origins
	}

	d := dns.NewServer(conf.DNS.Domain, conf.DNS.Timeout)
	dnsServer = d
	go d.Start(conf.DNS.Addr)
	t, err := bittorrent.NewTracker(conf.BitTorrent.Addr, conf.BitTorrent.Timeout)
	if err != nil {
		log.Fatalln("Failed to start BitTorrent tracker:", err)
	}
	go t.Start()
	bittorrentTracker = t
	startWebsocketServer(conf.Websocket.Addr, conf.Websocket.TLS, websocketOptions)
}
