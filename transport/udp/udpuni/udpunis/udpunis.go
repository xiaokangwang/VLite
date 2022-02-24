package udpunis

import (
	"context"
	"fmt"
	"github.com/xiaokangwang/VLite/interfaces"
	"github.com/xiaokangwang/VLite/transport"
	"github.com/xiaokangwang/VLite/transport/http/headerHolder"
	"github.com/xiaokangwang/VLite/transport/udp/packetarmor"
	"net"
	"reflect"
	"time"
)

func NewUdpUniServer(password string,
	ctx context.Context, upper transport.UnderlayTransportListener) *UdpUniServer {
	return &UdpUniServer{
		password: password,
		ctx:      ctx,
		hh:       headerHolder.NewHttpHeaderHolderProcessor2(password, "UdpUniSecret"),
		armor:    packetarmor.NewPacketArmor(password, "UdpUniPacketArmor", false),
		upper:    upper,
	}
}

type UdpUniServer struct {
	password string
	ctx      context.Context

	hh    *headerHolder.HttpHeaderHolderProcessor
	armor *packetarmor.PacketArmor

	upper transport.UnderlayTransportListener
}

func (uus *UdpUniServer) Connection(conn net.Conn, ctx context.Context) context.Context {

	InitialDataVal := ctx.Value(interfaces.ExtraOptionsUDPInitialData)
	InitialData := InitialDataVal.(*interfaces.ExtraOptionsUDPInitialDataValue).Data

	usePacketArmor := false
	packetArmorVal := ctx.Value(interfaces.ExtraOptionsUsePacketArmor)
	if packetArmorVal != nil {
		packetArmorVal := packetArmorVal.(*interfaces.ExtraOptionsUsePacketArmorValue)
		if packetArmorVal.UsePacketArmor {
			usePacketArmor = true
		}
	}
	initdata := string(InitialData)
	if usePacketArmor {
		data, err := uus.armor.Unpack(InitialData)
		if err != nil {
			fmt.Println("Unable to decrypt initial data armor")
			return nil
		}
		initdata = string(data)
	}

	phv := uus.hh.Open(initdata)
	if phv == nil {
		fmt.Println("Unable to decrypt initial data")
		return nil
	}

	univ := &interfaces.ExtraOptionsUniConnAttribValue{
		ID:   phv.ConnID[:],
		Rand: phv.Rand[:],
		Iter: phv.ConnIter,
	}

	ctx = context.WithValue(ctx, interfaces.ExtraOptionsUniConnAttrib, univ)

	/*
		if usePacketArmor {
			data, err := uus.armor.Pack([]byte(initdata), len(InitialData))
			if err != nil {
				fmt.Println("Unable to reply initial data")
				return nil
			}
			conn.Write(data)
		} else {
			conn.Write([]byte(initdata))
		}*/

	return uus.upper.Connection(&udpUniClientProxy{conn: conn, initBuf: []byte(initdata),
		armor: uus.armor, useArmoredPacket: usePacketArmor}, ctx)

}

func (uus *UdpUniServer) AsUnderlayTransportListener() transport.UnderlayTransportListener {
	return uus
}

type udpUniClientProxy struct {
	initBuf          []byte
	conn             net.Conn
	useArmoredPacket bool
	armor            *packetarmor.PacketArmor

	recvCount int
}

func (uucp *udpUniClientProxy) Read(b []byte) (n int, err error) {
	for {
		n, err = uucp.conn.Read(b)
		uucp.recvCount++
		if uucp.recvCount <= 300 {
			if uucp.useArmoredPacket {
				data, err := uucp.armor.Unpack(b[:n])
				if err != nil {
					return n, nil
				}
				if reflect.DeepEqual(data, uucp.initBuf) {
					data, err := uucp.armor.Pack(data, n)
					if err != nil {
						fmt.Println(err)
						continue
					}
					uucp.Write(data)
					continue
				}
			} else {
				if reflect.DeepEqual(b[:n], uucp.initBuf) {
					uucp.Write(uucp.initBuf)
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
