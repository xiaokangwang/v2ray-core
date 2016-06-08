package kcptun

import (
	"io"
	"sync"

	"github.com/v2ray/v2ray-core/app"
	"github.com/v2ray/v2ray-core/app/dispatcher"
	v2io "github.com/v2ray/v2ray-core/common/io"
	"github.com/v2ray/v2ray-core/common/log"
	v2net "github.com/v2ray/v2ray-core/common/net"
	"github.com/v2ray/v2ray-core/proxy"
	"github.com/v2ray/v2ray-core/proxy/internal"
	"github.com/xtaci/kcp-go"
)

func NewKCPInboundConnection(config *Config, space app.Space) *KCPInboundConnection {
	c := &KCPInboundConnection{
		address: config.Address,
		port:    config.Port,
	}

	space.InitializeApplication(func() error {
		if !space.HasApp(dispatcher.APP_ID) {
			log.Error("Dokodemo: Dispatcher is not found in the space.")
			return app.ErrorMissingApplication
		}
		c.packetDispatcher = space.GetApp(dispatcher.APP_ID).(dispatcher.PacketDispatcher)
		return nil
	})
	c.config = config
	return c
}

type KCPInboundConnection struct {
	accepting        bool
	config           *Config
	listeningPort    v2net.Port
	listeningAddress v2net.Address
	address          string
	port             string
	packetDispatcher dispatcher.PacketDispatcher
}

func (kic *KCPInboundConnection) Close() {
	kic.accepting = false
}
func (kic *KCPInboundConnection) ReadAndDecodeHeader(conn io.Reader) v2net.Destination {
	return decodeDestJSON(conn)
}
func (kic *KCPInboundConnection) HandleIncomingConn(conn *kcp.UDPSession) {
	dest := kic.ReadAndDecodeHeader(conn)

	defer conn.Close()

	ray := kic.packetDispatcher.DispatchToOutbound(dest)
	defer ray.InboundOutput().Release()

	var inputFinish, outputFinish sync.Mutex
	inputFinish.Lock()
	outputFinish.Lock()

	reader := conn

	go func() {
		v2reader := v2io.NewAdaptiveReader(reader)
		defer v2reader.Release()

		v2io.Pipe(v2reader, ray.InboundInput())
		inputFinish.Unlock()
		ray.InboundInput().Close()
	}()

	go func() {
		v2writer := v2io.NewAdaptiveWriter(conn)
		defer v2writer.Release()

		v2io.Pipe(ray.InboundOutput(), v2writer)
		outputFinish.Unlock()
	}()

	outputFinish.Lock()
	inputFinish.Lock()
}

func (kic *KCPInboundConnection) GoServerConn(us *kcp.Listener) {
	for {
		conn, err := us.Accept()

		nodelay, interval, resend, nc := 0, 40, 0, 0
		if kic.config.Mode != "manual" {
			switch kic.config.Mode {
			case "normal":
				nodelay, interval, resend, nc = 0, 30, 2, 1
			case "fast":
				nodelay, interval, resend, nc = 0, 20, 2, 1
			case "fast2":
				nodelay, interval, resend, nc = 1, 20, 2, 1
			case "fast3":
				nodelay, interval, resend, nc = 1, 10, 2, 1
			}
		} else {
			log.Error("kcptvn: Accepted Unsuccessfully: Manual mode is not supported.(yet!)")
			return
		}

		conn.SetNoDelay(nodelay, interval, resend, nc)
		conn.SetWindowSize(kic.config.Sndwnd, kic.config.Rcvwnd)
		conn.SetMtu(kic.config.Mtu)
		conn.SetACKNoDelay(kic.config.Acknodelay)
		conn.SetDSCP(kic.config.Dscp)
		if err != nil {
			log.Info("kcptvn: Failed to Accept KcpUdp connection: %s", err.Error())
			go kic.HandleIncomingConn(conn)
		}
	}
}

func (kic *KCPInboundConnection) Listen(address v2net.Address, port v2net.Port) error {
	lis, err := kcp.ListenWithOptions(kic.config.Fec, kic.config.Address+kic.config.Port, []byte(kic.config.Key))
	if err != nil {
		log.Info("kcptvn: Failed to listen KcpUdp connection: %s", err.Error())
		log.Error("kcptvn: Listened Unsuccessfully: Failed to listen KcpUdp connection")
		return err
	}

	go kic.GoServerConn(lis)
	return nil
}
func (kic *KCPInboundConnection) Port() v2net.Port {
	return kic.listeningPort
}
func init() {
	internal.MustRegisterInboundHandlerCreator("kcptvn",
		func(space app.Space, rawConfig interface{}) (proxy.InboundHandler, error) {
			return NewKCPInboundConnection(rawConfig.(*Config), space), nil
		})
}
