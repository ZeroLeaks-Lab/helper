package main

import (
	"flag"
	"log"

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
		Addr   string
		Domain string
	}
}

var conf Config

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

	go startDnsServer(conf.DNS.Addr, conf.DNS.Domain)
	startWebsocketServer(conf.Websocket.Addr, conf.Websocket.TLS, websocketOptions)
}
