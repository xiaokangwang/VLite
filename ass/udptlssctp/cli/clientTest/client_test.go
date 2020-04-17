package clientTest

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/xiaokangwang/VLite/ass/udptlssctp"
	"github.com/xiaokangwang/VLite/interfaces"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"
)


func TestInitializeNil(t *testing.T) {

	cli := `run github.com/xiaokangwang/VLite/ass/udptlssctp/cli/server -Password pw -Address 127.0.0.1:8811`
	gopath, err := exec.LookPath("go")
	if err != nil {
		panic(err)
	}
	command := exec.Command(gopath, strings.Split(cli, " ")...)
	command.Stderr = os.Stderr
	command.Stdout = os.Stdout

	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	err = command.Start()
	if err != nil {
		panic(err)
	}

	<-time.NewTimer(time.Second * 1).C


	rootContext := context.Background()
	ccontext, cancel := context.WithCancel(rootContext)

	var password string
	var address string

	password = "pw"

	address = "127.0.0.1:8811"

	uc := udptlssctp.NewUdptlsSctpClientDirect(address, password, ccontext)
	uc.Up()

	TunnelRxFromTun := uc.TunnelRxFromTun

	TunnelTxToTun := uc.TunnelTxToTun

	_ = TunnelRxFromTun

	_ = TunnelTxToTun

	cancel()

	err = syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
	if err != nil {
		panic(err)
	}
	err = command.Wait()
	if err != nil && !strings.Contains(err.Error(),"killed") {
		panic(err)
	}
}

func TestInitializeWithDataCopy(t *testing.T) {

	cli := `run github.com/xiaokangwang/VLite/ass/udptlssctp/cli/server -Password pw -Address 127.0.0.1:8811`
	gopath, err := exec.LookPath("go")
	if err != nil {
		panic(err)
	}
	command := exec.Command(gopath, strings.Split(cli, " ")...)
	command.Stderr = os.Stderr
	command.Stdout = os.Stdout

	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	err = command.Start()
	if err != nil {
		panic(err)
	}
	<-time.NewTimer(time.Second * 1).C

	rootContext := context.Background()
	ccontext, cancel := context.WithCancel(rootContext)

	var password string
	var address string

	password = "pw"

	address = "127.0.0.1:8811"

	uc := udptlssctp.NewUdptlsSctpClientDirect(address, password, ccontext)
	uc.Up()

	TunnelRxFromTun := uc.TunnelRxFromTun

	TunnelTxToTun := uc.TunnelTxToTun

	_ = TunnelRxFromTun

	_ = TunnelTxToTun

	l, err := net.ListenUDP("udp4", &net.UDPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 18999,
		Zone: "",
	})

	if err != nil {
		panic(err)
	}

	go func() {
		var buf [1600]byte
		for {
			//l.SetReadDeadline(time.Now().Add(10 * time.Second))

			n, addr, err := l.ReadFrom(buf[:])
			if err != nil {
				log.Println(err)
			}

			_, err = l.WriteTo(buf[:n], addr)
			if err != nil {
				log.Println(err)
			}

			if ccontext.Err() != nil {
				return
			}
		}
	}()

	TxPacket := interfaces.UDPPacket{
		Source: &net.UDPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 18990,
			Zone: "",
		},
		Dest: &net.UDPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 18999,
			Zone: "",
		},
		Payload: []byte("Test"),
	}

	TunnelRxFromTun <- TxPacket

	RxPacket := <-TunnelTxToTun

	assert.Equal(t, TxPacket.Payload, RxPacket.Payload)
	assert.Equal(t, TxPacket.Source.IP.To4(), RxPacket.Dest.IP.To4())
	assert.Equal(t, TxPacket.Source.Port, RxPacket.Dest.Port)
	assert.Equal(t, TxPacket.Dest.IP.To4(), RxPacket.Source.IP.To4())
	assert.Equal(t, TxPacket.Dest.Port, RxPacket.Source.Port)

	<-time.NewTimer(time.Second * 1).C
	cancel()
	l.Close()
	<-time.NewTimer(time.Second * 4).C

	cancel()

	err = syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
	if err != nil {
		panic(err)
	}
	err = command.Wait()
	if err != nil && !strings.Contains(err.Error(),"killed") {
		panic(err)
	}
}

