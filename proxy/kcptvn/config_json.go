// +build json
package kcptun

import (
	"encoding/json"

	"github.com/v2ray/v2ray-core/proxy/internal/config"
)

func (this *Config) UnmarshalJSON(data []byte) error {

	if err := json.Unmarshal(data, this); err != nil {
		return err
	}
	return nil
}

func init() {
	config.RegisterOutboundConfig("kcptun",
		func(data []byte) (interface{}, error) {
			c := new(Config)
			if err := json.Unmarshal(data, c); err != nil {
				return nil, err
			}
			return c, nil
		})
}
