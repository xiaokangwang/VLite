package connidutil

import (
	"context"
	"encoding/hex"
	"github.com/xiaokangwang/VLite/interfaces"
	"strings"
)

func connIDToString(connid []byte) string {
	if len(connid) == 24 {
		//HTTPMuxerAddress
		return hex.EncodeToString(connid)
	} else {
		//Assume this is a IP-Port
		noDot := strings.ReplaceAll(string(connid), ".", "_")
		NoAny := strings.ReplaceAll(noDot, "[", "_")
		NoAny2 := strings.ReplaceAll(NoAny, "]", "_")
		if strings.ContainsAny(NoAny2, "\\.+*?()|[]{}^$") {
			return hex.EncodeToString(connid)
		}
		return NoAny2
	}
}

func ConnIDToString(ctx context.Context) string {
	val := ctx.Value(interfaces.ExtraOptionsConnID)
	bval := val.([]byte)
	return connIDToString(bval)
}
