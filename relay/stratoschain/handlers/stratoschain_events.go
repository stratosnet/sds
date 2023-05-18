package handlers

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"time"

	"github.com/pkg/errors"

	coretypes "github.com/tendermint/tendermint/rpc/core/types"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	sdktx "github.com/cosmos/cosmos-sdk/types/tx"
	stakingTypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	pottypes "github.com/stratosnet/stratos-chain/x/pot/types"
	registertypes "github.com/stratosnet/stratos-chain/x/register/types"
	sdstypes "github.com/stratosnet/stratos-chain/x/sds/types"

	"github.com/stratosnet/sds/cmd/relayd/setting"
	"github.com/stratosnet/sds/msg/protos"
	relayTypes "github.com/stratosnet/sds/relay/types"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/crypto"
	"github.com/stratosnet/sds/utils/crypto/ed25519"
)

var Handlers map[string]func(coretypes.ResultEvent)
var cache *utils.AutoCleanMap // Cache with a TTL to make sure each event is only handled once

func init() {
	Handlers = make(map[string]func(coretypes.ResultEvent))
	Handlers[MSG_TYPE_CREATE_RESOURCE_NODE] = CreateResourceNodeMsgHandler()
	Handlers[MSG_TYPE_UPDATE_RESOURCE_NODE_STAKE] = UpdateResourceNodeStakeMsgHandler()
	Handlers[MSG_TYPE_REMOVE_RESOURCE_NODE] = UnbondingResourceNodeMsgHandler()
	//Handlers["complete_unbonding_resource_node"] = CompleteUnbondingResourceNodeMsgHandler()
	Handlers[MSG_TYPE_CREATE_META_NODE] = CreateMetaNodeMsgHandler()
	Handlers[MSG_TYPE_UPDATE_META_NODE_STAKE] = UpdateMetaNodeStakeMsgHandler()
	Handlers[MSG_TYPE_REMOVE_META_NODE] = UnbondingMetaNodeMsgHandler()
	Handlers[MSG_TYPE_WITHDRAWN_META_NODE_REG_STAKE] = WithdrawnStakeHandler()
	//Handlers["complete_unbonding_meta_node"] = CompleteUnbondingMetaNodeMsgHandler()
	Handlers[MSG_TYPE_META_NODE_REG_VOTE] = MetaNodeVoteMsgHandler()
	Handlers[MSG_TYPE_PREPAY] = PrepayMsgHandler()
	Handlers[MSG_TYPE_FILE_UPLOAD] = FileUploadMsgHandler()
	Handlers[MSG_TYPE_VOLUME_REPORT] = VolumeReportHandler()
	Handlers[MSG_TYPE_SLASHING_RESOURCE_NODE] = SlashingResourceNodeHandler()
	Handlers[MSG_TYPE_UPDATE_EFFECTIVE_STAKE] = UpdateEffectiveStakeHandler()

	cache = utils.NewAutoCleanMap(time.Minute)
}

func ProcessEvents(broadcastResponse sdktx.BroadcastTxResponse) map[string]coretypes.ResultEvent {
	response := broadcastResponse.TxResponse
	// Read the events from each msg in the log
	var events []map[string]string
	for _, msg := range response.Logs {
		msgMap := make(map[string]string)
		for _, stringEvent := range msg.Events {
			for _, attrib := range stringEvent.Attributes {
				msgMap[fmt.Sprintf("%v.%v", stringEvent.Type, attrib.Key)] = attrib.Value
			}
		}
		if len(msgMap) > 0 {
			events = append(events, msgMap)
		}
	}

	// Aggregate events by msg type
	aggregatedEvents := make(map[string]map[string][]string)
	for _, event := range events {
		typeStr := event["message.action"]
		currentMap := aggregatedEvents[typeStr]
		if currentMap == nil {
			currentMap = make(map[string][]string)
			currentMap["tx.hash"] = []string{response.TxHash}
		}

		for key, value := range event {
			switch key {
			case "message.action":
				continue
			default:
				currentMap[key] = append(currentMap[key], value)
			}
		}
		aggregatedEvents[typeStr] = currentMap
	}

	// Convert to coretypes.ResultEvent
	resultMap := make(map[string]coretypes.ResultEvent)
	for key, value := range aggregatedEvents {
		resultMap[key] = coretypes.ResultEvent{
			Query:  "",
			Data:   nil,
			Events: value,
		}
	}
	return resultMap
}

