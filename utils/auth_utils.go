package utils

// auth_utils.go provides functions for all authentication and signature validation related operations

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"go.uber.org/zap"
)

var (
	// HashPrefix32Bytes is an Ethereum signature prefix: see https://github.com/ethereum/go-ethereum/blob/bf468a81ec261745b25206b2a596eb0ee0a24a74/internal/ethapi/api.go#L361
	HashPrefix32Bytes = []byte("\x19Ethereum Signed Message:\n32")
)

// VerifySigner Extracts the signer address from the signature given the signature
// It returns signer address and error. nil error indicates the successful function execution
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
	messageFieldLog := zap.String("message", BytesToBase64(message))
	signatureFieldLog := zap.String("signature", BytesToBase64(signature))

	messageHash := crypto.Keccak256(
		HashPrefix32Bytes,
		crypto.Keccak256(message),
	)
	messageHashFieldLog := zap.String("messageHash", hex.EncodeToString(messageHash))

	v, _, _, err := ParseSignature(signature)
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

// VerifyAddress Check if the payment address/signer passed matches to what is present in the metadata
func VerifyAddress(address common.Address, otherAddress common.Address) error {
	if otherAddress != address {
		return fmt.Errorf("the address: %s does not match to what has been expected / registered", address.Hex())
	}
	return nil
}

func GetSignature(message []byte, privateKey *ecdsa.PrivateKey) (signature []byte) {
	hash := crypto.Keccak256(
		HashPrefix32Bytes,
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
