package main

import (
	"flag"
	"log"
	"time"
	"zeroleaks/dns"

	"github.com/BurntSushi/toml"
	"github.com/coder/websocket"
)

type TLSConfig struct {
	Cert string
	Key  string
}

type Config struct {
	Timeout   uint32
	Websocket struct {
		Addr    string
		TLS     TLSConfig
		Origins []string
	}
	DNS struct {
		Addr   string
		Domain string
	}
}

var conf Config
var timeout time.Duration
var dnsServer dns.DnsServer

func main() {
	configPath := flag.String("config", "config.toml", "Configuration file path. Defaults to \"config.toml\"")
	flag.Parse()
	if _, err := toml.DecodeFile(*configPath, &conf); err != nil {
		log.Fatalln("Failed to parse config file:", err)
	}
	timeout = time.Duration(conf.Timeout) * time.Second
	websocketOptions := websocket.AcceptOptions{}
	if len(conf.Websocket.Origins) == 0 {
		websocketOptions.InsecureSkipVerify = true
	} else {
		websocketOptions.OriginPatterns = conf.Websocket.Origins
	}

	d := dns.NewServer(conf.DNS.Domain, timeout)
	dnsServer = d
	go d.Start(conf.DNS.Addr)
	startWebsocketServer(conf.Websocket.Addr, conf.Websocket.TLS, websocketOptions)
}