func CreateResourceNodeMsgHandler() func(event coretypes.ResultEvent) {
	return func(result coretypes.ResultEvent) {
		requiredAttributes := GetEventAttributes(registertypes.EventTypeCreateResourceNode,
			registertypes.AttributeKeyNetworkAddress,
			registertypes.AttributeKeyPubKey,
			registertypes.AttributeKeyOZoneLimitChanges,
			registertypes.AttributeKeyInitialStake,
		)

		processedEvents, txHash, initialEventCount := processEvents(result.Events, requiredAttributes)
		key := getCacheKey(requiredAttributes, result)
		if _, ok := cache.Load(key); ok {
			utils.DebugLogf("Event create_resource_node was already handled for tx [%v]. Ignoring...", txHash)
			return
		}
		cache.Store(key, true)

		req := &relayTypes.ActivatedPPReq{}
		for _, event := range processedEvents {
			p2pPubkey, err := processHexPubkey(event[GetEventAttribute(registertypes.EventTypeCreateResourceNode, registertypes.AttributeKeyPubKey)])
			if err != nil {
				utils.ErrorLog(err)
				continue
			}

			req.PPList = append(req.PPList, &protos.ReqActivatedPP{
				P2PAddress:        event[GetEventAttribute(registertypes.EventTypeCreateResourceNode, registertypes.AttributeKeyNetworkAddress)],
				P2PPubkey:         hex.EncodeToString(p2pPubkey.Bytes()),
				OzoneLimitChanges: event[GetEventAttribute(registertypes.EventTypeCreateResourceNode, registertypes.AttributeKeyOZoneLimitChanges)],
				TxHash:            txHash,
				InitialStake:      event[GetEventAttribute(registertypes.EventTypeCreateResourceNode, registertypes.AttributeKeyInitialStake)],
			})
		}

		if len(req.PPList) != initialEventCount {
			utils.ErrorLogf("activated PP message handler couldn't process all events (success: %v  missing_attribute: %v  invalid_attribute: %v",
				len(req.PPList), initialEventCount-len(processedEvents), len(processedEvents)-len(req.PPList))
		}
		if len(req.PPList) == 0 {
			return
		}

		err := postToSP("/pp/activated", req)
		if err != nil {
			utils.ErrorLog(err)
			return
		}
	}
}

func UpdateResourceNodeStakeMsgHandler() func(event coretypes.ResultEvent) {
	return func(result coretypes.ResultEvent) {
		requiredAttributes := GetEventAttributes(registertypes.EventTypeUpdateResourceNodeStake,
			registertypes.AttributeKeyNetworkAddress,
			registertypes.AttributeKeyOZoneLimitChanges,
			registertypes.AttributeKeyStakeDelta,
			registertypes.AttributeKeyCurrentStake,
			registertypes.AttributeKeyAvailableTokenBefore,
			registertypes.AttributeKeyAvailableTokenAfter,
		)

		processedEvents, txHash, initialEventCount := processEvents(result.Events, requiredAttributes)
		key := getCacheKey(requiredAttributes, result)
		if _, ok := cache.Load(key); ok {
			utils.DebugLogf("Event update_resource_node_stake was already handled for tx [%v]. Ignoring...", txHash)
			return
		}
		cache.Store(key, true)

		req := &relayTypes.UpdatedStakePPReq{}
		for _, event := range processedEvents {
			req.PPList = append(req.PPList, &protos.ReqUpdatedStakePP{
				P2PAddress:           event[GetEventAttribute(registertypes.EventTypeUpdateResourceNodeStake, registertypes.AttributeKeyNetworkAddress)],
				OzoneLimitChanges:    event[GetEventAttribute(registertypes.EventTypeUpdateResourceNodeStake, registertypes.AttributeKeyOZoneLimitChanges)],
				TxHash:               txHash,
				StakeDelta:           event[GetEventAttribute(registertypes.EventTypeUpdateResourceNodeStake, registertypes.AttributeKeyStakeDelta)],
				CurrentStake:         event[GetEventAttribute(registertypes.EventTypeUpdateResourceNodeStake, registertypes.AttributeKeyCurrentStake)],
				AvailableTokenBefore: event[GetEventAttribute(registertypes.EventTypeUpdateResourceNodeStake, registertypes.AttributeKeyAvailableTokenBefore)],
				AvailableTokenAfter:  event[GetEventAttribute(registertypes.EventTypeUpdateResourceNodeStake, registertypes.AttributeKeyAvailableTokenAfter)],
			})
		}

		if len(req.PPList) != initialEventCount {
			utils.ErrorLogf("updatedStake PP message handler couldn't process all events (success: %v  missing_attribute: %v  invalid_attribute: %v",
				len(req.PPList), initialEventCount-len(processedEvents), len(processedEvents)-len(req.PPList))
		}
		if len(req.PPList) == 0 {
			return
		}

		err := postToSP("/pp/updatedStake", req)
		if err != nil {
			utils.ErrorLog(err)
			return
		}
	}
}

