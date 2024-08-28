package main

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/coder/websocket"
)

const LOG_TAG = "Websocket server:"
const TESTS_NUMBER = 6

type DnsLeakTestParams struct {
	Base       string   `json:"base"`
	Subdomains []string `json:"subdomains"`
}

func dnsLeakTest(ws *websocket.Conn) {
	random := make([]byte, 4*TESTS_NUMBER)
	if _, err := rand.Read(random); err != nil {
		log.Panicln(LOG_TAG, "failed to read random bytes:", err)
	}
	ctx := context.Background()
	ws.CloseRead(ctx)
	params := DnsLeakTestParams{
		Base:       conf.DNS.Domain,
		Subdomains: make([]string, 0, TESTS_NUMBER),
	}
	ipSet := make(map[string]struct{})
	closed := false
	for i := 0; i < TESTS_NUMBER; i++ {
		s := binary.LittleEndian.Uint32(random[i : i+4])
		params.Subdomains = append(params.Subdomains, strconv.FormatUint(uint64(s), 10))
		registerDnsCallback(s, func(ip net.IP) {
			if !closed {
				ipStr := ip.String()
				if _, ok := ipSet[ipStr]; !ok {
					ipSet[ipStr] = struct{}{}
					if err := ws.Write(ctx, websocket.MessageText, []byte(ip.String())); err != nil {
						log.Println(LOG_TAG, "failed to send IP:", err.Error())
					}
				}
			}
		})
	}
	b, err := json.Marshal(params)
	if err != nil {
		log.Panicln("Cannot serialize DnsLeakTestParams:", err)
	}
	if err := ws.Write(ctx, websocket.MessageText, b); err != nil {
		log.Println(LOG_TAG, "failed to send subdomain:", err.Error())
	}
	go func() {
		time.Sleep(timeout)
		ws.Close(websocket.StatusNormalClosure, "")
		closed = true
	}()
}

func startWebsocketServer(addr string, tls TLSConfig, options websocket.AcceptOptions) {
	http.HandleFunc("/v1/dns", func(w http.ResponseWriter, r *http.Request) {
		ws, err := websocket.Accept(w, r, &options)
		if err != nil {
			log.Println(LOG_TAG, "failed to accept:", err.Error())
			return
		}
		dnsLeakTest(ws)
	})
	var err error
	if conf.Websocket.TLS.Cert == "" && conf.Websocket.TLS.Key == "" {
		err = http.ListenAndServe(addr, nil)
	} else {
		err = http.ListenAndServeTLS(addr, tls.Cert, tls.Key, nil)
	}
	log.Fatalln(LOG_TAG, "failed to start:", err)
}
