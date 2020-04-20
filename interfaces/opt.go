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
	ShouldNotReDial int64
}
