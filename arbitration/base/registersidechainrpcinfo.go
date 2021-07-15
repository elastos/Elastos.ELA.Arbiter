package base

type RegisterSidechainRpcInfo struct {
	GenesisBlockHash string `json:"genesisblockhash"`
	Httpjsonport     int    `json:"httpjsonport"`
	IpAddr           string `json:"ipaddr"`
	User             string `json:"user"`
	Pass             string `json:"pass"`
}
