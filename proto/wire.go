package proto

const (
	CommandByte_Send = iota + 1
	CommandByte_SendV6
	CommandByte_Associate
	CommandByte_AssociateV6
	CommandByte_AssociateDone
	CommandByte_AssociateDoneV6
	CommandByte_ChannelDestroy
	CommandByte_Ping
	CommandByte_Pong
)

type CommandHeader struct {
	CommandByte uint8 `struc:"uint8"`
}
type DataHeader struct {
	Channel uint16 `struc:"uint16"`
}
type SendHeader struct {
	//CommandByte uint8   `struc:"uint8"`
	SourceIP   [4]byte `struc:"[4]byte"`
	DestIP     [4]byte `struc:"[4]byte"`
	SourcePort uint16  `struc:"uint16"`
	DestPort   uint16  `struc:"uint16"`
	PayloadLen uint16  `struc:"uint16,sizeof=Payload"`
	Payload    []byte  `struc:"[]byte"`
}

type SendV6Header struct {
	//CommandByte uint8    `struc:"uint8"`
	SourceIP   [16]byte `struc:"[16]byte"`
	DestIP     [16]byte `struc:"[16]byte"`
	SourcePort uint16   `struc:"uint16"`
	DestPort   uint16   `struc:"uint16"`
	PayloadLen uint16   `struc:"uint16,sizeof=Payload"`
	Payload    []byte   `struc:"[]byte"`
}

type AssociateHeader struct {
	//CommandByte uint8   `struc:"uint8"`
	SourceIP   [4]byte `struc:"[4]byte"`
	DestIP     [4]byte `struc:"[4]byte"`
	SourcePort uint16  `struc:"uint16"`
	DestPort   uint16  `struc:"uint16"`
	Channel    uint16  `struc:"uint16"`
}

type AssociateDoneHeader AssociateHeader

type AssociateV6Header struct {
	//CommandByte uint8    `struc:"uint8"`
	SourceIP   [16]byte `struc:"[16]byte"`
	DestIP     [16]byte `struc:"[16]byte"`
	SourcePort uint16   `struc:"uint16"`
	DestPort   uint16   `struc:"uint16"`
	Channel    uint16   `struc:"uint16"`
}

type AssociateDoneV6Header AssociateV6Header

type AssociateChannelDestroy struct {
	//CommandByte uint8   `struc:"uint8"`
	Channel uint16 `struc:"uint16"`
}

const (
	CommandByte_Stream_ConnectV4IP = iota + 1
	CommandByte_Stream_ConnectV6IP
	CommandByte_Stream_ConnectDomain
)

type StreamConnectDomainHeaderLen struct {
	Length uint16 `struc:"uint16"`
}

type StreamConnectDomainHeader struct {
	//CommandByte uint8    `struc:"uint8"`
	DestDomain_len uint16 `struc:"uint16,sizeof=DestDomain"`
	DestDomain     string `struc:"[]byte"`
	DestPort       uint16 `struc:"uint16"`
}

type PingHeader struct {
	Seq        uint64 `struc:"uint64"`
	Seq2       uint64 `struc:"uint64"`
	SentPacket uint64 `struc:"uint64"`
	RecvPacket uint64 `struc:"uint64"`
}
type PongHeader struct {
	SeqCopy    uint64 `struc:"uint64"`
	Seq2Copy   uint64 `struc:"uint64"`
	SentPacket uint64 `struc:"uint64"`
	RecvPacket uint64 `struc:"uint64"`
}

//WARNING: we assume this header is 8 byte! Since it is used to fill network buffer!
type HTTPLenHeader struct {
	Length int64 `struc:"int64"`
}

type HttpHeaderHolder struct {
	Masker   int64    `struc:"int64"`
	ConnID   [24]byte `struc:"[24]byte"`
	Rand     [24]byte `struc:"[24]byte"`
	Time     int64    `struc:"int64"`
	Flags    int64    `struc:"int64"`
	ConnIter int32    `struc:"int32"`
}

const HttpHeaderFlag_BoostConnection = 1 << 0
const HttpHeaderFlag_WebsocketConnection = 1 << 1
const HttpHeaderFlag_AlternativeChannelConnection = 1 << 2
