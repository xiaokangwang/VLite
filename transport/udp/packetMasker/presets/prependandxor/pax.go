package prependandxor

import (
	"github.com/xiaokangwang/VLite/interfaces"
	"github.com/xiaokangwang/VLite/transport/udp/packetMasker/constantXor"
	"github.com/xiaokangwang/VLite/transport/udp/packetMasker/layers"
	"github.com/xiaokangwang/VLite/transport/udp/packetMasker/polyxor"
	"github.com/xiaokangwang/VLite/transport/udp/packetMasker/prepend"
)

func GetPrependAndXorMask(pw string, prependPattern []byte) interfaces.Masker {
	XorLayer := constantXor.NewXorMasker(pw)
	prependLayer := prepend.NewPrependingMasker(prependPattern)

	SynLayer := layers.NewSyntheticLayerMasker([]interfaces.Masker{XorLayer, prependLayer})
	return SynLayer
}

func GetPrependAndPolyXorMask(pw string, prependPattern []byte) interfaces.Masker {
	XorLayer := polyxor.NewPolyXorMasker(pw)
	CXorLayer := constantXor.NewXorMasker(pw)
	prependLayer := prepend.NewPrependingMasker(prependPattern)

	SynLayer := layers.NewSyntheticLayerMasker([]interfaces.Masker{XorLayer, CXorLayer, prependLayer})
	return SynLayer
}