func TestInitializeWithDataCopy2(t *testing.T) {
	cli := `run github.com/xiaokangwang/VLite/ass/udptlssctp/cli/server -Password pw -Address 127.0.0.1:8811`
	gopath, err := exec.LookPath("go")
	if err != nil {
		panic(err)
	}
	command := exec.Command(gopath, strings.Split(cli, " ")...)
	command.Stderr = os.Stderr
	command.Stdout = os.Stdout

	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	err = command.Start()
	if err != nil {
		panic(err)
	}
	<-time.NewTimer(time.Second * 1).C

	rootContext := context.Background()
	ccontext, cancel := context.WithCancel(rootContext)

	var password string
	var address string

	password = "pw"

	address = "127.0.0.1:8811"

	uc := udptlssctp.NewUdptlsSctpClientDirect(address, password, ccontext)
	uc.Up()

	TunnelRxFromTun := uc.TunnelRxFromTun

	TunnelTxToTun := uc.TunnelTxToTun

	_ = TunnelRxFromTun

	_ = TunnelTxToTun

	l, err := net.ListenUDP("udp4", &net.UDPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 18999,
		Zone: "",
	})

	if err != nil {
		panic(err)
	}

	go func() {
		var buf [1600]byte
		for {
			//l.SetReadDeadline(time.Now().Add(10 * time.Second))

			n, addr, err := l.ReadFrom(buf[:])
			if err != nil {
				log.Println(err)
			}

			<-time.NewTimer(time.Second * 1).C

			_, err = l.WriteTo(buf[:n], addr)
			if err != nil {
				log.Println(err)
			}

			if ccontext.Err() != nil {
				return
			}
		}
	}()

	TxPacket := interfaces.UDPPacket{
		Source: &net.UDPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 18990,
			Zone: "",
		},
		Dest: &net.UDPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 18999,
			Zone: "",
		},
		Payload: []byte("Test"),
	}

	TunnelRxFromTun <- TxPacket

	RxPacket := <-TunnelTxToTun

	assert.Equal(t, TxPacket.Payload, RxPacket.Payload)
	assert.Equal(t, TxPacket.Source.IP.To4(), RxPacket.Dest.IP.To4())
	assert.Equal(t, TxPacket.Source.Port, RxPacket.Dest.Port)
	assert.Equal(t, TxPacket.Dest.IP.To4(), RxPacket.Source.IP.To4())
	assert.Equal(t, TxPacket.Dest.Port, RxPacket.Source.Port)

	<-time.NewTimer(time.Second * 1).C
	cancel()
	l.Close()

	<-time.NewTimer(time.Second * 4).C

	err = syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
	if err != nil {
		panic(err)
	}
	err = command.Wait()
	if err != nil && !strings.Contains(err.Error(),"killed") {
		panic(err)
	}
}

func TestInitializeWithDataCopy3(t *testing.T) {
	cli := `run github.com/xiaokangwang/VLite/ass/udptlssctp/cli/server -Password pw -Address 127.0.0.1:8811`
	gopath, err := exec.LookPath("go")
	if err != nil {
		panic(err)
	}
	command := exec.Command(gopath, strings.Split(cli, " ")...)
	command.Stderr = os.Stderr
	command.Stdout = os.Stdout

	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	err = command.Start()
	if err != nil {
		panic(err)
	}
	<-time.NewTimer(time.Second * 1).C

	rootContext := context.Background()
	ccontext, cancel := context.WithCancel(rootContext)

	var password string
	var address string

	password = "pw"

	address = "127.0.0.1:8811"

	uc := udptlssctp.NewUdptlsSctpClientDirect(address, password, ccontext)
	uc.Up()

	TunnelRxFromTun := uc.TunnelRxFromTun

	TunnelTxToTun := uc.TunnelTxToTun

	_ = TunnelRxFromTun

	_ = TunnelTxToTun

	l, err := net.ListenUDP("udp4", &net.UDPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 18999,
		Zone: "",
	})

	if err != nil {
		panic(err)
	}

	go func() {
		var buf [1600]byte
		for {
			//l.SetReadDeadline(time.Now().Add(10 * time.Second))

			n, addr, err := l.ReadFrom(buf[:])
			if err != nil {
				log.Println(err)
			}

			<-time.NewTimer(time.Second * 1).C

			_, err = l.WriteTo(buf[:n], addr)
			if err != nil {
				log.Println(err)
			}

			if ccontext.Err() != nil {
				return
			}
		}
	}()

	TxPacket := interfaces.UDPPacket{
		Source: &net.UDPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 18990,
			Zone: "",
		},
		Dest: &net.UDPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 18999,
			Zone: "",
		},
		Payload: []byte("Test"),
	}

	for i := 0; i <= 10; i++ {
		TunnelRxFromTun <- TxPacket

		RxPacket := <-TunnelTxToTun

		assert.Equal(t, TxPacket.Payload, RxPacket.Payload)
		assert.Equal(t, TxPacket.Source.IP.To4(), RxPacket.Dest.IP.To4())
		assert.Equal(t, TxPacket.Source.Port, RxPacket.Dest.Port)
		assert.Equal(t, TxPacket.Dest.IP.To4(), RxPacket.Source.IP.To4())
		assert.Equal(t, TxPacket.Dest.Port, RxPacket.Source.Port)
	}

	<-time.NewTimer(time.Second * 1).C
	cancel()
	l.Close()

	<-time.NewTimer(time.Second * 4).C

	err = syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
	if err != nil {
		panic(err)
	}
	err = command.Wait()
	if err != nil && !strings.Contains(err.Error(),"killed") {
		panic(err)
	}
}

