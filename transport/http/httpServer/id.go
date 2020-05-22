package httpServer

import (
	"github.com/xiaokangwang/VLite/proto"
)

func synthid(hhh *proto.HttpHeaderHolder) string {
	syniter := hhh.ConnIter
	if syniter <= 0 {
		syniter = -syniter
	}
	return string(hhh.ConnID[:]) + string(syniter)
}
