package app

import (
	"errors"

	secondarykeys "example/x/secondarykeys/module"
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/ethereum/go-ethereum/crypto"
	EthereumK1 "github.com/ethereum/go-ethereum/crypto"
)

type HandlerOptions struct {
	ante.HandlerOptions
}

// SecondarySignatureVerificationDecorator verifies the secondary signature in the memo
type SecondarySignatureVerificationDecorator struct{}

// NewSecondarySignatureVerificationDecorator creates a new decorator instance
func NewSecondarySignatureVerificationDecorator() SecondarySignatureVerificationDecorator {
	return SecondarySignatureVerificationDecorator{}
}

func NewAnteHandler(options ante.HandlerOptions) (sdk.AnteHandler, error) {

	if options.AccountKeeper == nil {
		return nil, errors.New("account keeper is required for ante builder")
	}
	if options.BankKeeper == nil {
		return nil, errors.New("bank keeper is required for ante builder")
	}
	if options.SignModeHandler == nil {
		return nil, errors.New("sign mode handler is required for ante builder")
	}

	anteDecorators := []sdk.AnteDecorator{
		ante.NewSetUpContextDecorator(), // outermost AnteDecorator. SetUpContext must be called first
		ante.NewExtensionOptionsDecorator(options.ExtensionOptionChecker),
		ante.NewValidateBasicDecorator(),
		ante.NewTxTimeoutHeightDecorator(),
		ante.NewValidateMemoDecorator(options.AccountKeeper),
		ante.NewConsumeGasForTxSizeDecorator(options.AccountKeeper),
		ante.NewDeductFeeDecorator(options.AccountKeeper, options.BankKeeper, options.FeegrantKeeper, options.TxFeeChecker),
		ante.NewSetPubKeyDecorator(options.AccountKeeper),
		ante.NewValidateSigCountDecorator(options.AccountKeeper),
		ante.NewSigGasConsumeDecorator(options.AccountKeeper, options.SigGasConsumer),
		ante.NewSigVerificationDecorator(options.AccountKeeper, options.SignModeHandler),

		NewSecondarySignatureVerificationDecorator(),
		ante.NewIncrementSequenceDecorator(options.AccountKeeper),
	}

	return sdk.ChainAnteDecorators(anteDecorators...), nil
}

// AnteHandle implements the ante handler interface
func (svd SecondarySignatureVerificationDecorator) AnteHandle(
	ctx sdk.Context,
	tx sdk.Tx,
	simulate bool,
	next sdk.AnteHandler,
) (sdk.Context, error) {
	// Skip verification in simulation mode
	if simulate {
		return next(ctx, tx, simulate)
	}
	// Skip verification during genesis (chain height 0)
	if ctx.BlockHeight() == 0 {
		return next(ctx, tx, simulate)
	}
	// Get the memo from the tx
	memoTx, ok := tx.(sdk.TxWithMemo)
	if !ok {
		return ctx, sdkerrors.ErrTxDecode
	}
	memo := memoTx.GetMemo()

	// If memo is empty, skip
	if memo == "" {
		ctx.Logger().Info("AnteHandle called,empty memo")
		return next(ctx, tx, simulate)
	}
	var foundPrefix bool
	memo, foundPrefix = strings.CutPrefix(memo, secondarykeys.AnteHandlerPrefix)

	// Check if the memo has the prefix.
	if !foundPrefix {
		ctx.Logger().Info("AnteHandle called,no prefix")
		return next(ctx, tx, simulate)
	}
	// Decode the secondarySignature and publicKey from memo
	secondSig, err := DecodeSecondSigFromMemo([]byte(memo))
	if err != nil {
		ctx.Logger().Info("AnteHandle called,decode err", memo)
		return ctx, sdkerrors.ErrInvalidRequest
	}

	// Validate the signature structure
	if err := secondSig.Validate(); err != nil {
		ctx.Logger().Info("AnteHandle called, empty secondsig")
		return ctx, sdkerrors.ErrInvalidRequest
	}

	hsh := crypto.Keccak256([]byte(secondSig.PublicKey))

	// Verify the signature
	if !EthereumK1.VerifySignature(secondSig.PublicKey, hsh, secondSig.Signature) {
		ctx.Logger().Info("AnteHandle called,invalid signature")
		return ctx, fmt.Errorf("signature verification failed")
	}

	ctx.Logger().Info("AnteHandle called,tx valid")
	return next(ctx, tx, simulate)
}
