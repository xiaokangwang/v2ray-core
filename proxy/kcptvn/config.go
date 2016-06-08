package kcptun

/*Config : Kcptun configure
HeaderFormat: The formart of stream header
https://github.com/xtaci/kcptun
*/
type Config struct {
	Headerfmt  string `json:"HeaderFormat"`
	Address    string `json:"Address"`
	Port       string `json:"Port"`
	Mode       string `json:"Mode"`
	Key        string `json:"EncryptionKey"`
	Mtu        int    `json:"MaximumTransmissionUnit"`
	Sndwnd     int    `json:"SendingWindowSize"`
	Rcvwnd     int    `json:"ReceivingWindowSize"`
	Fec        int    `json:"ForwardErrorCorrectionGroupSize"`
	Acknodelay bool   `json:"AcknowledgeNoDelay"`
	Dscp       int    `json:"Dscp"`
}
