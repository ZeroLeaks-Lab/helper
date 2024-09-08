package dns

import (
	"math/rand"
	"net"
	"os"
	"strconv"
	"testing"
	"time"
	"zeroleaks/utils"

	"github.com/miekg/dns"
)

const invalidDomain = "invalid.domain"
const addr = "127.0.0.1:35353"
const timeout = time.Millisecond * 100

var server DnsServer

func TestMain(m *testing.M) {
	server = *NewServer("test", timeout)
	os.Exit(m.Run())
}

func fullDomainFromKey(key uint32) string {
	return strconv.FormatUint(uint64(key), 10) + "." + server.topDomain
}

func TestCallback(t *testing.T) {
	ipv4 := utils.RandomIPv4()
	ipv6 := utils.RandomIPv6()
	expectedIP := ipv4
	key := rand.Uint32()
	server.RegisterCallback(key, func(ip net.IP) {
		if !ip.Equal(expectedIP) {
			utils.TErrorf(t, "Callback for subdomain %d called with wrong IP. Got %s, expected %s", key, ip, expectedIP)
		} else if expectedIP.Equal(ipv4) {
			expectedIP = ipv6
		} else {
			expectedIP = nil
		}
	})
	server.onRequest(invalidDomain, ipv6) // callback should not be triggered
	server.onRequest("", ipv6)            // callback should not be triggered
	domain := fullDomainFromKey(key)
	server.onRequest(domain, expectedIP)
	if !expectedIP.Equal(ipv6) {
		utils.TErrorf(t, "Callback for subdomain %d not called with IPv4: %s", key, ipv4)
	}
	server.onRequest(domain, expectedIP)
	if expectedIP != nil {
		utils.TErrorf(t, "Callback for subdomain %d not called with IPv6: %s", key, ipv6)
	}
	server.onRequest(domain, expectedIP)
	server.onRequest(invalidDomain, ipv4) // callback should not be triggered
}

func TestExpiration(t *testing.T) {
	key := rand.Uint32()
	server.RegisterCallback(key, func(ip net.IP) {
		utils.TErrorf(t, "Callback for subdomain %d unexpectedly called with IP %s", key, ip)
	})
	if !server.subdomains.Has(key) {
		utils.TErrorf(t, "Subdomain %d not registered", key)
	}
	time.Sleep(timeout + 20*time.Millisecond) // add 20ms margin
	if server.subdomains.Has(key) {
		utils.TErrorf(t, "Subdomain %d not expired after %s", key, timeout)
	}

	expectedIP := utils.RandomIPv4()
	called := false
	server.RegisterCallback(key, func(ip net.IP) {
		called = true
		if !ip.Equal(expectedIP) {
			utils.TErrorf(t, "Callback for subdomain %d called with wrong IP. Got %s, expected %s", key, ip, expectedIP)
		}
	})
	time.Sleep(timeout - 20*time.Millisecond)
	if !server.subdomains.Has(key) {
		utils.TErrorf(t, "Subdomain %d expired before timeout", key)
	}
	server.onRequest(fullDomainFromKey(key), expectedIP)
	time.Sleep(40 * time.Millisecond)
	if server.subdomains.Has(key) {
		utils.TErrorf(t, "Subdomain %d not expired after more than %s passed", key, timeout)
	}
	if !called {
		utils.TErrorf(t, "Callback not triggered")
	}
}

func query(t *testing.T, c *dns.Client, domain string, expectedRcode int) {
	m := new(dns.Msg)
	m.SetQuestion(domain, dns.TypeA)
	r, _, err := c.Exchange(m, addr)
	if err != nil {
		utils.TFatalf(t, "Client error: %s", err)
	}
	if r.Rcode != expectedRcode {
		utils.TErrorf(t, "Invalid error code: %d. Expected %d", r.Rcode, expectedRcode)
	}
}

func TestDnsServer(t *testing.T) {
	go server.Start(addr)
	time.Sleep(20 * time.Millisecond) // wait for the server to start
	c := new(dns.Client)
	query(t, c, invalidDomain+".", dns.RcodeRefused)
	query(t, c, ".", dns.RcodeRefused)
	query(t, c, "unknown."+server.topDomain+".", dns.RcodeNameError)
	key := rand.Uint32()
	var requestIp net.IP
	server.RegisterCallback(key, func(ip net.IP) {
		requestIp = ip
	})
	query(t, c, fullDomainFromKey(key)+".", dns.RcodeNameError)
	if !requestIp.Equal(net.IPv4(127, 0, 0, 1)) {
		utils.TErrorf(t, "Invalid request IP: %s", requestIp)
	}
}
