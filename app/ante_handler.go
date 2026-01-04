package app

import (
	"bytes"
	"errors"

	"example/common"
	"example/x/secondarykeys/keeper"
	secondarykeys "example/x/secondarykeys/module"
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	"github.com/ethereum/go-ethereum/crypto"
	EthereumK1 "github.com/ethereum/go-ethereum/crypto"
)

type HandlerOptions struct {
	ante.HandlerOptions
}

// SecondarySignatureVerificationDecorator verifies the secondary signature in the memo
type SecondarySignatureVerificationDecorator struct {
	k keeper.Keeper
}

// NewSecondarySignatureVerificationDecorator creates a new decorator instance
func NewSecondarySignatureVerificationDecorator(k keeper.Keeper) SecondarySignatureVerificationDecorator {
	return SecondarySignatureVerificationDecorator{
		k: k,
	}
}

func NewAnteHandler(options ante.HandlerOptions, secondaryKeeper keeper.Keeper) (sdk.AnteHandler, error) {

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

		NewSecondarySignatureVerificationDecorator(secondaryKeeper),
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

	// Get the memo from the tx
	memoTx, ok := tx.(sdk.TxWithMemo)
	if !ok {
		return ctx, sdkerrors.ErrTxDecode
	}
	memo := memoTx.GetMemo()

	var foundPrefix bool
	memo, foundPrefix = strings.CutPrefix(memo, secondarykeys.AnteHandlerPrefix)

	// Check if the memo has the prefix.
	if !foundPrefix {
		ctx.Logger().Info("AnteHandle called,no prefix")
		return next(ctx, tx, simulate)
	}
	// Decode the secondarySignature and publicKey from memo
	secondSig, err := common.DecodeSecondSigFromMemo([]byte(memo))
	if err != nil {
		ctx.Logger().Info("AnteHandle called,decode err", memo)
		return ctx, sdkerrors.ErrInvalidRequest
	}
	addr, err := common.GetAddr(tx)
	if err != nil {
		return ctx, sdkerrors.ErrLogic
	}
	exists, err := svd.k.AnteHandlerMap.Has(ctx, addr)
	if err != nil {
		return ctx, errors.New(common.ErrInvalidSecondaryPublicKey)
	}
	if !exists {
		return ctx, sdkerrors.ErrNotFound
	}
	mappedVal, err := svd.k.GetSecondaryPubKeyAnteHandler(ctx, addr)
	if err != nil {
		return ctx, err
	}
	if !bytes.Equal(mappedVal, secondSig.PublicKey) {
		return ctx, errors.New(common.ErrInvalidSecondaryPublicKey)
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
