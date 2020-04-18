package rawmask

import (
	"github.com/xiaokangwang/VLite/interfaces"
	"github.com/xiaokangwang/VLite/transport/udp/packetMasker/layers"
)

func GetRawMask() interfaces.Masker {
	NopLayer := layers.NewSyntheticLayerMasker([]interfaces.Masker{})
	return NopLayer
}
