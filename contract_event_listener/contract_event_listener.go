package contractlistener

import (
	"github.com/singnet/snet-daemon/v5/blockchain"
	"github.com/singnet/snet-daemon/v5/etcddb"
)

type EventSignature string

type ContractEventListener struct {
	BlockchainProcessor         blockchain.Processor
	CurrentOrganizationMetaData *blockchain.OrganizationMetaData
	CurrentEtcdClient           *etcddb.EtcdClient
}
