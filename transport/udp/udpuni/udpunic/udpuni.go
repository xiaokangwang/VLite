package udpunic

import (
	"context"
	"crypto/rand"
	"fmt"
	"github.com/xiaokangwang/VLite/interfaces"
	"github.com/xiaokangwang/VLite/proto"
	"github.com/xiaokangwang/VLite/transport"
	"github.com/xiaokangwang/VLite/transport/http/headerHolder"
	"github.com/xiaokangwang/VLite/transport/udp/packetarmor"
	"io"
	"log"
	mrand "math/rand"
	"net"
	"reflect"
	"time"
)

func NewUdpUniClient(password string,
	ctx context.Context, under transport.UnderlayTransportDialer) *UdpUniClient {
	return &UdpUniClient{
		password: password,
		ctx:      ctx,
		hh:       headerHolder.NewHttpHeaderHolderProcessor2(password, "UdpUniSecret"),
		armor:    packetarmor.NewPacketArmor(password, "UdpUniPacketArmor", true),
		under:    under,
	}
}

type UdpUniClient struct {
	password string
	ctx      context.Context

	hh    *headerHolder.HttpHeaderHolderProcessor
	armor *packetarmor.PacketArmor

	under transport.UnderlayTransportDialer
}

func (uuc *UdpUniClient) Connect(ctx context.Context) (net.Conn, error, context.Context) {
	conn, err, connctx := uuc.under.Connect(ctx)
	if err != nil {
		return nil, err, nil
	}
	uc := &udpUniClientProxy{
		ctx:     connctx,
		initBuf: nil,
		conn:    conn,
		armor:   uuc.armor,
	}

	ph := proto.HttpHeaderHolder{}

	id := make([]byte, 24)
	io.ReadFull(rand.Reader, id)

	unival := ctx.Value(interfaces.ExtraOptionsUniConnAttrib)

	if unival != nil {
		uniAtt := unival.(*interfaces.ExtraOptionsUniConnAttribValue)
		copy(id, uniAtt.ID)
	}

	copy(ph.ConnID[:], id)

	if unival != nil {
		uniAtt := unival.(*interfaces.ExtraOptionsUniConnAttribValue)
		ph.ConnIter = uniAtt.Iter
	}

	mrand.Read(ph.Rand[:])
	ph.Time = time.Now().Unix()

	w := uuc.hh.Seal(ph)

	packetArmorPaddingTo := 0
	packetArmorVal := ctx.Value(interfaces.ExtraOptionsUsePacketArmor)
	if packetArmorVal != nil {
		packetArmorVal := packetArmorVal.(*interfaces.ExtraOptionsUsePacketArmorValue)
		if packetArmorVal.UsePacketArmor {
			packetArmorPaddingTo = packetArmorVal.PacketArmorPaddingTo
		}
	}

	err = uc.UniHandShake(w, packetArmorPaddingTo)
	if err != nil {
		return nil, err, nil
	}
	return uc, nil, uc.ctx
}

func (uucp *udpUniClientProxy) UniHandShake(token string, packetarmorPaddingTo int) error {
	var err error
	var n int
	uucp.token = token
	if packetarmorPaddingTo != 0 {
		uucp.shouldUseArmor = true
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		ticker := time.NewTicker(time.Second / 4)
		defer ticker.Stop()
		for i := 0; i < 300; i++ {
			select {
			case <-ticker.C:
				if packetarmorPaddingTo != 0 {
					pack, err := uucp.armor.Pack([]byte(token), packetarmorPaddingTo)
					if err != nil {
						log.Println("unable to create pack", err)
					}
					uucp.conn.Write(pack)
				} else {
					uucp.conn.Write([]byte(token))
				}

				break
			case <-ctx.Done():
				return
			}
		}
	}()
	for i := 0; i < 300; i++ {
		var buf [1600]byte
		n, err = uucp.conn.Read(buf[:])

		if err == nil {
			if packetarmorPaddingTo == 0 {
				if !reflect.DeepEqual(buf[:n], []byte(token)) {
					uucp.initBuf = buf[:n]
				}
			} else {
				decrypted, err := uucp.armor.Unpack(buf[:n])
				if err != nil {
					continue
				}
				if !reflect.DeepEqual(decrypted, []byte(token)) {
					err = nil
					break
				}
			}
			fmt.Println("Uni Handshake Done")
			return nil
		}
	}
	return err
}

type udpUniClientProxy struct {
	ctx            context.Context
	initBuf        []byte
	token          string
	conn           net.Conn
	armor          *packetarmor.PacketArmor
	shouldUseArmor bool

	recvCount int
}

func (uucp *udpUniClientProxy) Read(b []byte) (n int, err error) {
	if uucp.initBuf != nil {
		n = copy(b, uucp.initBuf)
		uucp.initBuf = nil
		return n, nil
	}
	for {
		n, err = uucp.conn.Read(b)
		uucp.recvCount++
		if uucp.recvCount <= 300 {
			if uucp.shouldUseArmor {
				_, err := uucp.armor.Unpack(b[:n])
				if err == nil {
					continue
				}
			} else {
				if reflect.DeepEqual(b[:n], []byte(uucp.token)) {
					continue
				}
			}
		}

		return
	}
}

func (uucp *udpUniClientProxy) Write(b []byte) (n int, err error) {
	return uucp.conn.Write(b)
}

func (uucp *udpUniClientProxy) Close() error {
	return uucp.conn.Close()
}

func (uucp *udpUniClientProxy) LocalAddr() net.Addr {
	return uucp.conn.LocalAddr()
}

func (uucp *udpUniClientProxy) RemoteAddr() net.Addr {
	return uucp.conn.RemoteAddr()
}

func (uucp *udpUniClientProxy) SetDeadline(t time.Time) error {
	return uucp.conn.SetDeadline(t)
}

func (uucp *udpUniClientProxy) SetReadDeadline(t time.Time) error {
	return uucp.conn.SetReadDeadline(t)
}

func (uucp *udpUniClientProxy) SetWriteDeadline(t time.Time) error {
	return uucp.conn.SetWriteDeadline(t)
}
