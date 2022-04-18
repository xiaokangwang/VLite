package interfaces

const ExtraOptionsUDPTimeoutTime = "ExtraOptionsUDPTimeoutTime"

type ExtraOptionsUDPTimeoutTimeValue struct {
	TimeoutTimeInSeconds int
}

const ExtraOptionsMessageBus = "ExtraOptionsMessageBus"

const ExtraOptionsMessageBusByConn = "ExtraOptionsMessageBusByConn"

const ExtraOptionsConnID = "ExtraOptionsConnID"

const ExtraOptionsBoostConnectionSettingsHTTPTransport = "ExtraOptionsBoostConnectionSettingsHTTPTransport"

type ExtraOptionsBoostConnectionSettingsHTTPTransportValue struct {
	MaxBoostTxConnection int
	MaxBoostRxConnection int
}

const ExtraOptionsHTTPTransportConnIsBoostConnection = "ExtraOptionsHTTPTransportConnIsBoostConnection"

const ExtraOptionsBoostConnectionGracefulShutdownRequest = "ExtraOptionsBoostConnectionGracefulShutdownRequest"

type ExtraOptionsBoostConnectionGracefulShutdownRequestValue struct {
	ShouldClose chan interface{}
}

const ExtraOptionsBoostConnectionShouldNotRedial = "ExtraOptionsBoostConnectionShouldNotRedial"

type ExtraOptionsBoostConnectionShouldNotRedialValue struct {
	ShouldNotReDial int32
}

const ExtraOptionsFECPacketAssemblyOpt = "ExtraOptionsFECPacketAssemblyOpt"

type ExtraOptionsFECPacketAssemblyOptValue struct {
	RxMaxTimeInSecond int
	TxEpochTimeInMs   int
}

const ExtraOptionsUDPFECEnabled = "ExtraOptionsUDPFECEnabled"

const ExtraOptionsHTTPUseSystemHTTPProxy = "ExtraOptionsHTTPUseSystemHTTPProxy"

const ExtraOptionsHTTPNetworkBufferSize = "ExtraOptionsHTTPNetworkBufferSize"

type ExtraOptionsHTTPNetworkBufferSizeValue struct {
	NetworkBufferSize int
}

const ExtraOptionsHTTPUseSystemSocksProxy = "ExtraOptionsHTTPUseSystemSocksProxy"

const ExtraOptionsHTTPDialAddr = "ExtraOptionsHTTPDialAddr"

type ExtraOptionsHTTPDialAddrValue struct {
	Addr string
}

const ExtraOptionsUniConnAttrib = "ExtraOptionsUniConnAttrib"

type ExtraOptionsUniConnAttribValue struct {
	ID   []byte
	Rand []byte
	Iter int32
}

const ExtraOptionsUseWebSocketInsteadOfHTTP = "ExtraOptionsUseWebSocketInsteadOfHTTP"

const ExtraOptionsUDPInitialData = "ExtraOptionsUDPInitialData"

type ExtraOptionsUDPInitialDataValue struct {
	Data []byte
}

const ExtraOptionsHTTPServerStreamRelay = "ExtraOptionsHTTPServerStreamRelay"

type ExtraOptionsHTTPServerStreamRelayValue struct {
	Relay StreamRelayer
}

const ExtraOptionsHTTPClientDialAlternativeChannel = "ExtraOptionsHTTPClientDialAlternativeChannel"

const ExtraOptionsUDPShouldMask = "ExtraOptionsUDPShouldMask"

const ExtraOptionsUDPMask = "ExtraOptionsUDPMask"

const ExtraOptionsAbstractDialer = "ExtraOptionsUseAbstractDialer"

type ExtraOptionsAbstractDialerValue struct {
	AbsDialer    AbstractDialer
	UseAbsDialer bool
}

const ExtraOptionsAbstractListener = "ExtraOptionsUseAbstractListener"

type ExtraOptionsAbstractListenerValue struct {
	AbsListener    AbstractListener
	UseAbsListener bool
}

const ExtraOptionsDisableFEC = "ExtraOptionsDisableFEC"

const ExtraOptionsUsePacketArmor = "ExtraOptionsUsePacketArmor"

type ExtraOptionsUsePacketArmorValue struct {
	PacketArmorPaddingTo int
	UsePacketArmor       bool
}

const ExtraOptionsDisableAutoQuitForClient = "ExtraOptionsDisableAutoQuitForClient"
