package udpsctpserver

import (
	context "context"
	"fmt"
	dtls "github.com/pion/dtls/v2"
	"github.com/pion/logging"
	"github.com/pion/sctp"
	"github.com/xiaokangwang/VLite/interfaces"
	"github.com/xiaokangwang/VLite/transport/filteredConn"
	"github.com/xiaokangwang/VLite/transport/udp/errorCorrection/assembly"
	"github.com/xtaci/smux"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"runtime/debug"
	"time"
)

func NewPacketRelayServer(conn net.Conn,
	TxChannel chan interfaces.TrafficWithChannelTag,
	TxDataChannel chan interfaces.TrafficWithChannelTag,
	RxChannel chan interfaces.TrafficWithChannelTag,
	streamrelay interfaces.StreamRelayer,
	Password []byte,
	ctx context.Context) *PacketSCTPRelay {
	pse := &PacketSCTPRelay{
		conn:          conn,
		TxChannel:     TxChannel,
		TxDataChannel: TxDataChannel,
		RxChannel:     RxChannel,
		streamrelay:   streamrelay,
		Password:      Password,
		ctx:           ctx,
	}
	pse.tlsopenServer()

	go pse.Listen()
	go pse.PacketTx()
	return pse
}

func NewPacketRelayClient(conn net.Conn,
	TxChannel chan interfaces.TrafficWithChannelTag,
	TxDataChannel chan interfaces.TrafficWithChannelTag,
	RxChannel chan interfaces.TrafficWithChannelTag,
	Password []byte,
	ctx context.Context) *PacketSCTPRelay {
	pse := &PacketSCTPRelay{
		conn:          conn,
		TxChannel:     TxChannel,
		TxDataChannel: TxDataChannel,
		RxChannel:     RxChannel,
		Password:      Password,
		ctx:           ctx,
	}
	pse.tlsopenClient()
	go pse.ClientOpen()
	go pse.PacketTx()
	return pse
}

type PacketSCTPRelay struct {
	conn      net.Conn
	tlsconn   *dtls.Conn
	scconn    *sctp.Association
	scconnctl *sctp.Stream

	TxChannel     chan interfaces.TrafficWithChannelTag
	TxDataChannel chan interfaces.TrafficWithChannelTag
	RxChannel     chan interfaces.TrafficWithChannelTag
	streamrelay   interfaces.StreamRelayer

	Password []byte

	ctx context.Context

	muxer *smux.Session

	ratelimitServerTCPWriteBytePerSecond int
	ratelimitServerTCPWriteMaxBucketSize int
	ratelimitServerTCPWriteInitialSize   int

	packetReceivingFrontier *filteredConn.FilteredConn
}

var cipherSuiteIDS = []dtls.CipherSuiteID{dtls.TLS_PSK_WITH_AES_128_CCM_8}

var _ = logging.DefaultLoggerFactory{
	Writer:          os.Stdout,
	DefaultLogLevel: 0,
	ScopeLevels:     nil,
}

var loggingf = logging.NewDefaultLoggerFactory()

func (s *PacketSCTPRelay) RateLimitTcpServerWrite(ratelimitServerTCPWriteBytePerSecond int,
	ratelimitServerTCPWriteMaxBucketSize int,
	ratelimitServerTCPWriteInitialSize int) {
	s.ratelimitServerTCPWriteBytePerSecond = ratelimitServerTCPWriteBytePerSecond
	s.ratelimitServerTCPWriteMaxBucketSize = ratelimitServerTCPWriteMaxBucketSize
	s.ratelimitServerTCPWriteInitialSize = ratelimitServerTCPWriteInitialSize
}
func (s *PacketSCTPRelay) tlsopenServer() {
	dtlsserver, err := dtls.Server(s.conn, &dtls.Config{
		CipherSuites:           cipherSuiteIDS,
		SRTPProtectionProfiles: nil,
		ClientAuth:             0,
		ExtendedMasterSecret:   dtls.RequireExtendedMasterSecret,
		FlightInterval:         time.Second / 4,
		PSK: func(i2 []byte) (i []byte, err error) {
			return s.Password, nil
		},
		PSKIdentityHint:       nil,
		InsecureSkipVerify:    false,
		VerifyPeerCertificate: nil,
		RootCAs:               nil,
		ServerName:            "",
		LoggerFactory:         loggingf,
		MTU:                   1350,
		ConnectContextMaker: func() (ctx context.Context, f func()) {
			return context.WithTimeout(s.ctx, time.Second*30)
		},
	})

	if err != nil {
		log.Println(err)
	}

	s.tlsconn = dtlsserver

	fc := s.getConn(dtlsserver)
	s.scconn, err = sctp.Server(sctp.Config{
		NetConn:              fc,
		MaxReceiveBufferSize: 65536,
		LoggerFactory:        loggingf,
	})
	if err != nil {
		println(err)
	}

}

