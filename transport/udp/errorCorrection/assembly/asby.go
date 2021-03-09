package assembly

import (
	"context"
	"github.com/patrickmn/go-cache"
	"github.com/xiaokangwang/VLite/interfaces"
	"github.com/xiaokangwang/VLite/transport/http/adp"
	rsf "github.com/xiaokangwang/VLite/transport/udp/errorCorrection/reconstruction/reedSolomon"
	"io"
	"net"
	"time"
)

func NewPacketAssembly(ctx context.Context, conn net.Conn) *PacketAssembly {
	pa := &PacketAssembly{}
	pa.ctx = ctx
	pa.ctx, pa.cancel = context.WithCancel(pa.ctx)

	pa.conn = conn

	pa.RxMaxTimeInSecond = 9
	pa.TxNextSeq = 1
	pa.TxRingBufferSize = 30
	pa.TxRingBuffer = make([]packetAssemblyTxChunkHolder, pa.TxRingBufferSize)
	pa.MaxDataShardPerChunk = 40
	pa.TxFECSoftPacketSoftLimitPerEpoch = 40
	pa.TxEpochTimeInMs = 35
	pa.RxChan = make(chan []byte, 8)
	pa.TxChan = make(chan []byte, 8)
	pa.TxNoFECChan = make(chan []byte, 8)

	pa.FECEnabled = 1

	pa.ecff = rsf.NewRSErrorCorrectionFacilityFactory()

	pa.RxReassembleBuffer = cache.New(
		time.Second*time.Duration(pa.RxMaxTimeInSecond),
		4*time.Second*time.Duration(pa.RxMaxTimeInSecond))

	eov := ctx.Value(interfaces.ExtraOptionsFECPacketAssemblyOpt)
	if eov != nil {
		eovs := eov.(*interfaces.ExtraOptionsFECPacketAssemblyOptValue)
		pa.RxMaxTimeInSecond = eovs.RxMaxTimeInSecond
		pa.TxEpochTimeInMs = eovs.TxEpochTimeInMs
	}

	go pa.Rx()
	go pa.Tx()
	go pa.Report()

	disablefec := ctx.Value(interfaces.ExtraOptionsDisableFEC)
	if disablefec != nil {
		if disablefec.(bool) == true {
			pa.FECEnabled = 0
			return pa
		}
	}
	go pa.boostingReceiver()

	return pa
}

type PacketAssembly struct {
	RxReassembleBuffer *cache.Cache

	TxNextSeq        uint32
	TxRingBuffer     []packetAssemblyTxChunkHolder
	TxRingBufferSize int

	RxChan      chan []byte
	TxChan      chan []byte
	TxNoFECChan chan []byte

	conn io.ReadWriteCloser

	ecff interfaces.ErrorCorrectionFacilityFactory

	ctx context.Context

	MaxDataShardPerChunk int
	RxMaxTimeInSecond    int

	FECEnabled uint32

	TxEpochTimeInMs                  int
	TxFECSoftPacketSoftLimitPerEpoch int

	TxEpocSeq uint64

	RxShardOriginalNoFEC uint64
	RxShardOriginal      uint64
	RxShardRecovered     uint64

	TxShardOriginal      uint64
	TxShardFEC           uint64
	TxShardOriginalNoFEC uint64

	RxBytes uint64
	TxBytes uint64

	cancel context.CancelFunc
}

func (pa *PacketAssembly) Close() error {
	return pa.conn.Close()
}

type PacketWireHead struct {
	Seq uint32 `struc:"uint32"`
	Id  int    `struc:"uint16"`
}

func (pa *PacketAssembly) AsConn() net.Conn {
	return adp.NewRxTxToConn(pa.TxChan, pa.RxChan, pa, pa.ctx)
}
