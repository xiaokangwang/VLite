package tcpServer

import (
	"bytes"
	"github.com/lunixbochs/struc"
	"github.com/stretchr/testify/assert"
	"github.com/xiaokangwang/VLite/proto"
	"github.com/xiaokangwang/VLite/workers/tcp/tcpClient"
	"testing"
)

func TestPrimary(t *testing.T) {
	var b = &bytes.Buffer{}
	dom := "123.111.222.333"
	port := uint16(80)

	tcpClient.WriteTcpDialHeader(b, dom, port)

	lw := &bytes.Buffer{}

	lenb := &proto.StreamConnectDomainHeaderLen{}

	lenb.Length = uint16(b.Len())

	struc.Pack(lw, lenb)

	bybuf := bytes.NewBuffer(nil)

	_, _ = bybuf.Write(lw.Bytes())
	_, _ = bybuf.Write(b.Bytes())

	h, err := ParseHeader(bytes.NewReader(bybuf.Bytes()))

	assert.Equal(t, false, err)
	assert.Equal(t, dom, h.DestDomain)
	assert.Equal(t, port, h.DestPort)
}
