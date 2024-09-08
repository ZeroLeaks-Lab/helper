package main

import (
	"context"
	"math/rand"
	"net"
	"os"
	"strconv"
	"testing"
	"time"
	"zeroleaks/bittorrent"
	"zeroleaks/utils"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

const addr = "127.0.0.1:38080"
const timeout = 100 * time.Millisecond

type MockLogger[T comparable] struct {
	callbacks map[T]func(net.IP)
}

func (l *MockLogger[T]) RegisterCallback(k T, f func(net.IP)) {
	l.callbacks[k] = f
}

type WebsocketClient struct {
	ctx context.Context
	ws  *websocket.Conn
}

func TestMain(m *testing.M) {
	go startWebsocketServer(addr, TLSConfig{}, websocket.AcceptOptions{})
	time.Sleep(10 * time.Millisecond) // let the websocket server start
	os.Exit(m.Run())
}

func wsConnect(endpoint string, t *testing.T) WebsocketClient {
	ctx := context.Background()
	ws, _, err := websocket.Dial(ctx, "ws://"+addr+"/v1/"+endpoint, nil)
	if err != nil {
		t.Fatalf("Cannot establish websocket connection: %s", err)
	}
	return WebsocketClient{ctx: ctx, ws: ws}
}

func (w *WebsocketClient) readJson(v interface{}, t *testing.T) {
	if err := wsjson.Read(w.ctx, w.ws, v); err != nil {
		t.Fatalf("Failed to read json: %s", err)
	}
}

func (w *WebsocketClient) readString(t *testing.T) string {
	msgType, msg, err := w.ws.Read(w.ctx)
	if err != nil {
		t.Fatalf("Failed to read text message: %s", err)
	}
	if msgType != websocket.MessageText {
		t.Errorf("Invalid message type: got %s, expected %s", msgType, websocket.MessageText)
	}
	return string(msg)
}

func (w *WebsocketClient) readBinary(t *testing.T) []byte {
	msgType, msg, err := w.ws.Read(w.ctx)
	if err != nil {
		t.Fatalf("Failed to read binary message: %s", err)
	}
	if msgType != websocket.MessageBinary {
		t.Errorf("Invalid message type: got %s, expected %s", msgType, websocket.MessageBinary)
	}
	return msg
}

func (w *WebsocketClient) readAssertEqualsIP(ip net.IP, t *testing.T) {
	msg := w.readString(t)
	if !net.ParseIP(msg).Equal(ip) {
		t.Errorf("Invalid IP received. Got %s, expected %s", msg, ip)
	}
}

func (w *WebsocketClient) assertEnd(timeout time.Duration, t *testing.T) {
	time.Sleep(timeout + 20*time.Millisecond) // wait for server to close connection
	_, msg, err := w.ws.Read(w.ctx)
	if err == nil {
		t.Fatalf("Unexpected message received: %s", msg)
	}
	if err := w.ws.Close(websocket.StatusNormalClosure, ""); err != nil {
		t.Fatalf("Failed to close websocket connection: %s", err)
	}
}

func TestDnsLeak(t *testing.T) {
	conf.DNS.Domain = "test"
	conf.DNS.Timeout = timeout
	dnsServer = &MockLogger[uint32]{
		callbacks: make(map[uint32]func(net.IP)),
	}
	ws := wsConnect("dns", t)
	params := new(dnsLeakTestParams)
	ws.readJson(params, t)
	if params.Base != conf.DNS.Domain {
		t.Errorf("Invalid base domain. Got %s, expected %s", params.Base, conf.DNS.Domain)
	}
	if len(params.Subdomains) != DNS_LEAK_TESTS_NUMBER {
		t.Errorf("Incorrect number of subdomains received. Got %d, expected %d", len(params.Subdomains), DNS_LEAK_TESTS_NUMBER)
	}
	ips := make([]net.IP, DNS_LEAK_TESTS_NUMBER)
	for i := range ips {
		if rand.Intn(2) == 0 {
			ips[i] = utils.RandomIPv4()
		} else {
			ips[i] = utils.RandomIPv6()
		}
	}
	go func() {
		for i, s := range params.Subdomains {
			k, err := strconv.ParseUint(s, 10, 32)
			if err != nil {
				t.Errorf("Invalid subdomain received: %s", s)
			}
			dnsServer.(*MockLogger[uint32]).callbacks[uint32(k)](ips[i])
		}
	}()
	for _, ip := range ips {
		ws.readAssertEqualsIP(ip, t)
	}
	ws.assertEnd(conf.DNS.Timeout, t)
}

func TestBittorrentLeak(t *testing.T) {
	conf.BitTorrent.Timeout = timeout
	bittorrentTracker = &MockLogger[bittorrent.InfoHash]{
		callbacks: make(map[bittorrent.InfoHash]func(net.IP)),
	}
	ws := wsConnect("bittorrent", t)
	infoHash := ws.readBinary(t)
	if len(infoHash) != 20 {
		t.Fatalf("Invalid %d bytes long info hash received: %s", len(infoHash), infoHash)
	}
	ip1 := utils.RandomIPv4()
	ip2 := utils.RandomIPv4()
	ip3 := utils.RandomIPv4()
	ips := make([]net.IP, 0, 5)
	ips = append(ips, ip1)
	ips = append(ips, ip2)
	ips = append(ips, ip2)
	ips = append(ips, ip1)
	ips = append(ips, ip3)
	go func() {
		for _, ip := range ips {
			bittorrentTracker.(*MockLogger[bittorrent.InfoHash]).callbacks[bittorrent.InfoHash(infoHash)](ip)
		}
	}()
	ws.readAssertEqualsIP(ip1, t)
	ws.readAssertEqualsIP(ip2, t)
	ws.readAssertEqualsIP(ip3, t)
	ws.assertEnd(conf.BitTorrent.Timeout, t)
}