func UnbondingResourceNodeMsgHandler() func(event coretypes.ResultEvent) {
	return func(result coretypes.ResultEvent) {
		requiredAttributes := GetEventAttributes(registertypes.EventTypeUnbondingResourceNode,
			registertypes.AttributeKeyResourceNode,
			registertypes.AttributeKeyUnbondingMatureTime,
			registertypes.AttributeKeyStakeToRemove,
		)

		processedEvents, txHash, initialEventCount := processEvents(result.Events, requiredAttributes)
		key := getCacheKey(requiredAttributes, result)
		if _, ok := cache.Load(key); ok {
			utils.DebugLogf("Event unbonding_resource_node was already handled for tx [%v]. Ignoring...", txHash)
			return
		}
		cache.Store(key, true)

		req := &relayTypes.UnbondingPPReq{}
		for _, event := range processedEvents {
			req.PPList = append(req.PPList, &protos.ReqUnbondingPP{
				P2PAddress:          event[GetEventAttribute(registertypes.EventTypeUnbondingResourceNode, registertypes.AttributeKeyResourceNode)],
				UnbondingMatureTime: event[GetEventAttribute(registertypes.EventTypeUnbondingResourceNode, registertypes.AttributeKeyUnbondingMatureTime)],
				TxHash:              txHash,
				StakeToRemove:       event[GetEventAttribute(registertypes.EventTypeUnbondingResourceNode, registertypes.AttributeKeyStakeToRemove)],
			})
		}

		if len(req.PPList) != initialEventCount {
			utils.ErrorLogf("unbonding PP message handler couldn't process all events (success: %v  missing_attribute: %v  invalid_attribute: %v",
				len(req.PPList), initialEventCount-len(processedEvents), len(processedEvents)-len(req.PPList))
		}
		if len(req.PPList) == 0 {
			return
		}

		err := postToSP("/pp/unbonding", req)
		if err != nil {
			utils.ErrorLog(err)
			return
		}
	}
}

func CompleteUnbondingResourceNodeMsgHandler() func(event coretypes.ResultEvent) {
	return func(result coretypes.ResultEvent) {
		requiredAttributes := GetEventAttributes(registertypes.EventTypeCompleteUnbondingResourceNode,
			registertypes.AttributeKeyNetworkAddress,
		)

		processedEvents, txHash, initialEventCount := processEvents(result.Events, requiredAttributes)
		key := getCacheKey(requiredAttributes, result)
		if _, ok := cache.Load(key); ok {
			utils.DebugLogf("Event complete_unbonding_resource_node was already handled for tx [%v]. Ignoring...", txHash)
			return
		}
		cache.Store(key, true)

		req := &relayTypes.DeactivatedPPReq{}
		for _, event := range processedEvents {
			req.PPList = append(req.PPList, &protos.ReqDeactivatedPP{
				P2PAddress: event[GetEventAttribute(registertypes.EventTypeCompleteUnbondingResourceNode, registertypes.AttributeKeyNetworkAddress)],
				TxHash:     txHash,
			})
		}

		if len(req.PPList) != initialEventCount {
			utils.ErrorLogf("Complete unbonding PP message handler couldn't process all events (success: %v  missing_attribute: %v  invalid_attribute: %v",
				len(req.PPList), initialEventCount-len(processedEvents), len(processedEvents)-len(req.PPList))
		}
		if len(req.PPList) == 0 {
			return
		}

		err := postToSP("/pp/deactivated", req)
		if err != nil {
			utils.ErrorLog(err)
			return
		}
	}
}

