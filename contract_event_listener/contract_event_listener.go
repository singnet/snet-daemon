package contractlistener

import (
	"github.com/singnet/snet-daemon/v6/blockchain"
)

type EventSignature string

// etcdClientCloser is the subset of etcddb.EtcdClient which is used by the
// contract event listener. It is kept as an interface to make the listener
// testable.
type etcdClientCloser interface {
	Close()
}

type ContractEventListener struct {
	BlockchainProcessor         blockchain.Processor
	CurrentOrganizationMetaData *blockchain.OrganizationMetaData
	CurrentEtcdClient           etcdClientCloser
}
