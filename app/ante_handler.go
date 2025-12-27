package app

import (
	"encoding/binary"
	"errors"

	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/ethereum/go-ethereum/crypto"
	EthereumK1 "github.com/ethereum/go-ethereum/crypto"
)

type HandlerOptions struct {
	ante.HandlerOptions
}

type SecondarySignatureVerificationDecorator struct {
	accountKeeper ante.AccountKeeper
}

func NewSecondarySignatureVerificationDecorator(accountKeeper ante.AccountKeeper) SecondarySignatureVerificationDecorator {
	return SecondarySignatureVerificationDecorator{
		accountKeeper: accountKeeper,
	}
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

		NewSecondarySignatureVerificationDecorator(options.AccountKeeper),
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
	secondSig, err := ParseMemo(tx)
	if err != nil {
		if err.Error() == ErrNoPrefix {
			next(ctx, tx, simulate)
		}
		return ctx, err
	}
	// Validate the signature structure
	if err := secondSig.Validate(); err != nil {
		ctx.Logger().Info("AnteHandle called, empty secondsig")
		return ctx, sdkerrors.ErrInvalidRequest
	}
	addr, err := GetAddr(tx)
	if err != nil {
		return ctx, err
	}
	acc := svd.accountKeeper.GetAccount(ctx, addr)
	if acc == nil {
		return ctx, sdkerrors.ErrUnknownAddress
	}
	seq := make([]byte, 8)
	binary.BigEndian.PutUint64(seq, acc.GetSequence())

	hsh := crypto.Keccak256([]byte(seq))

	// Verify the signature
	if !EthereumK1.VerifySignature(secondSig.PublicKey, hsh, secondSig.Signature) {
		ctx.Logger().Info("AnteHandle called,invalid signature")
		return ctx, fmt.Errorf("signature verification failed")
	}

	ctx.Logger().Info("AnteHandle called,tx valid")
	return next(ctx, tx, simulate)
}
