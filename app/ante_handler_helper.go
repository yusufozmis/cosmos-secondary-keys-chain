package app

import (
	"encoding/json"
	"errors"
	secondarykeys "example/x/secondarykeys/module"
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
)

type SecondarySignature struct {
	PublicKey []byte `json:"public_key"`
	Signature []byte `json:"signature"`
}

const ErrNoPrefix = "NO_PREFIX"

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
func GetAddr(tx sdk.Tx) ([]byte, error) {

	sigTx, ok := tx.(authsigning.SigVerifiableTx)
	if !ok {
		return nil, sdkerrors.ErrTxDecode
	}

	signers, err := sigTx.GetSigners()
	if err != nil {
		return nil, sdkerrors.ErrPanic
	}
	if len(signers) == 0 {
		return nil, sdkerrors.ErrNoSignatures
	}

	return signers[0], nil

}
func ParseMemo(tx sdk.Tx) (*SecondarySignature, error) {
	// Get the memo from the tx
	memoTx, ok := tx.(sdk.TxWithMemo)
	if !ok {
		return &SecondarySignature{}, sdkerrors.ErrTxDecode
	}
	memo := memoTx.GetMemo()

	// If memo is empty, skip
	if memo == "" {
		return &SecondarySignature{}, errors.New("Empty Memo")
	}
	var foundPrefix bool
	memo, foundPrefix = strings.CutPrefix(memo, secondarykeys.AnteHandlerPrefix)

	// Check if the memo has the prefix.
	if !foundPrefix {
		return &SecondarySignature{}, errors.New(ErrNoPrefix)
	}
	// Decode the secondarySignature and publicKey from memo
	secondSig, err := DecodeSecondSigFromMemo([]byte(memo))
	if err != nil {
		return &SecondarySignature{}, sdkerrors.ErrInvalidRequest
	}
	return secondSig, nil
}
