package escrow

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type PrePaidService interface {
	GetUsage(key PrePaidDataKey) (*PrePaidData, bool, error)
	UpdateUsage(channelId *big.Int, revisedAmount *big.Int, updateUsageType string) error
}

type PrePaidTransaction interface {
	ChannelId() *big.Int
	Price() *big.Int
	Commit() error
	Rollback() error
}

type prePaidTransactionImpl struct {
	channelId *big.Int
	price     *big.Int
	signer    common.Address
}

func (transaction *prePaidTransactionImpl) GetSender() common.Address {
	return transaction.signer
}

func (transaction prePaidTransactionImpl) ChannelId() *big.Int {
	return transaction.channelId
}
func (transaction prePaidTransactionImpl) Price() *big.Int {
	return transaction.price
}
func (transaction prePaidTransactionImpl) Commit() error {
	return nil
}
func (transaction prePaidTransactionImpl) Rollback() error {
	return nil
}
