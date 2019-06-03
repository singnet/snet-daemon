//  authutils package provides functions for all authentication and singature validation related operations
package authutils

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/singnet/snet-daemon/blockchain"
	log "github.com/sirupsen/logrus"
	"math/big"
)

// TODO convert to separate authentication service. VERY MUCH REQUIRED FOR OPERATOR UI AUTHENTICATION

// Extracts the signer address from signature given the signature
// It returns signer address and error. nil error indicates the successful function execution
func GetSignerAddressFromMessage(message, signature []byte) (signer *common.Address, err error) {
	log := log.WithFields(log.Fields{
		"message":   blockchain.BytesToBase64(message),
		"signature": blockchain.BytesToBase64(signature),
	})

	messageHash := crypto.Keccak256(
		blockchain.HashPrefix32Bytes,
		crypto.Keccak256(message),
	)
	log = log.WithField("messageHash", hex.EncodeToString(messageHash))

	v, _, _, e := blockchain.ParseSignature(signature)
	if e != nil {
		log.WithError(e).Warn("Error parsing signature")
		return nil, errors.New("incorrect signature length")
	}

	modifiedSignature := bytes.Join([][]byte{signature[0:64], {v % 27}}, nil)
	publicKey, e := crypto.SigToPub(messageHash, modifiedSignature)
	if e != nil {
		log.WithError(e).WithField("modifiedSignature", modifiedSignature).Warn("Incorrect signature")
		return nil, errors.New("incorrect signature data")
	}
	log = log.WithField("publicKey", publicKey)

	keyOwnerAddress := crypto.PubkeyToAddress(*publicKey)
	log.WithField("keyOwnerAddress", keyOwnerAddress).Debug("Message signature parsed")

	return &keyOwnerAddress, nil
}

// Verify the signature done by given singer or not
// returns nil if signer indeed sign the message and singature proves it, if not throws an error
func VerifySigner(message []byte, signature []byte, signer common.Address) error {
	signerFromMessage, err := GetSignerAddressFromMessage(message, signature)
	if err != nil {
		log.Error(err)
		return err
	}
	if signerFromMessage.String() == signer.String() {
		return nil
	}
	return fmt.Errorf("Incorrect signer.")
}

//Check if the block number passed is not more +- 5 from the latest block number on chain
func CompareWithLatestBlockNumber(blockNumberPassed *big.Int) error {
	latestBlockNumber, err := CurrentBlock()
	if err != nil {
		return err
	}
	differenceInBlockNumber := blockNumberPassed.Sub(blockNumberPassed, latestBlockNumber)
	if differenceInBlockNumber.Abs(differenceInBlockNumber).Uint64() > 5 {
		return fmt.Errorf("difference between the latest block chain number and the block number passed is %v ", differenceInBlockNumber)
	}
	return nil
}

//Get the current block number from on chain
func CurrentBlock() (*big.Int, error) {
	if ethClient, err := blockchain.GetEthereumClient(); err != nil {
		return nil, err
	} else {
		defer ethClient.RawClient.Close()
		var currentBlockHex string
		if err = ethClient.RawClient.CallContext(context.Background(), &currentBlockHex, "eth_blockNumber"); err != nil {
			log.WithError(err).Error("error determining current block")
			return nil, fmt.Errorf("error determining current block: %v", err)
		}
		return new(big.Int).SetBytes(common.FromHex(currentBlockHex)), nil
	}
}

//Check if the payment address/signer passed matches to what is present in the metadata
func VerifyAddress(address common.Address, otherAddress common.Address) error {
	isSameAddress := otherAddress == address
	if !isSameAddress {
		return fmt.Errorf("the payment Address: %s  does not match to what has been registered", blockchain.AddressToHex(&address))
	}
	return nil
}
