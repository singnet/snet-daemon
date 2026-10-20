package utils

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
	// HashPrefix32Bytes is the Ethereum signed-message prefix for a 32-byte digest.
	HashPrefix32Bytes = []byte("\x19Ethereum Signed Message:\n32")
)

// VerifySigner recovers the signer using the daemon's Ethereum signed-message
// hashing convention and compares it with the supplied expected address.
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

// GetSignerAddressFromMessage recovers a signer from a 65-byte R || S || V
// signature over Keccak256(HashPrefix32Bytes || Keccak256(message)). It normalizes
// conventional recovery IDs 0/1 and 27/28 without modifying the supplied signature.
// Callers must construct the protocol payload and enforce signer authorization.
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

	keyOwnerAddress := crypto.PubkeyToAddress(*publicKey)
	keyOwnerAddressFieldLog := zap.Any("keyOwnerAddress", keyOwnerAddress)
	zap.L().Debug("Message signature parsed",
		keyOwnerAddressFieldLog)

	return &keyOwnerAddress, nil
}

// VerifyAddress compares two parsed Ethereum addresses. It does not consult
// metadata, recover a signature, or read a contract.
func VerifyAddress(address common.Address, otherAddress common.Address) error {
	if otherAddress != address {
		return fmt.Errorf("the address: %s does not match to what has been expected / registered", address.Hex())
	}
	return nil
}

// GetSignature signs Keccak256(HashPrefix32Bytes || Keccak256(message)) and returns
// R || S || V with recovery ID 0/1. A nil private key returns nil.
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
