package main

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"
	"zeroleaks/bittorrent"
	"zeroleaks/utils"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

const WS_LOG_TAG = "Websocket server:"
const DNS_LEAK_TESTS_NUMBER = 6

type dnsLeakTestParams struct {
	Base       string   `json:"base"`
	Subdomains []string `json:"subdomains"`
}

type IPSender struct {
	ws       *websocket.Conn
	ctx      context.Context
	timeout  time.Duration
	ch       chan net.IP
	Callback func(net.IP)
}

func NewIPSender(ws *websocket.Conn, ctx context.Context, timeout time.Duration) *IPSender {
	ch := make(chan net.IP, 32)
	return &IPSender{
		ws:      ws,
		ctx:     ctx,
		timeout: timeout,
		ch:      ch,
		Callback: func(ip net.IP) {
			select {
			case ch <- ip:
			default:
				// websocket connection closed
			}
		},
	}
}

func (s *IPSender) Start() {
	go func() {
		time.Sleep(s.timeout)
		s.ch <- nil
	}()
	ipSet := make(map[string]struct{})
	for {
		ip := <-s.ch
		if ip == nil { // timeout
			s.ws.Close(websocket.StatusNormalClosure, "")
			break
		} else {
			ipStr := ip.String()
			if _, ok := ipSet[ipStr]; !ok {
				ipSet[ipStr] = struct{}{}
				if err := s.ws.Write(s.ctx, websocket.MessageText, []byte(ipStr)); err != nil {
					log.Println(WS_LOG_TAG, "failed to send IP:", err.Error())
				}
			}
		}
	}
}

func dnsLeakTest(ws *websocket.Conn) {
	random := utils.RandomBytes(4 * DNS_LEAK_TESTS_NUMBER)
	ctx := context.Background()
	ws.CloseRead(ctx)
	params := dnsLeakTestParams{
		Base:       conf.DNS.Domain,
		Subdomains: make([]string, 0, DNS_LEAK_TESTS_NUMBER),
	}
	ipSender := NewIPSender(ws, ctx, conf.DNS.Timeout)
	for i := 0; i < DNS_LEAK_TESTS_NUMBER; i++ {
		s := binary.LittleEndian.Uint32(random[i : i+4])
		params.Subdomains = append(params.Subdomains, strconv.FormatUint(uint64(s), 10))
		dnsServer.RegisterCallback(s, ipSender.Callback)
	}
	if err := wsjson.Write(ctx, ws, params); err != nil {
		log.Println(WS_LOG_TAG, "failed to send DNS params:", err.Error())
		ws.CloseNow()
		return
	}
	go ipSender.Start()
}

func bittorrentLeakTest(ws *websocket.Conn) {
	infoHash := bittorrent.InfoHash(utils.RandomBytes(20))
	ctx := context.Background()
	ws.CloseRead(ctx)
	ipSender := NewIPSender(ws, ctx, conf.BitTorrent.Timeout)
	bittorrentTracker.RegisterCallback(infoHash, ipSender.Callback)
	magnetLink := "magnet:?xt=urn:btih:" + hex.EncodeToString(infoHash[:]) + "&tr=udp://" + conf.Host + ":" + strconv.FormatInt(int64(bittorrentTrackerPort), 10)
	if err := ws.Write(ctx, websocket.MessageText, []byte(magnetLink)); err != nil {
		log.Println(WS_LOG_TAG, "failed to send magnet link:", err)
		ws.CloseNow()
		return
	}
	go ipSender.Start()
}

func startWebsocketServer(addr string, tls TLSConfig, options websocket.AcceptOptions) {
	acceptWebsocket := func(callback func(*websocket.Conn)) func(http.ResponseWriter, *http.Request) {
		return func(w http.ResponseWriter, r *http.Request) {
			ws, err := websocket.Accept(w, r, &options)
			if err != nil {
				log.Println(WS_LOG_TAG, "failed to accept:", err.Error())
				return
			}
			callback(ws)
		}
	}
	http.HandleFunc("/v1/dns", acceptWebsocket(dnsLeakTest))
	http.HandleFunc("/v1/bittorrent", acceptWebsocket(bittorrentLeakTest))

	var err error
	if tls.Cert == "" && tls.Key == "" {
		err = http.ListenAndServe(addr, nil)
	} else {
		err = http.ListenAndServeTLS(addr, tls.Cert, tls.Key, nil)
	}
	log.Fatalln(WS_LOG_TAG, "failed to start:", err)
}
