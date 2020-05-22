package puniCommon

import (
	"context"
	"fmt"
	"github.com/xiaokangwang/VLite/interfaces/ibus"
	"github.com/xiaokangwang/VLite/interfaces/ibus/connidutil"
	"github.com/xiaokangwang/VLite/interfaces/ibus/ibusTopic"
	"github.com/xiaokangwang/VLite/interfaces/ibusInterface"
)

func ReHandshake(ctx context.Context) {

	msgbus := ibus.ConnectionMessageBusFromContext(ctx)

	connstr := connidutil.ConnIDToString(ctx)

	ConnReHandShake := ibusTopic.ConnReHandShake(connstr)

	msgbus.RegisterTopics(ConnReHandShake)

	w := ibusInterface.ConnReHandshake{}
	_, erremit := msgbus.Emit(ctx, ConnReHandShake, w)
	if erremit != nil {
		fmt.Println(erremit.Error())
	}
}
