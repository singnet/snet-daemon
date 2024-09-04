package contract_event_listener

import (
	"github.com/singnet/snet-daemon/blockchain"
	"github.com/singnet/snet-daemon/etcddb"
)

type EventSignature string

const (
	contractAddress                                = "0x4DCc70c6FCE4064803f0ae0cE48497B3f7182e5D"
	UpdateMetadataUriEventSignature EventSignature = "0x06ccb920be65231f5c9d04dd4883d3c7648ebe5f5317cc7177ee4f4a7cc2d038"
)

type ContractEventListener struct {
	BlockchainProcessor         *blockchain.Processor
	EventSignature              EventSignature
	CurrentOrganizationMetaData *blockchain.OrganizationMetaData
	CurrentEtcdClient           *etcddb.EtcdClient
}
