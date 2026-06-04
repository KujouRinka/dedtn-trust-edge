package smart_bft

type peer struct {
	*RpcClient
	id uint64
}

func newPeer(id uint64, host string, port uint16) (*peer, error) {
	client, err := NewRpcClient(host, port)
	if err != nil {
		return nil, err
	}
	return &peer{
		RpcClient: client,
		id:        id,
	}, nil
}

func (p *peer) Id() uint64 {
	return p.id
}
