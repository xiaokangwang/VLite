package assembly

import (
	"fmt"
	"time"
)

func (pa *PacketAssembly) Report() {
	for pa.ctx.Err() == nil {
		time.Sleep(time.Second * 3)
		fmt.Printf("FECConn RxSize %v TxSize %v RxOrig %v RxRecover %v RxNoFEC %v TxOrig %v TxFEC %v TxNoFEC %v\n", pa.RxBytes, pa.TxBytes,
			pa.RxShardOriginal, pa.RxShardRecovered, pa.RxShardOriginalNoFEC,
			pa.TxShardOriginal, pa.TxShardFEC, pa.TxShardOriginalNoFEC)
	}
}