func TestInitializeWithDataCopy4(t *testing.T) {
	cli := `run github.com/xiaokangwang/VLite/ass/udptlssctp/cli/server -Password pw -Address 127.0.0.1:8811`
	gopath, err := exec.LookPath("go")
	if err != nil {
		panic(err)
	}
	command := exec.Command(gopath, strings.Split(cli, " ")...)
	command.Stderr = os.Stderr
	command.Stdout = os.Stdout

	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	err = command.Start()
	if err != nil {
		panic(err)
	}
	<-time.NewTimer(time.Second * 1).C


	rootContext := context.Background()
	ccontext, cancel := context.WithCancel(rootContext)

	var password string
	var address string

	password = "pw"

	address = "127.0.0.1:8811"

	uc := udptlssctp.NewUdptlsSctpClientDirect(address, password, ccontext)
	uc.Up()

	TunnelRxFromTun := uc.TunnelRxFromTun

	TunnelTxToTun := uc.TunnelTxToTun

	_ = TunnelRxFromTun

	_ = TunnelTxToTun

	l, err := net.ListenUDP("udp4", &net.UDPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 18999,
		Zone: "",
	})

	if err != nil {
		panic(err)
	}

	go func() {
		var buf [1600]byte
		for {
			//l.SetReadDeadline(time.Now().Add(10 * time.Second))

			n, addr, err := l.ReadFrom(buf[:])
			if err != nil {
				log.Println(err)
			}

			//<- time.NewTimer(time.Second*1).C

			_, err = l.WriteTo(buf[:n], addr)
			if err != nil {
				log.Println(err)
			}

			if ccontext.Err() != nil {
				return
			}
		}
	}()

	TxPacket := interfaces.UDPPacket{
		Source: &net.UDPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 18990,
			Zone: "",
		},
		Dest: &net.UDPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 18999,
			Zone: "",
		},
		Payload: []byte("Test"),
	}

	for i := 0; i <= 10; i++ {
		TunnelRxFromTun <- TxPacket

		RxPacket := <-TunnelTxToTun

		assert.Equal(t, TxPacket.Payload, RxPacket.Payload)
		assert.Equal(t, TxPacket.Source.IP.To4(), RxPacket.Dest.IP.To4())
		assert.Equal(t, TxPacket.Source.Port, RxPacket.Dest.Port)
		assert.Equal(t, TxPacket.Dest.IP.To4(), RxPacket.Source.IP.To4())
		assert.Equal(t, TxPacket.Dest.Port, RxPacket.Source.Port)
	}

	<-time.NewTimer(time.Second * 1).C
	cancel()
	l.Close()

	<-time.NewTimer(time.Second * 4).C

	err = syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
	if err != nil {
		panic(err)
	}
	err = command.Wait()
	if err != nil && !strings.Contains(err.Error(),"killed") {
		panic(err)
	}
}

