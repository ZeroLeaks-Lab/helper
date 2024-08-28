package main

import (
	"log"
	"net"
	"strconv"
	"strings"

	"github.com/jellydator/ttlcache/v3"
	"github.com/miekg/dns"
)

var subdomains *ttlcache.Cache[uint32, func(net.IP)]

func onRequest(domain string, ip net.IP) {
	p := strings.Index(domain, conf.DNS.Domain)
	if p < 1 {
		return
	}
	subdomain := domain[:p-1]
	if s, err := strconv.ParseUint(subdomain, 10, 32); err == nil {
		entry := subdomains.Get(uint32(s))
		if entry != nil {
			entry.Value()(ip)
		}
	}
}

func startDnsServer(addr, domain string) {
	subdomains = ttlcache.New(ttlcache.WithTTL[uint32, func(net.IP)](timeout))
	go subdomains.Start()
	dns.HandleFunc(domain, func(w dns.ResponseWriter, m *dns.Msg) {
		onRequest(strings.ToLower(m.Question[0].Name), w.RemoteAddr().(*net.UDPAddr).IP)
		r := dns.Msg{}
		r.SetReply(m)
		r.Rcode = dns.RcodeNameError // avoid being queried again
		w.WriteMsg(&r)
	})
	if err := (&dns.Server{Net: "udp", Addr: addr}).ListenAndServe(); err != nil {
		log.Fatalln("Failed to start DNS server:", err)
	}
}

func registerDnsCallback(t uint32, f func(net.IP)) {
	subdomains.Set(t, f, ttlcache.DefaultTTL)
}