func (s *PacketSCTPRelay) tlsopenClient() {
	log.Println("Begin DTLS Handshake")
	dtlsserver, err := dtls.Client(s.conn, &dtls.Config{
		CipherSuites:           cipherSuiteIDS,
		SRTPProtectionProfiles: nil,
		ClientAuth:             0,
		ExtendedMasterSecret:   dtls.RequireExtendedMasterSecret,
		FlightInterval:         time.Second / 4,
		PSK: func(i2 []byte) (i []byte, err error) {
			return s.Password, nil
		},
		PSKIdentityHint:       []byte(""),
		InsecureSkipVerify:    false,
		VerifyPeerCertificate: nil,
		RootCAs:               nil,
		ServerName:            "",
		LoggerFactory:         loggingf,
		MTU:                   1350,
		ConnectContextMaker: func() (ctx context.Context, f func()) {
			return context.WithTimeout(s.ctx, time.Second*30)
		},
	})

	if err != nil {
		log.Println(err)
		debug.PrintStack()
		//TODO Error should be reported
	}

	s.tlsconn = dtlsserver
	log.Println("Over DTLS Handshake")
	log.Println("Begin SCTP Handshake")
	fc := s.getConn(dtlsserver)
	s.scconn, err = sctp.Client(sctp.Config{
		NetConn:              fc,
		MaxReceiveBufferSize: 65536,
		LoggerFactory:        loggingf,
	})
	if err != nil {
		log.Println(err)
		debug.PrintStack()
		//TODO Error should be reported
	}
	log.Println("Over SCTP Handshake")
}

func (s *PacketSCTPRelay) Listen() {
	for {
		if s.scconn == nil {
			return
		}
		st, err := s.scconn.AcceptStream()
		if err != nil {
			log.Println(err.Error())
			if s.ctx.Err() != nil {
				return
			}
			fmt.Println("Quitting Listener")
			return
		}
		id := st.StreamIdentifier()

		if id == 0 {
			s.scconnctl = st
			go s.CtlConn(st)
		} else if id == 1 {
			go s.tcprelayconn(st)
		} else {
			go io.Copy(ioutil.Discard, st)
		}
	}

}

func (s *PacketSCTPRelay) ClientOpen() {
	var err error
	if s.scconn == nil {
		return
	}
	s.scconnctl, err = s.scconn.OpenStream(0, sctp.PayloadTypeWebRTCBinary)
	if err != nil {
		log.Println(err.Error())
	}

	scnn, err2 := s.scconn.OpenStream(1, sctp.PayloadTypeWebRTCBinary)

	if err2 != nil {
		log.Println(err.Error())

	}

	scnn.SetReliabilityParams(false, sctp.ReliabilityTypeReliable, 0)

	conf := smux.DefaultConfig()
	conf.Version = 2
	conf.KeepAliveTimeout = 600 * time.Second

	s.muxer, err = smux.Client(NewBufferedConn(scnn), conf)
	if err != nil {
		log.Println(err.Error())
	}

	s.CtlConn(s.scconnctl)
}

func (s *PacketSCTPRelay) ClientOpenStream() io.ReadWriteCloser {

	scnn, err2 := s.muxer.Open()
	if err2 != nil {
		log.Println(err2.Error())
	}

	return scnn
}

func (s *PacketSCTPRelay) tcprelayconn(str *sctp.Stream) {
	var err error
	str.SetReliabilityParams(false, sctp.ReliabilityTypeReliable, 0)

	conf := smux.DefaultConfig()
	conf.Version = 2
	conf.KeepAliveTimeout = 600 * time.Second

	bufconn := NewBufferedConn(str)

	s.muxer, err = smux.Server(bufconn, conf)
	if err != nil {
		log.Println(err.Error())
	}

	for {
		conn, err2 := s.muxer.Accept()
		if err2 != nil {
			fmt.Println(err2.Error())
			return
		}
		if conn == nil {
			log.Println("conn is nil, why?")
			continue
		}
		go s.streamrelay.RelayStream(conn, s.ctx)

		if s.ctx.Err() != nil {
			return
		}
	}

}
func (s *PacketSCTPRelay) CtlConn(str *sctp.Stream) {
	str.SetReliabilityParams(true, sctp.ReliabilityTypeReliable, 0)
	for {
		data := make([]byte, 65536)
		n, err := str.Read(data)
		if err != nil {
			log.Println(err.Error())
			return
		}
		select {
		case s.RxChannel <- interfaces.TrafficWithChannelTag{
			Channel: 0,
			Payload: data[:n],
		}:
			continue
		case <-s.ctx.Done():
			return
		}
	}
}

func (s *PacketSCTPRelay) PacketTx() {
	for {
		select {
		case data := <-s.TxChannel:
			/*
				buf := bytes.NewBuffer(nil)
				datah := &proto.DataHeader{Channel: 0}

				err := struc.Pack(buf, datah)
				if err != nil {
					log.Println(err)
				}
				_, err = buf.Write(data.Payload)
				if err != nil {
					log.Println(err)
				}*/
			var err error
			for s.scconnctl == nil {
				time.Sleep(time.Second / 10)
			}
			_, err = s.scconnctl.Write(data.Payload)
			if err != nil {
				log.Println(err.Error())
			}
		case <-s.ctx.Done():
			return
		}

	}
}

func (s *PacketSCTPRelay) getConn(conn net.Conn) net.Conn {

	usageConn := conn

	if s.packetReceivingFrontier == nil {

		fec := false

		fecctxv := s.ctx.Value(interfaces.ExtraOptionsUDPFECEnabled)
		if fecctxv != nil {
			fec = fecctxv.(bool)
		}
		if fec {
			usageConn = assembly.NewPacketAssembly(s.ctx, usageConn).AsConn()
		}

		s.packetReceivingFrontier = filteredConn.NewFilteredConn(usageConn, s.TxDataChannel, s.RxChannel, s.ctx)
	}

	return s.packetReceivingFrontier
}
