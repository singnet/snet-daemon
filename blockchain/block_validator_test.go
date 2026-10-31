package blockchain

import (
	"errors"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEthereumBlockValidatorValidateBlock(t *testing.T) {
	largeDifference := new(big.Int).Lsh(big.NewInt(1), 64)
	largeBlock := new(big.Int).Add(big.NewInt(100), largeDifference)
	for _, tt := range []struct {
		name      string
		current   *big.Int
		block     *big.Int
		allowance uint64
		valid     bool
	}{
		{name: "current", current: big.NewInt(100), block: big.NewInt(100), allowance: 5, valid: true},
		{name: "past boundary", current: big.NewInt(100), block: big.NewInt(95), allowance: 5, valid: true},
		{name: "future boundary", current: big.NewInt(100), block: big.NewInt(105), allowance: 5, valid: true},
		{name: "stale", current: big.NewInt(100), block: big.NewInt(94), allowance: 5},
		{name: "future", current: big.NewInt(100), block: big.NewInt(106), allowance: 5},
		{name: "zero allowance equal", current: big.NewInt(100), block: big.NewInt(100), valid: true},
		{name: "zero allowance past", current: big.NewInt(100), block: big.NewInt(99)},
		{name: "zero allowance future", current: big.NewInt(100), block: big.NewInt(101)},
		{name: "genesis", current: big.NewInt(0), block: big.NewInt(0), valid: true},
		{name: "large future difference", current: big.NewInt(100), block: largeBlock, allowance: 5},
		{name: "large past difference", current: largeBlock, block: big.NewInt(100), allowance: 5},
		{name: "maximum allowance boundary", current: big.NewInt(0), block: new(big.Int).SetUint64(^uint64(0)), allowance: ^uint64(0), valid: true},
		{name: "maximum allowance exceeded", current: big.NewInt(0), block: largeDifference, allowance: ^uint64(0)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			currentBefore := new(big.Int).Set(tt.current)
			blockBefore := new(big.Int).Set(tt.block)
			validator := NewEthereumBlockValidator(func() (*big.Int, error) { return tt.current, nil })
			for range 2 {
				err := validator.ValidateBlock(tt.block, tt.allowance)
				if tt.valid {
					require.NoError(t, err)
				} else {
					require.EqualError(t, err, "authentication failed as the signature passed has expired")
				}
				require.Equal(t, currentBefore, tt.current)
				require.Equal(t, blockBefore, tt.block)
			}
		})
	}
}

func TestEthereumBlockValidatorInvalidRequest(t *testing.T) {
	validator := NewEthereumBlockValidator(func() (*big.Int, error) {
		t.Fatal("invalid input must be rejected before reading the current block")
		return nil, nil
	})
	for _, block := range []*big.Int{nil, big.NewInt(-1)} {
		require.EqualError(t, validator.ValidateBlock(block, 5), "invalid block number")
	}
}

func TestEthereumBlockValidatorInvalidSource(t *testing.T) {
	for _, block := range []*big.Int{nil, big.NewInt(-1)} {
		validator := NewEthereumBlockValidator(func() (*big.Int, error) { return block, nil })
		current, err := validator.CurrentBlock()
		require.Nil(t, current)
		require.EqualError(t, err, "cannot determine current block: invalid block number")
		require.EqualError(t, validator.ValidateBlock(big.NewInt(100), 5), err.Error())
	}
}

func TestEthereumBlockValidatorSourceError(t *testing.T) {
	wantErr := errors.New("blockchain unavailable")
	validator := NewEthereumBlockValidator(func() (*big.Int, error) { return nil, wantErr })
	current, err := validator.CurrentBlock()
	require.Nil(t, current)
	require.ErrorIs(t, err, wantErr)
	require.ErrorIs(t, validator.ValidateBlock(big.NewInt(100), 5), wantErr)
}

func TestEthereumBlockValidatorCopiesCurrentBlock(t *testing.T) {
	source := big.NewInt(100)
	reads := 0
	validator := NewEthereumBlockValidator(func() (*big.Int, error) {
		reads++
		return source, nil
	})
	current, err := validator.CurrentBlock()
	require.NoError(t, err)
	require.NotSame(t, source, current)
	current.Add(current, big.NewInt(5))
	require.Equal(t, big.NewInt(100), source)
	require.NoError(t, validator.ValidateBlock(source, 0))
	source.SetInt64(101)
	require.NoError(t, validator.ValidateBlock(big.NewInt(101), 0))
	require.Equal(t, 3, reads, "each operation must read the current block without caching")
}