func CreateMetaNodeMsgHandler() func(event coretypes.ResultEvent) {
	return func(result coretypes.ResultEvent) {
		// TODO
		utils.Logf("%+v", result)
	}
}

func UpdateMetaNodeStakeMsgHandler() func(event coretypes.ResultEvent) {
	return func(result coretypes.ResultEvent) {
		requiredAttributes := GetEventAttributes(registertypes.EventTypeUpdateMetaNodeStake,
			registertypes.AttributeKeyNetworkAddress,
			registertypes.AttributeKeyOZoneLimitChanges,
			registertypes.AttributeKeyIncrStake,
		)

		processedEvents, txHash, initialEventCount := processEvents(result.Events, requiredAttributes)
		key := getCacheKey(requiredAttributes, result)
		if _, ok := cache.Load(key); ok {
			utils.DebugLogf("Event update_meta_node_stake was already handled for tx [%v]. Ignoring...", txHash)
			return
		}
		cache.Store(key, true)

		req := &relayTypes.UpdatedStakeSPReq{}
		for _, event := range processedEvents {
			req.SPList = append(req.SPList, &protos.ReqUpdatedStakeSP{
				P2PAddress:        event[GetEventAttribute(registertypes.EventTypeUpdateMetaNodeStake, registertypes.AttributeKeyNetworkAddress)],
				OzoneLimitChanges: event[GetEventAttribute(registertypes.EventTypeUpdateMetaNodeStake, registertypes.AttributeKeyOZoneLimitChanges)],
				IncrStake:         event[GetEventAttribute(registertypes.EventTypeUpdateMetaNodeStake, registertypes.AttributeKeyIncrStake)],
				TxHash:            txHash,
			})
		}

		if len(req.SPList) != initialEventCount {
			utils.ErrorLogf("Updated SP stake message handler couldn't process all events (success: %v  missing_attribute: %v  invalid_attribute: %v",
				len(req.SPList), initialEventCount-len(processedEvents), len(processedEvents)-len(req.SPList))
		}
		if len(req.SPList) == 0 {
			return
		}

		err := postToSP("/chain/updatedStake", req)
		if err != nil {
			utils.ErrorLog(err)
			return
		}
	}
}

func UnbondingMetaNodeMsgHandler() func(event coretypes.ResultEvent) {
	return func(result coretypes.ResultEvent) {
		// TODO
		utils.Logf("%+v", result)
	}
}

func WithdrawnStakeHandler() func(event coretypes.ResultEvent) {
	return func(result coretypes.ResultEvent) {
		requiredAttributes := GetEventAttributes(registertypes.EventTypeWithdrawMetaNodeRegistrationStake,
			registertypes.AttributeKeyNetworkAddress,
			registertypes.AttributeKeyUnbondingMatureTime,
		)

		processedEvents, txHash, initialEventCount := processEvents(result.Events, requiredAttributes)
		key := getCacheKey(requiredAttributes, result)
		if _, ok := cache.Load(key); ok {
			utils.DebugLogf("Event withdraw_meta_node_reg_stake was already handled for tx [%v]. Ignoring...", txHash)
			return
		}
		cache.Store(key, true)

		req := &relayTypes.WithdrawnStakeSPReq{}
		for _, event := range processedEvents {
			networkAddr := event[GetEventAttribute(registertypes.EventTypeWithdrawMetaNodeRegistrationStake, registertypes.AttributeKeyNetworkAddress)]
			unbondingMatureTime := event[GetEventAttribute(registertypes.EventTypeWithdrawMetaNodeRegistrationStake, registertypes.AttributeKeyUnbondingMatureTime)]

			if len(networkAddr) == 0 || len(unbondingMatureTime) == 0 {
				continue
			}

			req.SPList = append(req.SPList, &protos.ReqWithdrawnStakeSP{
				P2PAddress:          networkAddr,
				UnbondingMatureTime: unbondingMatureTime,
				TxHash:              txHash,
			})
		}

		if len(req.SPList) != initialEventCount {
			utils.ErrorLogf("Indexing node vote message handler couldn't process all events (success: %v  missing_attribute: %v  invalid_attribute: %v",
				len(req.SPList), initialEventCount-len(processedEvents), len(processedEvents)-len(req.SPList))
		}
		if len(req.SPList) == 0 {
			return
		}

		err := postToSP("/chain/withdrawn", req)
		if err != nil {
			utils.ErrorLog(err)
			return
		}
	}

}