func TestInitializeWithDataCopy5(t *testing.T) {
	cli := `run github.com/xiaokangwang/VLite/ass/udptlssctp/cli/server -Password pw -Address 127.0.0.1:8811`
	gopath, err := exec.LookPath("go")
	if err != nil {
		panic(err)
	}
	command := exec.Command(gopath, strings.Split(cli, " ")...)
	command.Stderr = os.Stderr
	command.Stdout = os.Stdout

	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	err = command.Start()
	if err != nil {
		panic(err)
	}
	<-time.NewTimer(time.Second * 1).C

	rootContext := context.Background()
	ccontext, cancel := context.WithCancel(rootContext)

	var password string
	var address string

	password = "pw"

	address = "127.0.0.1:8811"

	uc := udptlssctp.NewUdptlsSctpClientDirect(address, password, ccontext)
	uc.Up()

	TunnelRxFromTun := uc.TunnelRxFromTun

	TunnelTxToTun := uc.TunnelTxToTun

	_ = TunnelRxFromTun

	_ = TunnelTxToTun

	l, err := net.ListenUDP("udp4", &net.UDPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 18999,
		Zone: "",
	})

	if err != nil {
		panic(err)
	}

	go func() {
		var buf [1600]byte
		for {
			//l.SetReadDeadline(time.Now().Add(10 * time.Second))

			n, addr, err := l.ReadFrom(buf[:])
			if err != nil {
				log.Println(err)
			}

			//<- time.NewTimer(time.Second*1).C

			_, err = l.WriteTo(buf[:n], addr)
			if err != nil {
				log.Println(err)
			}

			if ccontext.Err() != nil {
				return
			}
		}
	}()

	TxPacket := interfaces.UDPPacket{
		Source: &net.UDPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 18000,
			Zone: "",
		},
		Dest: &net.UDPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 18999,
			Zone: "",
		},
		Payload: []byte("Test"),
	}

	for i := 0; i <= 10; i++ {
		TxPacket.Source.Port += 1
		TunnelRxFromTun <- TxPacket

		RxPacket := <-TunnelTxToTun

		assert.Equal(t, TxPacket.Payload, RxPacket.Payload)
		assert.Equal(t, TxPacket.Source.IP.To4(), RxPacket.Dest.IP.To4())
		assert.Equal(t, TxPacket.Source.Port, RxPacket.Dest.Port)
		assert.Equal(t, TxPacket.Dest.IP.To4(), RxPacket.Source.IP.To4())
		assert.Equal(t, TxPacket.Dest.Port, RxPacket.Source.Port)
	}

	<-time.NewTimer(time.Second * 1).C
	cancel()
	l.Close()

	<-time.NewTimer(time.Second * 4).C

	err = syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
	if err != nil {
		panic(err)
	}
	err = command.Wait()
	if err != nil && !strings.Contains(err.Error(),"killed") {
		panic(err)
	}
}

