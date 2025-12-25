package secondaryKeyAnteHandler

import (
	"encoding/json"
	"fmt"
)

type SecondarySignature struct {
	PublicKey []byte `json:"public_key"`
	Signature []byte `json:"signature"`
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