func CompleteUnbondingMetaNodeMsgHandler() func(event coretypes.ResultEvent) {
	return func(result coretypes.ResultEvent) {
		// TODO
		utils.Logf("%+v", result)
	}
}

func MetaNodeVoteMsgHandler() func(event coretypes.ResultEvent) {
	return func(result coretypes.ResultEvent) {
		requiredAttributes := GetEventAttributes(registertypes.EventTypeMetaNodeRegistrationVote,
			registertypes.AttributeKeyCandidateNetworkAddress,
			registertypes.AttributeKeyCandidateStatus,
		)

		processedEvents, txHash, initialEventCount := processEvents(result.Events, requiredAttributes)
		key := getCacheKey(requiredAttributes, result)
		if _, ok := cache.Load(key); ok {
			utils.DebugLogf("Event meta_node_reg_vote was already handled for tx [%v]. Ignoring...", txHash)
			return
		}
		cache.Store(key, true)

		req := &relayTypes.ActivatedSPReq{}
		for _, event := range processedEvents {
			candidateNetworkAddr := event[GetEventAttribute(registertypes.EventTypeMetaNodeRegistrationVote, registertypes.AttributeKeyCandidateNetworkAddress)]

			if event[GetEventAttribute(registertypes.EventTypeMetaNodeRegistrationVote, registertypes.AttributeKeyCandidateStatus)] != stakingTypes.BondStatusBonded {
				utils.DebugLogf("Indexing node vote handler: The candidate [%v] needs more votes before being considered active", candidateNetworkAddr)
				continue
			}

			req.SPList = append(req.SPList, &protos.ReqActivatedSP{
				P2PAddress: candidateNetworkAddr,
				TxHash:     txHash,
			})
		}

		if len(req.SPList) != initialEventCount {
			utils.ErrorLogf("Indexing node vote message handler couldn't process all events (success: %v  missing_attribute: %v  invalid_attribute: %v",
				len(req.SPList), initialEventCount-len(processedEvents), len(processedEvents)-len(req.SPList))
		}
		if len(req.SPList) == 0 {
			return
		}

		err := postToSP("/chain/activated", req)
		if err != nil {
			utils.ErrorLog(err)
			return
		}
	}
}

