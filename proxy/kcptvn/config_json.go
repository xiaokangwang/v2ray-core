// +build json
package kcptun

import (
	"encoding/json"

	"github.com/v2ray/v2ray-core/proxy/internal/config"
)

func init() {
	config.RegisterOutboundConfig("kcptvn",
		func(data []byte) (interface{}, error) {
			c := new(Config)
			if err := json.Unmarshal(data, c); err != nil {
				return nil, err
			}
			return c, nil
		})
	config.RegisterInboundConfig("kcptvn",
		func(data []byte) (interface{}, error) {
			c := new(Config)
			if err := json.Unmarshal(data, c); err != nil {
				return nil, err
			}
			return c, nil
		})
}
