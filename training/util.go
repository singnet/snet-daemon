package training

import (
	"bytes"
	"math/big"

	"github.com/ethereum/go-ethereum/common/math"
	"github.com/singnet/snet-daemon/v5/authutils"
	"github.com/singnet/snet-daemon/v5/utils"
)

// message used to sign is of the form ("__create_model", mpe_address, current_block_number)
func (ds *DaemonService) verifySignature(request *AuthorizationDetails) error {
	return authutils.VerifySigner(ds.getMessageBytes(request.Message, request),
		request.GetSignature(), utils.ToChecksumAddress(request.SignerAddress))
}

// "user passed message	", user_address, current_block_number
func (ds *DaemonService) getMessageBytes(prefixMessage string, request *AuthorizationDetails) []byte {
	userAddress := utils.ToChecksumAddress(request.SignerAddress)
	message := bytes.Join([][]byte{
		[]byte(prefixMessage),
		userAddress.Bytes(),
		math.U256Bytes(big.NewInt(int64(request.CurrentBlock))),
	}, nil)
	return message
}

func remove(s []string, r string) []string {
	for i, v := range s {
		if v == r {
			return append(s[:i], s[i+1:]...)
		}
	}
	return s
}

func difference(oldAddresses []string, newAddresses []string) []string {
	var diff []string
	for i := 0; i < 2; i++ {
		for _, s1 := range oldAddresses {
			found := false
			for _, s2 := range newAddresses {
				if s1 == s2 {
					found = true
					break
				}
			}
			// String not found. We add it to return slice
			if !found {
				diff = append(diff, s1)
			}
		}
		// Swap the slices, only if it was the first loop
		if i == 0 {
			oldAddresses, newAddresses = newAddresses, oldAddresses
		}
	}
	return diff
}

func isValuePresent(value string, list []string) bool {
	for _, v := range list {
		if v == value {
			return true
		}
	}
	return false
}
