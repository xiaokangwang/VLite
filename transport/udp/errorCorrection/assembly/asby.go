package assembly

import (
	"context"
	"github.com/patrickmn/go-cache"
	"github.com/xiaokangwang/VLite/interfaces"
	rsf "github.com/xiaokangwang/VLite/transport/udp/errorCorrection/reconstruction/reedSolomon"
	"io"
	"net"
	"time"
)

func NewPacketAssembly(ctx context.Context, conn net.Conn) *PacketAssembly {
	pa := &PacketAssembly{}
	pa.ctx = ctx
	pa.conn = conn

	pa.RxMaxTimeInSecond = 4
	pa.TxNextSeq = 1
	pa.TxRingBufferSize = 30
	pa.TxRingBuffer = make([]packetAssemblyTxChunkHolder, pa.TxRingBufferSize)
	pa.MaxDataShardPerChunk = 40
	pa.TxFECSoftPacketSoftLimitPerEpoch = 40
	pa.TxEpochTimeInMs = 30
	pa.RxChan = make(chan []byte, 8)
	pa.TxChan = make(chan []byte, 8)
	pa.TxNoFECChan = make(chan []byte, 8)

	pa.FECEnabled = 1

	pa.ecff = rsf.NewRSErrorCorrectionFacilityFactory()

	pa.RxReassembleBuffer = cache.New(
		time.Second*time.Duration(pa.RxMaxTimeInSecond),
		4*time.Second*time.Duration(pa.RxMaxTimeInSecond))
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
}

type PacketWireHead struct {
	Seq uint32 `struc:"uint32"`
	Id  int    `struc:"uint16"`
}
