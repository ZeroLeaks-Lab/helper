package bittorrent

import (
	"bytes"
	"encoding/binary"
	"log"
	"math/rand"
	"net"
	"time"

	"github.com/jellydator/ttlcache/v3"
	"github.com/lunixbochs/struc"
)

const (
	INTERVAL_SECS   = 0
	TRACKER_LOG_TAG = "BitTorrent Tracker:"

	PROTOCOL_ID = 0x41727101980

	CONNECT_REQUEST_SIZE   = 16
	CONNECT_RESPONSE_SIZE  = 16
	ANNONCE_REQUEST_SIZE   = 98
	ANNOUNCE_RESPONSE_SIZE = 20

	ACTION_CONNECT  = 0
	ACTION_ANNOUNCE = 1
	ACTION_SCRAPE   = 2
)

type InfoHash [20]byte

type ConnectRequest struct {
	ProtocolId    int64
	Action        int32
	TransactionId uint32
}

type ConnectResponse struct {
	Action        int32
	TransactionId uint32
	ConnectionId  uint64
}

type AnnounceRequest struct {
	ConnectionId  uint64
	Action        int32
	TransactionId uint32
	InfoHash      InfoHash
	PeerId        [20]byte
	Downloaded    int64
	Left          int64
	Uploaded      int64
	Event         int32
	IPAddress     uint32
	Key           uint32
	NumWant       int32
	Port          uint16
}

type AnnounceResponse struct {
	Action        int32
	TransactionId uint32
	Interval      int32
	Leechers      int32
	Seeders       int32
}

type Tracker struct {
	udpServer  net.PacketConn
	infoHashes *ttlcache.Cache[InfoHash, func(net.IP)]
}

func NewTracker(addr string, timeout time.Duration) (*Tracker, int, error) {
	server, err := net.ListenPacket("udp", addr)
	if err != nil {
		return nil, -1, err
	}
	tracker := Tracker{
		udpServer:  server,
		infoHashes: ttlcache.New(ttlcache.WithTTL[InfoHash, func(net.IP)](timeout)),
	}
	go tracker.infoHashes.Start()
	return &tracker, server.LocalAddr().(*net.UDPAddr).Port, nil
}

func (t *Tracker) RegisterCallback(k InfoHash, f func(net.IP)) {
	t.infoHashes.Set(k, f, ttlcache.DefaultTTL)
}

func (t *Tracker) reply(dst net.Addr, response interface{}, size int) {
	buff := bytes.NewBuffer(make([]byte, 0, size))
	struc.Pack(buff, response)
	n, err := t.udpServer.WriteTo(buff.Bytes(), dst)
	if err != nil {
		log.Printf("%s Error while sending UDP packet to %s: %s", TRACKER_LOG_TAG, dst, err)
		return
	}
	if n != size {
		log.Printf("%s Error: Incorrect amount of bytes sent to %s: written %d instead of %d", TRACKER_LOG_TAG, dst, n, size)
		return
	}
}

func (t *Tracker) handleConnect(src net.Addr, buff []byte) {
	if len(buff) < CONNECT_REQUEST_SIZE {
		log.Printf("%s Error: incomplete connect request size received from %s: %d", TRACKER_LOG_TAG, src, len(buff))
		return
	}
	var connectRequest ConnectRequest
	err := struc.Unpack(bytes.NewBuffer(buff), &connectRequest)
	if err != nil {
		log.Printf("%s Error: failed to unpack connect request from %s: %s", TRACKER_LOG_TAG, src, err)
		return
	}
	if connectRequest.ProtocolId != PROTOCOL_ID {
		log.Printf("%s Warning: unkown protocol_id received from %s: %x", TRACKER_LOG_TAG, src, connectRequest.ProtocolId)
	}
	connectionId := rand.Uint64()
	connectResponse := ConnectResponse{
		Action:        ACTION_CONNECT,
		TransactionId: connectRequest.TransactionId,
		ConnectionId:  connectionId,
	}
	t.reply(src, &connectResponse, CONNECT_RESPONSE_SIZE)
}

func (t *Tracker) handleAnnounce(src net.Addr, buff []byte) {
	if len(buff) < CONNECT_REQUEST_SIZE {
		log.Printf("%s Error: incomplete announce request size received from %s: %d", TRACKER_LOG_TAG, src, len(buff))
		return
	}
	var announceRequest AnnounceRequest
	err := struc.Unpack(bytes.NewBuffer(buff), &announceRequest)
	if err != nil {
		log.Printf("%s Error: failed to unpack announce request from %s: %s", TRACKER_LOG_TAG, src, err)
		return
	}
	entry := t.infoHashes.Get(announceRequest.InfoHash)
	if entry != nil {
		entry.Value()(src.(*net.UDPAddr).IP)
	}
	if announceRequest.IPAddress != 0 {
		ip := make(net.IP, 4)
		binary.BigEndian.PutUint32(ip, announceRequest.IPAddress)
		log.Printf("%s Warning: IP address field not supported. %s set it to: %s:%d", TRACKER_LOG_TAG, src, ip, announceRequest.Port)
	}
	announceResponse := AnnounceResponse{
		Action:        ACTION_ANNOUNCE,
		TransactionId: announceRequest.TransactionId,
		Interval:      INTERVAL_SECS,
		Leechers:      0,
		Seeders:       0,
	}
	t.reply(src, &announceResponse, ANNOUNCE_RESPONSE_SIZE)
}

func (t *Tracker) Start() {
	buff := make([]byte, max(CONNECT_REQUEST_SIZE, ANNONCE_REQUEST_SIZE))
	for {
		n, src, err := t.udpServer.ReadFrom(buff)
		if err != nil {
			log.Printf("%s Error while reading UDP packet from %s: %s", TRACKER_LOG_TAG, src, err)
			continue
		}
		if n < 12 {
			log.Printf("%s Error: invalid packet size received from %s: %d", TRACKER_LOG_TAG, src, n)
			continue
		}
		action := binary.BigEndian.Uint32(buff[8:12])
		switch action {
		case ACTION_CONNECT:
			t.handleConnect(src, buff[:n])
		case ACTION_ANNOUNCE:
			t.handleAnnounce(src, buff[:n])
		case ACTION_SCRAPE:
			// not implemented
		default:
			log.Printf("%s Error: invalid action received from %s: %x", TRACKER_LOG_TAG, src, action)
		}
	}
}
