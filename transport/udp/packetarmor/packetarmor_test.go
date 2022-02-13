package packetarmor

import (
	"crypto/rand"
	"github.com/stretchr/testify/assert"
	"io"
	"testing"
)

func TestArmor(t *testing.T) {
	packetArmer := NewPacketArmor("Testing Password", "PacketArmor,VLite")
	var data [64]byte
	io.ReadFull(rand.Reader, data[:])
	result, err := packetArmer.Pack(data[:], 1200)
	assert.NoError(t, err)
	assert.Equal(t, 1200, len(result))
	original, err := packetArmer.Unpack(result)
	assert.NoError(t, err)
	assert.Equal(t, data[:], original)
}
