package app

import (
	"encoding/json"
	secondarykeys "example/x/secondarykeys/module"
	"fmt"
	"log"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	"github.com/ethereum/go-ethereum/crypto"
	EthereumK1 "github.com/ethereum/go-ethereum/crypto"
)

type SecondarySignature struct {
	PublicKey []byte `json:"public_key"`
	Signature []byte `json:"signature"`
}

const (
	ErrInvalidSecondaryPublicKey = "INVALID_SECONDARY_PUBLIC_KEY"
)

func GetAddr(tx sdk.Tx) ([]byte, error) {

	sigTx, ok := tx.(authsigning.SigVerifiableTx)
	if !ok {
		return nil, sdkerrors.ErrTxDecode
	}

	signers, err := sigTx.GetSignaturesV2()
	if err != nil {
		log.Println(err)
		return nil, sdkerrors.ErrPanic
	}
	if len(signers) == 0 {
		return nil, sdkerrors.ErrNoSignatures
	}
	pubKey := signers[0].PubKey
	if pubKey == nil {
		return nil, sdkerrors.ErrInvalidPubKey.Wrap("signature has no public key")
	}
	return signers[0].PubKey.Bytes(), nil

}

func CreateValidMemo() (string, error) {
	// Generate a random Ethereum private key
	secondaryPrivKey, err := EthereumK1.GenerateKey()
	if err != nil {
		return "", err
	}

	// Get the public key (uncompressed format, 65 bytes)
	secondaryPubKey := crypto.FromECDSAPub(&secondaryPrivKey.PublicKey)

	// Sign the predefined messagea
	messageToSign := []byte(secondaryPubKey)
	hsh := crypto.Keccak256(messageToSign)

	signature, err := EthereumK1.Sign(hsh, secondaryPrivKey)
	if err != nil {
		panic(err)
	}
	// Remove the recovery byte of the signature
	sigNoV := signature[:64]

	// Create the secondary signature struct
	secondSig := SecondarySignature{
		PublicKey: secondaryPubKey,
		Signature: sigNoV,
	}

	// Encode it into memo format
	memoBytes, err := EncodeMemoWithSecondSig(secondSig)
	if err != nil {
		return "", err
	}
	log.Println("memo string", string(memoBytes))

	// Verify the signature
	if !EthereumK1.VerifySignature(secondSig.PublicKey, hsh, secondSig.Signature) {
		return "", err
	}

	memo := secondarykeys.AnteHandlerPrefix + string(memoBytes)

	return memo, nil
}

func (s *SecondarySignature) Validate() error {
	if len(s.PublicKey) == 0 {
		return fmt.Errorf("missing public key")
	}
	if len(s.Signature) == 0 {
		return fmt.Errorf("missing signature")
	}
	return nil
}

// EncodeMemoWithSecondSig - just encode the signature
func EncodeMemoWithSecondSig(secondSig SecondarySignature) ([]byte, error) {

	memoBytes, err := json.Marshal(secondSig)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal memo: %w", err)
	}

	return memoBytes, nil
}

// DecodeSecondSigFromMemo - extract the signature and remove recovery byte from signature
func DecodeSecondSigFromMemo(memo []byte) (*SecondarySignature, error) {
	if memo == nil {
		return nil, fmt.Errorf("empty memo")
	}

	var memoData SecondarySignature
	if err := json.Unmarshal(memo, &memoData); err != nil {
		return nil, fmt.Errorf("invalid memo format: %w", err)
	}

	sig := memoData.Signature

	// remove the recovery byte from the signature
	if len(sig) == 65 {
		sig = sig[:64]
	}

	return &SecondarySignature{
		PublicKey: memoData.PublicKey,
		Signature: sig,
	}, nil
}
