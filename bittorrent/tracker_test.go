package bittorrent

import (
	"bytes"
	"encoding/binary"
	"math/rand"
	"net"
	"os"
	"testing"
	"time"
	"zeroleaks/utils"

	"github.com/lunixbochs/struc"
)

const addr = "127.0.0.1:31337"
const timeout = time.Millisecond * 100

var tracker *Tracker

func TestMain(m *testing.M) {
	t, err := NewTracker(addr, timeout)
	// Fuzzing spawns several processes in parallel. Consequently,
	// the tracker server will not be able to listen to on the same UDP port
	// if another process has already bound to it earlier. In the case of an
	// error is returned, we therefore consider that a tracker is already
	// listening in another process and we ignore the error. If no error is
	// returned, it means that we are the first process that tried to create
	// the tracker server and so we are in charge to start it.
	if err == nil {
		tracker = t
		go tracker.Start()
		time.Sleep(10 * time.Millisecond) // wait for the tracker to start
	}
	os.Exit(m.Run())
}

func send(c net.Conn, buff []byte, name string, t *testing.T) {
	n, err := c.Write(buff)
	if err != nil {
		utils.TFatalf(t, "Failed to send %s: %s", name, err)
	}
	if n != len(buff) {
		utils.TErrorf(t, "Incorrect amount of %s bytes sent: sent %d, expected %d", name, n, len(buff))
	}
}

func connect(t *testing.T) net.Conn {
	c, err := net.Dial("udp", addr)
	if err != nil {
		utils.TFatalf(t, "Cannot connect to the tracker: %s", err)
	}
	return c
}

func FuzzRandomPacket(f *testing.F) {
	f.Add([]byte{})
	f.Fuzz(func(t *testing.T, packet []byte) {
		c := connect(t)
		if len(packet) >= 12 {
			// insert valid action
			binary.BigEndian.PutUint32(packet[8:], uint32(rand.Intn(2)))
		}
		send(c, packet, "random packet", t)
		if err := c.Close(); err != nil {
			utils.TErrorf(t, "Failed to close connection: %s", err)
		}
	})
}

func TestTracker(t *testing.T) {
	infoHash := InfoHash(utils.RandomBytes(20))
	var requestIp net.IP
	tracker.RegisterCallback(infoHash, func(ip net.IP) {
		requestIp = ip
	})
	unknownInfoHash := InfoHash(utils.RandomBytes(20))
	tracker.RegisterCallback(unknownInfoHash, func(ip net.IP) {
		utils.TErrorf(t, "Callback for info hash %x unexpectedly called with IP %s", unknownInfoHash, ip)
	})

	c := connect(t)
	// sending connect request
	trId := rand.Uint32()
	sendBuff := bytes.NewBuffer(make([]byte, 0, CONNECT_REQUEST_SIZE))
	if err := struc.Pack(sendBuff, &ConnectRequest{
		ProtocolId:    PROTOCOL_ID,
		Action:        ACTION_CONNECT,
		TransactionId: trId,
	}); err != nil {
		utils.TFatalf(t, "Failed to pack connect request: %s", err)
	}
	send(c, sendBuff.Bytes(), "connect request", t)
	// receiving connect response
	recvBuff := make([]byte, max(CONNECT_RESPONSE_SIZE, ANNOUNCE_RESPONSE_SIZE))
	n, err := c.Read(recvBuff)
	if err != nil {
		utils.TFatalf(t, "Failed to read connect response: %s", err)
	}
	if n != CONNECT_RESPONSE_SIZE {
		utils.TErrorf(t, "Unexpected connect response size: received %d bytes, expected %d", n, CONNECT_RESPONSE_SIZE)
	}
	var connectResponse ConnectResponse
	if err = struc.Unpack(bytes.NewBuffer(recvBuff), &connectResponse); err != nil {
		utils.TFatalf(t, "Failed to unpack connect response: %s", err)
	}
	if connectResponse.Action != ACTION_CONNECT {
		utils.TErrorf(t, "Invalid connect response action: received %x, expected %x", connectResponse.Action, ACTION_CONNECT)
	}
	if connectResponse.TransactionId != trId {
		utils.TErrorf(t, "Incorrect transaction_id received: %d, expected %d", connectResponse.TransactionId, trId)
	}
	// sending announce request
	trId = rand.Uint32()
	sendBuff.Reset()
	if err = struc.Pack(sendBuff, &AnnounceRequest{
		ConnectionId:  connectResponse.ConnectionId,
		Action:        ACTION_ANNOUNCE,
		TransactionId: trId,
		InfoHash:      infoHash,
	}); err != nil {
		utils.TFatalf(t, "Failed to pack announce request: %s", err)
	}
	send(c, sendBuff.Bytes(), "announce request", t)
	// receiving announce response
	n, err = c.Read(recvBuff)
	if err != nil {
		utils.TFatalf(t, "Failed to read announce response: %s", err)
	}
	if n != ANNOUNCE_RESPONSE_SIZE {
		utils.TErrorf(t, "Unexpected announce response size: received %d bytes, expected %d", n, ANNOUNCE_RESPONSE_SIZE)
	}
	var announceResponse AnnounceResponse
	if err = struc.Unpack(bytes.NewBuffer(recvBuff), &announceResponse); err != nil {
		utils.TFatalf(t, "Failed to unpack announce response: %s", err)
	}
	if announceResponse.Action != ACTION_ANNOUNCE {
		utils.TErrorf(t, "Invalid announce response action: received %x, expected %x", announceResponse.Action, ACTION_ANNOUNCE)
	}
	if announceResponse.TransactionId != trId {
		utils.TErrorf(t, "Incorrect transaction_id received: %d, expected %d", announceResponse.TransactionId, trId)
	}

	if !requestIp.Equal(net.IPv4(127, 0, 0, 1)) {
		utils.TErrorf(t, "Invalid request IP: %s", requestIp)
	}
}
