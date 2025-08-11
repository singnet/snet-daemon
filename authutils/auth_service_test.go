// authutils package provides functions for all authentication and signature validation related operations
package authutils

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

func TestVerifyAddress(t *testing.T) {
	var addr = common.Address(common.FromHex("0x7DF35C98f41F3AF0DF1DC4C7F7D4C19A71DD079F"))
	var addrLowCase = common.Address(common.FromHex("0x7df35c98f41f3af0df1dc4c7f7d4c19a71Dd079f"))
	assert.Nil(t, VerifyAddress(addr, addrLowCase))
}
