package puniCommon

import (
	"context"
	"fmt"
	"github.com/xiaokangwang/VLite/interfaces/ibus"
	"github.com/xiaokangwang/VLite/interfaces/ibus/connidutil"
	"github.com/xiaokangwang/VLite/interfaces/ibus/ibusTopic"
	"github.com/xiaokangwang/VLite/interfaces/ibusInterface"
)

func ReHandshake2(ctx context.Context, rehandshake bool) {
	fmt.Println("Rehandshake", rehandshake)

	msgbus := ibus.ConnectionMessageBusFromContext(ctx)

	connstr := connidutil.ConnIDToString(ctx)

	ConnReHandShake := ibusTopic.ConnReHandShake(connstr)

	msgbus.RegisterTopics(ConnReHandShake)

	w := ibusInterface.ConnReHandshake{
		Fire:            true,
		FullReHandshake: rehandshake,
	}
	_, erremit := msgbus.Emit(ctx, ConnReHandShake, w)
	if erremit != nil {
		fmt.Println(erremit.Error())
	}
}
