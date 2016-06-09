package kcptun

import (
	"encoding/json"
	"io"
)

import v2net "github.com/v2ray/v2ray-core/common/net"

func decodeDestJSON(input io.Reader) v2net.Destination {
	decoder := json.NewDecoder(input)
	var dest jsonDest
	decoder.Decode(&dest)
	p, _ := v2net.PortFromInt(dest.Port)
	destv2 := v2net.TCPDestination(v2net.ParseAddress(dest.Address), p)
	return destv2
}

type jsonDest struct {
	Address string `json:"Address"`
	Port    int    `json:"Port"`
}

func encodeDestJSON(dest v2net.Destination, output io.Writer) {
	var destj jsonDest
	destj.Address = dest.Address().String()
	destj.Port = int(dest.Port().Value())
	encoder := json.NewEncoder(output)
	encoder.Encode(destj)
}