func PrepayMsgHandler() func(event coretypes.ResultEvent) {
	return func(result coretypes.ResultEvent) {
		utils.Logf("%+v", result)
		requiredAttributes := GetEventAttributes(sdstypes.EventTypePrepay,
			sdktypes.AttributeKeySender,
			sdstypes.AttributeKeyBeneficiary,
			sdstypes.AttributeKeyPurchasedNoz,
		)

		processedEvents, txHash, initialEventCount := processEvents(result.Events, requiredAttributes)
		key := getCacheKey(requiredAttributes, result)
		if _, ok := cache.Load(key); ok {
			utils.DebugLogf("Event Prepay was already handled for tx [%v]. Ignoring...", txHash)
			return
		}
		cache.Store(key, true)

		req := &relayTypes.PrepaidReq{}
		for _, event := range processedEvents {
			req.WalletList = append(req.WalletList, &protos.ReqPrepaid{
				WalletAddress: event[GetEventAttribute(sdstypes.EventTypePrepay, sdstypes.AttributeKeyBeneficiary)],
				PurchasedUoz:  event[GetEventAttribute(sdstypes.EventTypePrepay, sdstypes.AttributeKeyPurchasedNoz)],
				TxHash:        txHash,
			})
		}

		if len(req.WalletList) != initialEventCount {
			utils.ErrorLogf("Prepay message handler couldn't process all events (success: %v  missing_attribute: %v  invalid_attribute: %v",
				len(req.WalletList), initialEventCount-len(processedEvents), len(processedEvents)-len(req.WalletList))
		}
		if len(req.WalletList) == 0 {
			return
		}

		err := postToSP("/pp/prepaid", req)
		if err != nil {
			utils.ErrorLog(err)
			return
		}
	}
}

func FileUploadMsgHandler() func(event coretypes.ResultEvent) {
	return func(result coretypes.ResultEvent) {
		requiredAttributes := GetEventAttributes(sdstypes.EventTypeFileUpload,
			sdstypes.AttributeKeyReporter,
			sdstypes.AttributeKeyUploader,
			sdstypes.AttributeKeyFileHash,
		)

		processedEvents, txHash, initialEventCount := processEvents(result.Events, requiredAttributes)
		key := getCacheKey(requiredAttributes, result)
		if _, ok := cache.Load(key); ok {
			utils.DebugLogf("Event FileUpload was already handled for tx [%v]. Ignoring...", txHash)
			return
		}
		cache.Store(key, true)

		req := &relayTypes.FileUploadedReq{}
		for _, event := range processedEvents {
			req.UploadList = append(req.UploadList, &protos.Uploaded{
				ReporterAddress: event[GetEventAttribute(sdstypes.EventTypeFileUpload, sdstypes.AttributeKeyReporter)],
				UploaderAddress: event[GetEventAttribute(sdstypes.EventTypeFileUpload, sdstypes.AttributeKeyUploader)],
				FileHash:        event[GetEventAttribute(sdstypes.EventTypeFileUpload, sdstypes.AttributeKeyFileHash)],
				TxHash:          txHash,
			})
		}

		if len(req.UploadList) != initialEventCount {
			utils.ErrorLogf("File upload message handler couldn't process all events (success: %v  missing_attribute: %v  invalid_attribute: %v",
				len(req.UploadList), initialEventCount-len(processedEvents), len(processedEvents)-len(req.UploadList))
		}
		if len(req.UploadList) == 0 {
			return
		}

		err := postToSP("/pp/uploaded", req)
		if err != nil {
			utils.ErrorLog(err)
			return
		}
	}
}

func VolumeReportHandler() func(event coretypes.ResultEvent) {
	return func(result coretypes.ResultEvent) {
		requiredAttributes := GetEventAttributes(pottypes.EventTypeVolumeReport,
			pottypes.AttributeKeyEpoch,
		)

		processedEvents, txHash, initialEventCount := processEvents(result.Events, requiredAttributes)
		key := getCacheKey(requiredAttributes, result)
		if _, ok := cache.Load(key); ok {
			utils.DebugLogf("Event volume_report was already handled for tx [%v]. Ignoring...", txHash)
			return
		}
		cache.Store(key, true)

		req := &relayTypes.VolumeReportedReq{}
		for _, event := range processedEvents {
			req.Epochs = append(req.Epochs, event[GetEventAttribute(pottypes.EventTypeVolumeReport, pottypes.AttributeKeyEpoch)])
		}

		if len(req.Epochs) != initialEventCount {
			utils.ErrorLogf("Volume report message handler couldn't process all events (success: %v  missing_attribute: %v  invalid_attribute: %v",
				len(req.Epochs), initialEventCount-len(processedEvents), len(processedEvents)-len(req.Epochs))
		}
		if len(req.Epochs) == 0 {
			return
		}

		err := postToSP("/volume/reported", req)
		if err != nil {
			utils.ErrorLog(err)
			return
		}
	}
}

