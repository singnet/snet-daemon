//go:generate protoc -I . ./state_service.proto --go_out=plugins=grpc:.
package escrow

import (
	"github.com/singnet/snet-daemon/blockchain"
	"golang.org/x/net/context"
	"math/big"
)

type PaymentChannelStateService struct {
}

func (service *PaymentChannelStateService) GetChannelState(context context.Context, request *ChannelStateRequest) (reply *ChannelStateReply, err error) {
	return &ChannelStateReply{
		CurrentNonce: bigIntToBytes(big.NewInt(1)),
		CurrentValue: bigIntToBytes(big.NewInt(2)),
		SignedNonce:  bigIntToBytes(big.NewInt(3)),
		SignedAmount: bigIntToBytes(big.NewInt(4)),
		Signature:    blockchain.HexToBytes("0xde4e998341307b036e460b1cc1593ddefe2e9ea261bd6c3d75967b29b2c3d0a24969b4a32b099ae2eded90bbc213ad0a159a66af6d55be7e04f724ffa52ce3cc1b"),
	}, nil
}
