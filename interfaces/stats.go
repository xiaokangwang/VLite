package interfaces

type GetTransmitLayerSentRecvStats interface {
	GetTransmitLayerSentRecvStats() (uint64, uint64)
}
