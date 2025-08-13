// Package authutils provides functions for all authentication and signature validation related operations
package authutils

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/singnet/snet-daemon/v6/blockchain"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"go.uber.org/zap"
)

// TODO convert to separate authentication service. VERY MUCH REQUIRED FOR OPERATOR UI AUTHENTICATION

// Extracts the signer address from signature given the signature
// It returns signer address and error. nil error indicates the successful function execution

const (
	AllowedBlockChainDifference = 5
)

func VerifySigner(message []byte, signature []byte, signer common.Address) error {
	derivedSigner, err := GetSignerAddressFromMessage(message, signature)
	if err != nil {
		zap.L().Error(err.Error())
		return err
	}
	if err = VerifyAddress(*derivedSigner, signer); err != nil {
		return err
	}
	return nil
}

func GetSignerAddressFromMessage(message, signature []byte) (signer *common.Address, err error) {
	messageFieldLog := zap.String("message", blockchain.BytesToBase64(message))
	signatureFieldLog := zap.String("signature", blockchain.BytesToBase64(signature))

	messageHash := crypto.Keccak256(
		blockchain.HashPrefix32Bytes,
		crypto.Keccak256(message),
	)
	messageHashFieldLog := zap.String("messageHash", hex.EncodeToString(messageHash))

	v, _, _, err := blockchain.ParseSignature(signature)
	if err != nil {
		zap.L().Warn("Error parsing signature", zap.Error(err), messageFieldLog, signatureFieldLog, messageHashFieldLog)
		return nil, errors.New("incorrect signature length")
	}

	modifiedSignature := bytes.Join([][]byte{signature[0:64], {v % 27}}, nil)
	publicKey, err := crypto.SigToPub(messageHash, modifiedSignature)
	modifiedSignatureFieldLog := zap.ByteString("modifiedSignature", modifiedSignature)
	if err != nil {
		zap.L().Warn("Incorrect signature",
			zap.Error(err),
			modifiedSignatureFieldLog,
			messageFieldLog,
			signatureFieldLog,
			messageHashFieldLog)
		return nil, errors.New("incorrect signature data")
	}
	//publicKeyFieldLog := zap.Any("publicKey", publicKey)

	keyOwnerAddress := crypto.PubkeyToAddress(*publicKey)
	keyOwnerAddressFieldLog := zap.Any("keyOwnerAddress", keyOwnerAddress)
	zap.L().Debug("Message signature parsed",
		//messageFieldLog,
		//signatureFieldLog,
		//messageHashFieldLog,
		//publicKeyFieldLog,
		keyOwnerAddressFieldLog)

	return &keyOwnerAddress, nil
}

// VerifySigner Verify the signature done by given singer or not
// returns nil if signer indeed sign the message and signature proves it, if not throws an error
//func VerifySigner(message []byte, signature []byte, signer common.Address) error {
//	signerFromMessage, err := GetSignerAddressFromMessage(message, signature)
//	if err != nil {
//		zap.L().Error("error from getSignerAddressFromMessage", zap.Error(err))
//		return err
//	}
//	if signerFromMessage.String() == signer.String() {
//		return nil
//	}
//	return fmt.Errorf("incorrect signer")
//}

// deprecated
// CompareWithLatestBlockNumber Check if the block number passed is not more +- 5 from the latest block number on chain
//func CompareWithLatestBlockNumber(blockNumberPassed *big.Int) error {
//	latestBlockNumber, err := CurrentBlock()
//	if err != nil {
//		return err
//	}
//	differenceInBlockNumber := blockNumberPassed.Sub(blockNumberPassed, latestBlockNumber)
//	if differenceInBlockNumber.Abs(differenceInBlockNumber).Uint64() > AllowedBlockChainDifference {
//		return fmt.Errorf("authentication failed as the signature passed has expired")
//	}
//	return nil
//}

// CurrentBlock Get the current block number from on chain
// deprecated
//func CurrentBlock() (*big.Int, error) {
//	if ethHttpClient, err := blockchain.CreateHTTPEthereumClient(); err != nil {
//		return nil, err
//	} else {
//		defer ethHttpClient.RawClient.Close()
//		var currentBlockHex string
//		if err = ethHttpClient.RawClient.CallContext(context.Background(), &currentBlockHex, "eth_blockNumber"); err != nil {
//			zap.L().Error("error determining current block", zap.Error(err))
//			return nil, fmt.Errorf("error determining current block: %v", err)
//		}
//		return new(big.Int).SetBytes(common.FromHex(currentBlockHex)), nil
//	}
//}

// VerifyAddress Check if the payment address/signer passed matches to what is present in the metadata
func VerifyAddress(address common.Address, otherAddress common.Address) error {
	if otherAddress != address {
		return fmt.Errorf("the address: %s does not match to what has been expected / registered", blockchain.AddressToHex(&address))
	}
	return nil
}

func GetSignature(message []byte, privateKey *ecdsa.PrivateKey) (signature []byte) {
	hash := crypto.Keccak256(
		blockchain.HashPrefix32Bytes,
		crypto.Keccak256(message),
	)

	if privateKey == nil {
		return nil
	}

	signature, err := crypto.Sign(hash, privateKey)
	if err != nil {
		zap.L().Fatal(fmt.Sprintf("Cannot sign test message: %v", err))
	}

	return signature
}
