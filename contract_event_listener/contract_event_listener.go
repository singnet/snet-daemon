package contractlistener

import (
	"github.com/singnet/snet-daemon/blockchain"
	"github.com/singnet/snet-daemon/etcddb"
)

type EventSignature string

type ContractEventListener struct {
	BlockchainProcessor         *blockchain.Processor
	CurrentOrganizationMetaData *blockchain.OrganizationMetaData
	CurrentEtcdClient           *etcddb.EtcdClient
}
