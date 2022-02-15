package packetarmor

import (
	"crypto/rand"
	"github.com/stretchr/testify/assert"
	"io"
	"testing"
)

func TestArmor(t *testing.T) {
	packetArmerC := NewPacketArmor("Testing Password", "PacketArmor,VLite", true)
	var data [64]byte
	io.ReadFull(rand.Reader, data[:])
	result, err := packetArmerC.Pack(data[:], 1200)
	assert.NoError(t, err)
	assert.Equal(t, 1200, len(result))
	packetArmerS := NewPacketArmor("Testing Password", "PacketArmor,VLite", false)
	original, err := packetArmerS.Unpack(result)
	assert.NoError(t, err)
	assert.Equal(t, data[:], original)
}