func SlashingResourceNodeHandler() func(event coretypes.ResultEvent) {
	return func(result coretypes.ResultEvent) {
		requiredAttributes := GetEventAttributes(pottypes.EventTypeSlashing,
			pottypes.AttributeKeyNodeP2PAddress,
			pottypes.AttributeKeyNodeSuspended,
			pottypes.AttributeKeyAmount,
		)

		processedEvents, txHash, initialEventCount := processEvents(result.Events, requiredAttributes)
		key := getCacheKey(requiredAttributes, result)
		if _, ok := cache.Load(key); ok {
			utils.DebugLogf("Event slashing was already handled for tx [%v]. Ignoring...", txHash)
			return
		}
		cache.Store(key, true)
		var slashedPPs []relayTypes.SlashedPP
		for _, event := range processedEvents {
			suspended, err := strconv.ParseBool(event[GetEventAttribute(pottypes.EventTypeSlashing, pottypes.AttributeKeyNodeSuspended)])
			if err != nil {
				utils.DebugLog("Invalid suspended boolean in the slashing message from stratos-chain", err)
				continue
			}
			slashedAmt, ok := new(big.Int).SetString(event[GetEventAttribute(pottypes.EventTypeSlashing, pottypes.AttributeKeyAmount)], 10)
			if !ok {
				utils.DebugLog("Invalid slashed amount in big integer in the slashing message from stratos-chain")
				continue
			}
			slashedPP := relayTypes.SlashedPP{
				P2PAddress: event[GetEventAttribute(pottypes.EventTypeSlashing, pottypes.AttributeKeyNodeP2PAddress)],
				QueryFirst: false,
				Suspended:  suspended,
				SlashedAmt: slashedAmt,
			}
			slashedPPs = append(slashedPPs, slashedPP)
		}

		if len(slashedPPs) != initialEventCount {
			utils.ErrorLogf("slashing message handler couldn't process all events (success: %v  missing_attribute: %v  invalid_attribute: %v",
				len(slashedPPs), initialEventCount-len(processedEvents), len(processedEvents)-len(slashedPPs))
		}
		if len(slashedPPs) == 0 {
			return
		}

		req := relayTypes.SlashedPPReq{
			PPList: slashedPPs,
			TxHash: txHash,
		}
		err := postToSP("/pp/slashed", req)
		if err != nil {
			utils.ErrorLog(err)
			return
		}
	}
}

func UpdateEffectiveStakeHandler() func(event coretypes.ResultEvent) {
	return func(result coretypes.ResultEvent) {
		requiredAttributes := GetEventAttributes(registertypes.EventTypeUpdateEffectiveStake,
			registertypes.AttributeKeyNetworkAddress,
			registertypes.AttributeKeyIsUnsuspended,
			registertypes.AttributeKeyEffectiveStakeAfter,
		)

		processedEvents, txHash, initialEventCount := processEvents(result.Events, requiredAttributes)
		key := getCacheKey(requiredAttributes, result)
		if _, ok := cache.Load(key); ok {
			utils.DebugLogf("Event update_effective_stake was already handled for tx [%v]. Ignoring...", txHash)
			return
		}
		cache.Store(key, true)
		var updatedPPs []relayTypes.UpdatedEffectiveStakePP
		for _, event := range processedEvents {
			isUnsuspendedDuringUpdate, err := strconv.ParseBool(event[GetEventAttribute(registertypes.EventTypeUpdateEffectiveStake, registertypes.AttributeKeyIsUnsuspended)])
			if err != nil {
				utils.DebugLog("Invalid is_unsuspended boolean in the update_effective_stake message from stratos-chain", err)
				continue
			}

			effectiveStakeAfter, ok := new(big.Int).SetString(event[GetEventAttribute(registertypes.EventTypeUpdateEffectiveStake, registertypes.AttributeKeyEffectiveStakeAfter)], 10)
			if !ok {
				utils.DebugLog("Invalid effective_stake_after in big integer in the update_effective_stake message from stratos-chain")
				continue
			}
			utils.DebugLogf("network_address: %v, isUnsuspendedDuringUpdate is %v, effectiveStakeAfter: %v",
				event[GetEventAttribute(registertypes.EventTypeUpdateEffectiveStake, registertypes.AttributeKeyNetworkAddress)], isUnsuspendedDuringUpdate, effectiveStakeAfter.String())

			if !isUnsuspendedDuringUpdate {
				// only msg for unsuspended node will be transferred to SP
				continue
			}

			updatedPP := relayTypes.UpdatedEffectiveStakePP{
				P2PAddress:                event[GetEventAttribute(registertypes.EventTypeUpdateEffectiveStake, registertypes.AttributeKeyNetworkAddress)],
				IsUnsuspendedDuringUpdate: isUnsuspendedDuringUpdate,
				EffectiveStakeAfter:       effectiveStakeAfter,
			}
			updatedPPs = append(updatedPPs, updatedPP)
		}

		if len(updatedPPs) > 0 {
			utils.ErrorLogf("updatedEffectiveStake message handler is processing events to unsuspend pp "+
				"(ToBeUnsuspended Events: %v, Invalid Events: %v, Total : %v",
				len(updatedPPs), initialEventCount-len(processedEvents), initialEventCount)
		}
		if len(updatedPPs) == 0 {
			return
		}

		req := relayTypes.UpdatedEffectiveStakePPReq{
			PPList: updatedPPs,
			TxHash: txHash,
		}
		err := postToSP("/pp/updatedEffectiveStake", req)
		if err != nil {
			utils.ErrorLog(err)
			return
		}
	}
}

