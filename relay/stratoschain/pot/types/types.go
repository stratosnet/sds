package types

import sdk "github.com/cosmos/cosmos-sdk/types"

type SingleNodeVolume struct {
	NodeAddress sdk.AccAddress `json:"node_address" yaml:"node_address"`
	Volume      sdk.Int        `json:"node_volume" yaml:"node_volume"` //uoz
}

// NewSingleNodeVolume creates a new Msg<Action> instance
func NewSingleNodeVolume(
	nodeAddress sdk.AccAddress,
	volume sdk.Int,
) SingleNodeVolume {
	return SingleNodeVolume{
		NodeAddress: nodeAddress,
		Volume:      volume,
	}
}

type MiningRewardParam struct {
	TotalMinedValveStart                sdk.Int `json:"total_mined_valve_start" yaml:"total_mined_valve_start"`
	TotalMinedValveEnd                  sdk.Int `json:"total_mined_valve_end" yaml:"total_mined_valve_end"`
	MiningReward                        sdk.Int `json:"mining_reward" yaml:"mining_reward"`
	BlockChainPercentageInTenThousand   sdk.Int `json:"block_chain_percentage_in_ten_thousand" yaml:"block_chain_percentage_in_ten_thousand"`
	ResourceNodePercentageInTenThousand sdk.Int `json:"resource_node_percentage_in_ten_thousand" yaml:"resource_node_percentage_in_ten_thousand"`
	MetaNodePercentageInTenThousand     sdk.Int `json:"meta_node_percentage_in_ten_thousand" yaml:"meta_node_percentage_in_ten_thousand"`
}

func NewMiningRewardParam(totalMinedValveStart sdk.Int, totalMinedValveEnd sdk.Int, miningReward sdk.Int,
	resourceNodePercentageInTenThousand sdk.Int, metaNodePercentageInTenThousand sdk.Int, blockChainPercentageInTenThousand sdk.Int) MiningRewardParam {
	return MiningRewardParam{
		TotalMinedValveStart:                totalMinedValveStart,
		TotalMinedValveEnd:                  totalMinedValveEnd,
		MiningReward:                        miningReward,
		BlockChainPercentageInTenThousand:   blockChainPercentageInTenThousand,
		ResourceNodePercentageInTenThousand: resourceNodePercentageInTenThousand,
		MetaNodePercentageInTenThousand:     metaNodePercentageInTenThousand,
	}
}
