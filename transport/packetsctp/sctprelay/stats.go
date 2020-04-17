package udpsctpserver

func (s *PacketSCTPRelay) GetTransmitLayerSentRecvStats() (uint64, uint64) {
	return s.packetReceivingFrontier.GetPacketStatus()
}
