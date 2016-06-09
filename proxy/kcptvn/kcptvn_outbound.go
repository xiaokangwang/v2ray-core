package kcptun

import (
	"errors"
	"io"
	"sync"

	"github.com/v2ray/v2ray-core/app"
	"github.com/v2ray/v2ray-core/common/alloc"
	v2io "github.com/v2ray/v2ray-core/common/io"
	"github.com/v2ray/v2ray-core/common/log"
	v2net "github.com/v2ray/v2ray-core/common/net"
	"github.com/v2ray/v2ray-core/proxy"
	"github.com/v2ray/v2ray-core/proxy/internal"
	"github.com/v2ray/v2ray-core/transport/ray"
)
import "github.com/xtaci/kcp-go"

func NewKCPOutboundConnection(config *Config, space app.Space) *KCPOutboundConnection {
	c := &KCPOutboundConnection{}

	c.config = config
	return c
}

type KCPOutboundConnection struct {
	config *Config
}

func (koc *KCPOutboundConnection) InitializeNewKCPOutboundConnection() (*kcp.UDPSession, error) {
	conn, err := kcp.DialWithOptions(koc.config.Fec, koc.config.Address+":"+koc.config.Port, []byte(koc.config.Key))
	if err != nil {
		log.Info("kcptvn: Failed to initialize KcpUdp connection: %s", err.Error())
	}
	return conn, err
}

func (koc *KCPOutboundConnection) EncodeAndSendHeader(conn io.Writer, destination v2net.Destination) {
	encodeDestJSON(destination, conn)
}
func (koc *KCPOutboundConnection) Dispatch(destination v2net.Destination, payload *alloc.Buffer, ray ray.OutboundRay) error {
	log.Info("kcptvn: Dispatching.")
	koconn, err := koc.InitializeNewKCPOutboundConnection()
	defer ray.OutboundInput().Release()
	defer ray.OutboundOutput().Close()
	if err != nil {
		log.Error("kcptvn: Connected Unsuccessfully: Failed to initialize KcpUdp connection")
		return err
	}
	nodelay, interval, resend, nc := 0, 40, 0, 0
	if koc.config.Mode != "manual" {
		switch koc.config.Mode {
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
		log.Error("kcptvn: Connected Unsuccessfully: Manual mode is not supported.(yet!)")
		return errors.New("NI")
	}

	koconn.SetNoDelay(nodelay, interval, resend, nc)
	koconn.SetWindowSize(koc.config.Sndwnd, koc.config.Rcvwnd)
	koconn.SetMtu(koc.config.Mtu)
	koconn.SetACKNoDelay(koc.config.Acknodelay)
	koconn.SetDSCP(koc.config.Dscp)

	koc.EncodeAndSendHeader(koconn, destination)

	input := ray.OutboundInput()
	output := ray.OutboundOutput()
	var readMutex, writeMutex sync.Mutex
	readMutex.Lock()
	writeMutex.Lock()
	koconn.Write(payload.Value)
	payload.Release()
	go func() {
		v2writer := v2io.NewAdaptiveWriter(koconn)
		defer v2writer.Release()

		v2io.Pipe(input, v2writer)
		writeMutex.Unlock()
	}()

	go func() {
		defer readMutex.Unlock()

		var reader io.Reader = koconn

		v2reader := v2io.NewAdaptiveReader(reader)
		defer v2reader.Release()

		v2io.Pipe(v2reader, output)
		ray.OutboundOutput().Close()
	}()

	writeMutex.Lock()
	readMutex.Lock()

	koconn.Close()
	return nil
}

func init() {
	internal.MustRegisterOutboundHandlerCreator("kcptvn",
		func(space app.Space, config interface{}) (proxy.OutboundHandler, error) {
			return NewKCPOutboundConnection(config.(*Config), space), nil
		})
}