func TestInitializeWithDataCopy6(t *testing.T) {
	cli := `run github.com/xiaokangwang/VLite/ass/udptlssctp/cli/server -Password pw -Address 127.0.0.1:8811`
	gopath, err := exec.LookPath("go")
	if err != nil {
		panic(err)
	}
	command := exec.Command(gopath, strings.Split(cli, " ")...)
	command.Stderr = os.Stderr
	command.Stdout = os.Stdout

	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	err = command.Start()
	if err != nil {
		panic(err)
	}
	<-time.NewTimer(time.Second * 1).C


	rootContext := context.Background()
	ccontext, cancel := context.WithCancel(rootContext)

	var password string
	var address string

	password = "pw"

	address = "127.0.0.1:8811"

	uc := udptlssctp.NewUdptlsSctpClientDirect(address, password, ccontext)
	uc.Up()

	TunnelRxFromTun := uc.TunnelRxFromTun

	TunnelTxToTun := uc.TunnelTxToTun

	_ = TunnelRxFromTun

	_ = TunnelTxToTun

	l, err := net.ListenUDP("udp4", &net.UDPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 18999,
		Zone: "",
	})

	if err != nil {
		panic(err)
	}

	go func() {
		var buf [1600]byte
		for {
			//l.SetReadDeadline(time.Now().Add(10 * time.Second))

			n, addr, err := l.ReadFrom(buf[:])
			if err != nil {
				log.Println(err)
			}

			//<- time.NewTimer(time.Second*1).C

			_, err = l.WriteTo(buf[:n], addr)
			if err != nil {
				log.Println(err)
			}

			if ccontext.Err() != nil {
				return
			}
		}
	}()

	TxPacket := interfaces.UDPPacket{
		Source: &net.UDPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 18000,
			Zone: "",
		},
		Dest: &net.UDPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 18999,
			Zone: "",
		},
		Payload: []byte("Test"),
	}

	for j := 0; j <= 10; j++ {
		for i := 0; i <= 10; i++ {
			TxPacket.Source.Port = 19000 + i
			TunnelRxFromTun <- TxPacket

			RxPacket := <-TunnelTxToTun

			assert.Equal(t, TxPacket.Payload, RxPacket.Payload)
			assert.Equal(t, TxPacket.Source.IP.To4(), RxPacket.Dest.IP.To4())
			assert.Equal(t, TxPacket.Source.Port, RxPacket.Dest.Port)
			assert.Equal(t, TxPacket.Dest.IP.To4(), RxPacket.Source.IP.To4())
			assert.Equal(t, TxPacket.Dest.Port, RxPacket.Source.Port)
		}
	}

	<-time.NewTimer(time.Second * 1).C
	cancel()
	l.Close()

	<-time.NewTimer(time.Second * 4).C

	err = syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
	if err != nil {
		panic(err)
	}
	err = command.Wait()
	if err != nil && !strings.Contains(err.Error(),"killed") {
		panic(err)
	}
}

func TestInitializeWithDataCopy6WithShortTimeout(t *testing.T) {
	cli := `run github.com/xiaokangwang/VLite/ass/udptlssctp/cli/server -Password pw -Address 127.0.0.1:8811`
	gopath, err := exec.LookPath("go")
	if err != nil {
		panic(err)
	}
	command := exec.Command(gopath, strings.Split(cli, " ")...)
	command.Stderr = os.Stderr
	command.Stdout = os.Stdout

	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	err = command.Start()
	if err != nil {
		panic(err)
	}
	<-time.NewTimer(time.Second * 1).C


	rootContext := context.Background()
	ccontext, cancel := context.WithCancel(rootContext)

	var password string
	var address string

	password = "pw"

	address = "127.0.0.1:8811"

	uc := udptlssctp.NewUdptlsSctpClientDirect(address, password, ccontext)
	uc.Up()

	TunnelRxFromTun := uc.TunnelRxFromTun

	TunnelTxToTun := uc.TunnelTxToTun

	_ = TunnelRxFromTun

	_ = TunnelTxToTun

	l, err := net.ListenUDP("udp4", &net.UDPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 18999,
		Zone: "",
	})

	if err != nil {
		panic(err)
	}

	go func() {
		var buf [1600]byte
		for {
			//l.SetReadDeadline(time.Now().Add(10 * time.Second))

			n, addr, err := l.ReadFrom(buf[:])
			if err != nil {
				log.Println(err)
			}

			//<- time.NewTimer(time.Second*1).C

			_, err = l.WriteTo(buf[:n], addr)
			if err != nil {
				log.Println(err)
			}

			if ccontext.Err() != nil {
				return
			}
		}
	}()

	TxPacket := interfaces.UDPPacket{
		Source: &net.UDPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 18000,
			Zone: "",
		},
		Dest: &net.UDPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 18999,
			Zone: "",
		},
		Payload: []byte("Test"),
	}

	for j := 0; j <= 10; j++ {
		for i := 0; i <= 10; i++ {
			TxPacket.Source.Port = 19000 + i
			TunnelRxFromTun <- TxPacket

			RxPacket := <-TunnelTxToTun

			assert.Equal(t, TxPacket.Payload, RxPacket.Payload)
			assert.Equal(t, TxPacket.Source.IP.To4(), RxPacket.Dest.IP.To4())
			assert.Equal(t, TxPacket.Source.Port, RxPacket.Dest.Port)
			assert.Equal(t, TxPacket.Dest.IP.To4(), RxPacket.Source.IP.To4())
			assert.Equal(t, TxPacket.Dest.Port, RxPacket.Source.Port)
		}
		<-time.NewTimer(time.Second * 4).C
	}

	<-time.NewTimer(time.Second * 1).C
	cancel()
	l.Close()

	<-time.NewTimer(time.Second * 4).C

	err = syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
	if err != nil {
		panic(err)
	}
	err = command.Wait()
	if err != nil && !strings.Contains(err.Error(),"killed") {
		panic(err)
	}
}

