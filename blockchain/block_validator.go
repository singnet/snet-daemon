package blockchain

import (
	"fmt"
	"math/big"
)

// EthereumBlockValidator reads the current block and validates request freshness.
type EthereumBlockValidator struct {
	currentBlock func() (*big.Int, error)
}

// NewEthereumBlockValidator creates a validator using the supplied block source.
func NewEthereumBlockValidator(currentBlock func() (*big.Int, error)) *EthereumBlockValidator {
	return &EthereumBlockValidator{currentBlock: currentBlock}
}

// CurrentBlock returns a nonnegative copy of the latest block from the source.
func (validator *EthereumBlockValidator) CurrentBlock() (*big.Int, error) {
	currentBlock, err := validator.currentBlock()
	if err != nil {
		return nil, err
	}
	if currentBlock == nil || currentBlock.Sign() < 0 {
		return nil, fmt.Errorf("cannot determine current block: invalid block number")
	}
	return new(big.Int).Set(currentBlock), nil
}

// ValidateBlock accepts blocks within the inclusive allowance in either direction.
// Neither the supplied block nor the source's block is modified.
func (validator *EthereumBlockValidator) ValidateBlock(block *big.Int, allowedDifference uint64) error {
	if block == nil || block.Sign() < 0 {
		return fmt.Errorf("invalid block number")
	}
	currentBlock, err := validator.CurrentBlock()
	if err != nil {
		return err
	}
	difference := new(big.Int).Sub(block, currentBlock)
	if difference.Abs(difference).Cmp(new(big.Int).SetUint64(allowedDifference)) > 0 {
		return fmt.Errorf("authentication failed as the signature passed has expired")
	}
	return nil
}
