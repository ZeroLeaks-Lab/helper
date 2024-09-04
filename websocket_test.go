package main

import (
	"context"
	"math/rand"
	"net"
	"strconv"
	"testing"
	"time"
	"zeroleaks/utils"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

type MockDnsServer struct {
	subdomains map[uint32]func(net.IP)
}

func (s *MockDnsServer) RegisterCallback(k uint32, f func(net.IP)) {
	s.subdomains[k] = f
}

func TestDnsLeak(t *testing.T) {
	const addr = "127.0.0.1:38080"
	conf.DNS.Domain = "test"
	timeout = time.Millisecond * 100
	dnsServer = &MockDnsServer{
		subdomains: make(map[uint32]func(net.IP)),
	}
	go startWebsocketServer(addr, TLSConfig{}, websocket.AcceptOptions{})
	time.Sleep(10 * time.Millisecond) // let the websocket server start
	ctx := context.Background()
	ws, _, err := websocket.Dial(ctx, "ws://"+addr+"/v1/dns", nil)
	if err != nil {
		t.Fatalf("Cannot establish websocket connection: %s", err)
	}
	params := new(dnsLeakTestParams)
	if err = wsjson.Read(ctx, ws, params); err != nil {
		t.Fatalf("Failed to read dns leak test params: %s", err)
	}
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
			dnsServer.(*MockDnsServer).subdomains[uint32(k)](ips[i])
		}
	}()
	for i, ip := range ips {
		msgType, msg, err := ws.Read(ctx)
		if err != nil {
			t.Fatalf("Failed to read IP %d: %s", i, err)
		}
		if msgType != websocket.MessageText {
			t.Errorf("Invalid message type received: %s", msgType)
		} else {
			receivedIp := net.ParseIP(string(msg[:]))
			if !receivedIp.Equal(ip) {
				t.Errorf("Invalid IP received. Got %s, expected %s", msg, ip)
			}
		}
	}
	time.Sleep(2 * timeout) // wait for server to close connection
	_, msg, err := ws.Read(ctx)
	if err == nil {
		t.Fatalf("Unexpected message received: %s", msg)
	}
	if err = ws.Close(websocket.StatusNormalClosure, ""); err != nil {
		t.Fatalf("Failed to close websocket connection: %s", err)
	}
}
