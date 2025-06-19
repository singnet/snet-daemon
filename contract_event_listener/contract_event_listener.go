package contractlistener

import (
	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/etcddb"
)

type EventSignature string

type ContractEventListener struct {
	BlockchainProcessor         blockchain.Processor
	CurrentOrganizationMetaData *blockchain.OrganizationMetaData
	CurrentEtcdClient           *etcddb.EtcdClient
}