func postToSP(endpoint string, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return errors.New("Error when trying to marshal data to json: " + err.Error())
	}

	url := utils.Url{
		Scheme: "http",
		Host:   setting.Config.SDS.NetworkAddress,
		Port:   setting.Config.SDS.ApiPort,
		Path:   endpoint,
	}

	resp, err := http.Post(url.String(true, true, true, false), "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return errors.New("Error when calling " + endpoint + " endpoint in SP node: " + err.Error())
	}

	var res map[string]interface{}
	if err = json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return err
	}

	utils.Log(endpoint+" endpoint response from SP node", resp.StatusCode, res["Msg"])
	return nil
}

func processEvents(eventsMap map[string][]string, attributesRequired []string) (processedEvents []map[string]string, txHash string, totalEventCount int) {
	if len(attributesRequired) < 1 {
		return nil, "", 0
	}

	// Get tx hash
	if len(eventsMap["tx.hash"]) > 0 {
		txHash = eventsMap["tx.hash"][0]
	}

	// Count how many events are valid (all required attributes are present)
	validEventCount := len(eventsMap[attributesRequired[0]])
	for _, attribute := range attributesRequired {
		numberOfEvents := len(eventsMap[attribute])
		if numberOfEvents > totalEventCount {
			totalEventCount = numberOfEvents
		}
		if numberOfEvents < validEventCount {
			validEventCount = numberOfEvents
		}
	}

	// Separate the events map into an individual map for each valid event
	for i := 0; i < validEventCount; i++ {
		processedEvent := make(map[string]string)
		for _, attribute := range attributesRequired {
			processedEvent[attribute] = eventsMap[attribute][i]
		}
		processedEvents = append(processedEvents, processedEvent)
	}
	return
}

func processHexPubkey(attribute string) (cryptotypes.PubKey, error) {
	p2pPubkeyRaw, err := hex.DecodeString(attribute)
	if err != nil {
		return nil, errors.Wrap(err, "Error when trying to decode P2P pubkey hex")
	}
	p2pPubkey := ed25519.PubKeyBytesToSdkPubKey(p2pPubkeyRaw)

	return p2pPubkey, nil
}

func getCacheKey(requiredAttributes []string, result coretypes.ResultEvent) string {
	rawKey := ""
	if len(result.Events["tx.hash"]) > 0 {
		rawKey = result.Events["tx.hash"][0]
	}

	for _, attribute := range requiredAttributes {
		rawKey += attribute
		for _, value := range result.Events[attribute] {
			rawKey += value
		}
	}
	hash := crypto.Keccak256([]byte(rawKey))
	return string(hash)
}