func TestInitializeWithDataCopyFullCone(t *testing.T) {
	cli := `run github.com/xiaokangwang/VLite/ass/udptlssctp/cli/server -Password pw -Address 127.0.0.1:8811`
	gopath, err := exec.LookPath("go")
	if err != nil {
		panic(err)
	}
	command := exec.Command(gopath, strings.Split(cli, " ")...)
	command.Stderr = os.Stderr
	command.Stdout = os.Stdout

	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	err = command.Start()
	if err != nil {
		panic(err)
	}
	<-time.NewTimer(time.Second * 1).C

	rootContext := context.Background()
	ccontext, cancel := context.WithCancel(rootContext)

	var password string
	var address string

	password = "pw"

	address = "127.0.0.1:8811"

	uc := udptlssctp.NewUdptlsSctpClientDirect(address, password, ccontext)
	uc.Up()

	TunnelRxFromTun := uc.TunnelRxFromTun

	TunnelTxToTun := uc.TunnelTxToTun

	_ = TunnelRxFromTun

	_ = TunnelTxToTun

	l, err := net.ListenUDP("udp4", &net.UDPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 18999,
		Zone: "",
	})

	l2, err := net.ListenUDP("udp4", &net.UDPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 19999,
		Zone: "",
	})

	if err != nil {
		panic(err)
	}

	go func() {
		var buf [1600]byte
		for {
			//l.SetReadDeadline(time.Now().Add(10 * time.Second))

			n, addr, err := l.ReadFrom(buf[:])
			if err != nil {
				log.Println(err)
			}

			<-time.NewTimer(time.Second * 1).C

			_, err = l.WriteTo(buf[:n], addr)
			if err != nil {
				log.Println(err)
			}

			if ccontext.Err() != nil {
				return
			}
			<-time.NewTimer(time.Second * 1).C
			_, err = l2.WriteTo(buf[:n], addr)
			if err != nil {
				log.Println(err)
			}

			if ccontext.Err() != nil {
				return
			}
		}
	}()

	TxPacket := interfaces.UDPPacket{
		Source: &net.UDPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 18990,
			Zone: "",
		},
		Dest: &net.UDPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 18999,
			Zone: "",
		},
		Payload: []byte("Test"),
	}

	TxPacket2 := interfaces.UDPPacket{
		Source: &net.UDPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 18990,
			Zone: "",
		},
		Dest: &net.UDPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 19999,
			Zone: "",
		},
		Payload: []byte("Test"),
	}

	TunnelRxFromTun <- TxPacket

	RxPacket := <-TunnelTxToTun

	assert.Equal(t, TxPacket.Payload, RxPacket.Payload)
	assert.Equal(t, TxPacket.Source.IP.To4(), RxPacket.Dest.IP.To4())
	assert.Equal(t, TxPacket.Source.Port, RxPacket.Dest.Port)
	assert.Equal(t, TxPacket.Dest.IP.To4(), RxPacket.Source.IP.To4())
	assert.Equal(t, TxPacket.Dest.Port, RxPacket.Source.Port)

	RxPacket2 := <-TunnelTxToTun

	assert.Equal(t, TxPacket2.Payload, RxPacket2.Payload)
	assert.Equal(t, TxPacket2.Source.IP.To4(), RxPacket2.Dest.IP.To4())
	assert.Equal(t, TxPacket2.Source.Port, RxPacket2.Dest.Port)
	assert.Equal(t, TxPacket2.Dest.IP.To4(), RxPacket2.Source.IP.To4())
	assert.Equal(t, TxPacket2.Dest.Port, RxPacket2.Source.Port)

	<-time.NewTimer(time.Second * 1).C
	cancel()
	l.Close()

	<-time.NewTimer(time.Second * 4).C

	err = syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
	if err != nil {
		panic(err)
	}
	err = command.Wait()
	if err != nil && !strings.Contains(err.Error(),"killed") {
		panic(err)
	}
}

