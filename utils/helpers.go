package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"runtime/debug"
	"testing"
)

func RandomBytes(size int) []byte {
	bytes := make([]byte, size)
	if _, err := rand.Read(bytes); err != nil {
		log.Panicf("Failed to read %d random bytes: %s", size, err)
	}
	return bytes
}

func RandomIPv4() net.IP {
	bytes := RandomBytes(4)
	return net.IPv4(bytes[0], bytes[1], bytes[2], bytes[3])
}

func RandomIPv6() net.IP {
	bytes := RandomBytes(16)
	return net.ParseIP(fmt.Sprintf("%s:%s:%s:%s:%s:%s:%s:%s",
		hex.EncodeToString(bytes[:2]),
		hex.EncodeToString(bytes[2:4]),
		hex.EncodeToString(bytes[4:6]),
		hex.EncodeToString(bytes[6:8]),
		hex.EncodeToString(bytes[8:10]),
		hex.EncodeToString(bytes[10:12]),
		hex.EncodeToString(bytes[12:14]),
		hex.EncodeToString(bytes[14:]),
	))
}

func LogStack(t *testing.T) {
	t.Log(string(debug.Stack()))
}

func TErrorf(t *testing.T, format string, args ...any) {
	LogStack(t)
	t.Errorf(format, args...)
}

func TFatalf(t *testing.T, format string, args ...any) {
	LogStack(t)
	t.Fatalf(format, args...)
}
