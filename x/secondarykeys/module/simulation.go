package secondarykeys

import (
	"math/rand"

	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"

	secondarykeyssimulation "example/x/secondarykeys/simulation"
	"example/x/secondarykeys/types"
)

// GenerateGenesisState creates a randomized GenState of the module.
func (AppModule) GenerateGenesisState(simState *module.SimulationState) {
	accs := make([]string, len(simState.Accounts))
	for i, acc := range simState.Accounts {
		accs[i] = acc.Address.String()
	}
	secondarykeysGenesis := types.GenesisState{
		Params: types.DefaultParams(),
	}
	simState.GenState[types.ModuleName] = simState.Cdc.MustMarshalJSON(&secondarykeysGenesis)
}

// RegisterStoreDecoder registers a decoder.
func (am AppModule) RegisterStoreDecoder(_ simtypes.StoreDecoderRegistry) {}

// WeightedOperations returns the all the gov module operations with their respective weights.
func (am AppModule) WeightedOperations(simState module.SimulationState) []simtypes.WeightedOperation {
	operations := make([]simtypes.WeightedOperation, 0)
	const (
		opWeightMsgBroadcastData          = "op_weight_msg_secondarykeys"
		defaultWeightMsgBroadcastData int = 100
	)

	var weightMsgBroadcastData int
	simState.AppParams.GetOrGenerate(opWeightMsgBroadcastData, &weightMsgBroadcastData, nil,
		func(_ *rand.Rand) {
			weightMsgBroadcastData = defaultWeightMsgBroadcastData
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgBroadcastData,
		secondarykeyssimulation.SimulateMsgBroadcastData(am.authKeeper, am.bankKeeper, am.keeper, simState.TxConfig),
	))

	return operations
}

// ProposalMsgs returns msgs used for governance proposals for simulations.
func (am AppModule) ProposalMsgs(simState module.SimulationState) []simtypes.WeightedProposalMsg {
	return []simtypes.WeightedProposalMsg{}
}