func TestInitializeWithDataCopy5_Plus(t *testing.T) {
	cli := `run github.com/xiaokangwang/VLite/ass/udptlssctp/cli/server -Password pw -Address 127.0.0.1:8811`
	gopath, err := exec.LookPath("go")
	if err != nil {
		panic(err)
	}
	command := exec.Command(gopath, strings.Split(cli, " ")...)
	command.Stderr = os.Stderr
	command.Stdout = os.Stdout

	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	err = command.Start()
	if err != nil {
		panic(err)
	}
	<-time.NewTimer(time.Second * 1).C

	rootContext := context.Background()
	ccontext, cancel := context.WithCancel(rootContext)

	var password string
	var address string

	password = "pw"

	address = "127.0.0.1:8811"

	uc := udptlssctp.NewUdptlsSctpClientDirect(address, password, ccontext)
	uc.Up()

	TunnelRxFromTun := uc.TunnelRxFromTun

	TunnelTxToTun := uc.TunnelTxToTun

	_ = TunnelRxFromTun

	_ = TunnelTxToTun

	l, err := net.ListenUDP("udp4", &net.UDPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 18999,
		Zone: "",
	})

	if err != nil {
		panic(err)
	}

	go func() {
		var buf [1600]byte
		for {
			//l.SetReadDeadline(time.Now().Add(10 * time.Second))

			n, addr, err := l.ReadFrom(buf[:])
			if err != nil {
				log.Println(err)
			}

			//<- time.NewTimer(time.Second*1).C

			_, err = l.WriteTo(buf[:n], addr)
			if err != nil {
				log.Println(err)
			}

			if ccontext.Err() != nil {
				return
			}
		}
	}()

	for j := 0; j <= 10; j++ {
		for i := 0; i <= 800; i++ {
			TxPacketL := interfaces.UDPPacket{
				Source: &net.UDPAddr{
					IP:   net.ParseIP("127.0.0.1"),
					Port: 11000,
					Zone: "",
				},
				Dest: &net.UDPAddr{
					IP:   net.ParseIP("127.0.0.1"),
					Port: 18999,
					Zone: "",
				},
				Payload: []byte("Test"),
			}
			TxPacketL.Source.Port += i
			TunnelRxFromTun <- TxPacketL

			RxPacket := <-TunnelTxToTun

			assert.Equal(t, TxPacketL.Payload, RxPacket.Payload)
			assert.Equal(t, TxPacketL.Source.IP.To4(), RxPacket.Dest.IP.To4())
			assert.Equal(t, TxPacketL.Source.Port, RxPacket.Dest.Port)
			assert.Equal(t, TxPacketL.Dest.IP.To4(), RxPacket.Source.IP.To4())
			assert.Equal(t, TxPacketL.Dest.Port, RxPacket.Source.Port)
			<-time.NewTimer(time.Nanosecond * 1).C
		}
	}

	<-time.NewTimer(time.Second * 4).C

	for j := 0; j <= 10; j++ {
		for i := 0; i <= 800; i++ {
			TxPacketL := interfaces.UDPPacket{
				Source: &net.UDPAddr{
					IP:   net.ParseIP("127.0.0.1"),
					Port: 11000,
					Zone: "",
				},
				Dest: &net.UDPAddr{
					IP:   net.ParseIP("127.0.0.1"),
					Port: 18999,
					Zone: "",
				},
				Payload: []byte("Test"),
			}
			TxPacketL.Source.Port += i
			TunnelRxFromTun <- TxPacketL

			RxPacket := <-TunnelTxToTun

			assert.Equal(t, TxPacketL.Payload, RxPacket.Payload)
			assert.Equal(t, TxPacketL.Source.IP.To4(), RxPacket.Dest.IP.To4())
			assert.Equal(t, TxPacketL.Source.Port, RxPacket.Dest.Port)
			assert.Equal(t, TxPacketL.Dest.IP.To4(), RxPacket.Source.IP.To4())
			assert.Equal(t, TxPacketL.Dest.Port, RxPacket.Source.Port)
			<-time.NewTimer(time.Nanosecond * 1).C
		}
	}

	<-time.NewTimer(time.Second * 4).C
	cancel()
	l.Close()

	<-time.NewTimer(time.Second * 4).C

	err = syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
	if err != nil {
		panic(err)
	}
	err = command.Wait()
	if err != nil && !strings.Contains(err.Error(),"killed") {
		panic(err)
	}
}
