package httpServer

import "github.com/xiaokangwang/VLite/proto"

func synthid(hhh *proto.HttpHeaderHolder) string {
	return string(hhh.ConnID[:]) + string(hhh.ConnIter)
}
