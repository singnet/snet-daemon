package escrow

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// PrePaidService maintains prepaid accounting counters independently of channel
// nonce and claim settlement.
type PrePaidService interface {
	// GetUsage returns one counter, with ok false when no record is stored.
	GetUsage(key PrePaidDataKey) (*PrePaidData, bool, error)
	// UpdateUsage adds revisedAmount as a delta to the selected counter.
	// Used-amount updates fail if used would exceed planned plus refunds.
	UpdateUsage(channelId *big.Int, revisedAmount *big.Int, updateUsageType string) error
}

// PrePaidTransaction records a call's channel and charged price. The built-in
// implementation's Commit and Rollback are no-ops; the handler records usage
// before invoking the service and credits failures through PrePaidService.UpdateUsage.
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
