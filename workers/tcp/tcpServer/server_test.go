package tcpServer

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"github.com/xiaokangwang/VLite/workers/tcp/tcpClient"
	"testing"
)

func TestPrimary(t *testing.T) {
	var b = &bytes.Buffer{}
	dom := "123.111.222.333"
	port := uint16(80)
	tcpClient.WriteTcpDialHeader(b, dom, port)

	h, err := ParseHeader(bytes.NewReader(b.Bytes()))

	assert.Equal(t, false, err)
	assert.Equal(t, dom, h.DestDomain)
	assert.Equal(t, port, h.DestPort)
}
