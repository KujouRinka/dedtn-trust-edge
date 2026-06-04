package smart_bft

type Config struct {
	Id      uint64
	NodeDir string

	ListenAddr string
	ListenPort uint16

	Peers []*PeerCfg
}

type PeerCfg struct {
	Id   uint64
	Addr string
	Port uint16
}
