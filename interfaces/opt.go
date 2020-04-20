package interfaces

const ExtraOptionsUDPTimeoutTime = "ExtraOptionsUDPTimeoutTime"

type ExtraOptionsUDPTimeoutTimeValue struct {
	TimeoutTimeInSeconds int
}

const ExtraOptionsMessageBus = "ExtraOptionsMessageBus"

const ExtraOptionsConnID = "ExtraOptionsConnID"

const ExtraOptionsBoostConnectionSettingsHTTPTransport = "ExtraOptionsBoostConnectionSettingsHTTPTransport"

type ExtraOptionsBoostConnectionSettingsHTTPTransportValue struct {
	MaxBoostTxConnection int
	MaxBoostRxConnection int
}

const ExtraOptionsHTTPTransportConnIsBoostConnection = "ExtraOptionsHTTPTransportConnIsBoostConnection"
