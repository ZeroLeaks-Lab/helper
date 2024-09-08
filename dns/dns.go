package dns

import (
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/jellydator/ttlcache/v3"
	"github.com/miekg/dns"
)

type DnsServer struct {
	topDomain  string
	subdomains *ttlcache.Cache[uint32, func(net.IP)]
}

func NewServer(topDomain string, timeout time.Duration) *DnsServer {
	server := DnsServer{
		topDomain:  topDomain,
		subdomains: ttlcache.New(ttlcache.WithTTL[uint32, func(net.IP)](timeout)),
	}
	go server.subdomains.Start()
	return &server
}

func (s *DnsServer) RegisterCallback(k uint32, f func(net.IP)) {
	s.subdomains.Set(k, f, ttlcache.DefaultTTL)
}

func (s *DnsServer) onRequest(domain string, ip net.IP) {
	p := strings.Index(domain, s.topDomain)
	if p < 1 {
		return
	}
	subdomain := domain[:p-1]
	if k, err := strconv.ParseUint(subdomain, 10, 32); err == nil {
		entry := s.subdomains.Get(uint32(k))
		if entry != nil {
			entry.Value()(ip)
		}
	}
}

func (s *DnsServer) Start(addr string) {
	dns.HandleFunc(s.topDomain, func(w dns.ResponseWriter, m *dns.Msg) {
		s.onRequest(strings.ToLower(m.Question[0].Name), w.RemoteAddr().(*net.UDPAddr).IP)
		r := dns.Msg{}
		r.SetReply(m)
		r.Rcode = dns.RcodeNameError // avoid being queried again
		w.WriteMsg(&r)
	})
	if err := (&dns.Server{Net: "udp", Addr: addr}).ListenAndServe(); err != nil {
		log.Fatalln("Failed to start DNS server:", err)
	}
}
