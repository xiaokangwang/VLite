package tun

import (
	"github.com/FlowerWrong/water"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/xiaokangwang/VLite/interfaces"
	"log"
	"net"
)

func NewTunToUDPLink(TxToTun chan interfaces.UDPPacket,
	RxFromTun chan interfaces.UDPPacket,
	tunInterface water.Interface) (*TunToUDPLink, error) {
	l := &TunToUDPLink{LocalTxToTun: TxToTun, LocalRxFromTun: RxFromTun, tunInterface: tunInterface}
	go l.InjectionLink()
	return l, nil
}

type TunToUDPLink struct {
	LocalTxToTun   chan interfaces.UDPPacket
	LocalRxFromTun chan interfaces.UDPPacket
	tunInterface   water.Interface
}

func (ttl *TunToUDPLink) Read(udpmsg []byte) (int, error) {
	for {
		UDPacket :=
			interfaces.UDPPacket{
				Source:  &net.UDPAddr{},
				Dest:    &net.UDPAddr{},
				Payload: nil,
			}
		var decodedCounter int
		n, err := ttl.tunInterface.Read(udpmsg[:])
		if err != nil {
			return 0, err
		}
		packet := gopacket.NewPacket(udpmsg[:n], layers.LayerTypeIPv4, gopacket.DecodeOptions{
			Lazy:                     true,
			NoCopy:                   false,
			SkipDecodeRecovery:       false,
			DecodeStreamsAsDatagrams: false,
		})

		ipv4 := packet.Layer(layers.LayerTypeIPv4)

		if ipv4 != nil {
			ipv4l := ipv4.(*layers.IPv4)
			UDPacket.Source.IP = ipv4l.SrcIP
			UDPacket.Dest.IP = ipv4l.DstIP
			decodedCounter++
		}

		udp := packet.Layer(layers.LayerTypeUDP)

		if udp != nil {
			udpl := udp.(*layers.UDP)
			UDPacket.Source.Port = int(udpl.SrcPort)
			UDPacket.Dest.Port = int(udpl.DstPort)
			UDPacket.Payload = udpl.Payload
			decodedCounter++
		}

		if decodedCounter == 2 {
			ttl.LocalRxFromTun <- UDPacket
			continue
		}

		return n, err
	}

}

func (ttl *TunToUDPLink) Write(p []byte) (int, error) {
	return ttl.tunInterface.Write(p)
}

func (ttl *TunToUDPLink) InjectionLink() {
	for {
		packet := <-ttl.LocalTxToTun
		buffer := gopacket.NewSerializeBuffer()

		ipv4l := &layers.IPv4{SrcIP: packet.Source.IP, DstIP: packet.Dest.IP, Protocol: layers.IPProtocolUDP, Version:4,TTL:64}
		udpl := &layers.UDP{SrcPort: layers.UDPPort(packet.Source.Port), DstPort: layers.UDPPort(packet.Dest.Port)}
		err2 := udpl.SetNetworkLayerForChecksum(ipv4l)
		if err2 != nil {
			log.Println(err2)
			continue
		}
		err := gopacket.SerializeLayers(buffer, gopacket.SerializeOptions{
			FixLengths:       true,
			ComputeChecksums: true,
		},
			ipv4l,
			udpl,
			gopacket.Payload(packet.Payload),
		)
		if err != nil {
			log.Println(err)
			continue
		}

		_, err = ttl.tunInterface.Write(buffer.Bytes())
		if err != nil {
			log.Println(err)
		}
	}
}

func (ttl *TunToUDPLink) Close() error {
	return ttl.tunInterface.Close()
}
