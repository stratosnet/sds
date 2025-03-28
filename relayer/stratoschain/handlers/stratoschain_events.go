package handlers

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"net/http"
	"strconv"
	"time"

	abciv1beta1 "cosmossdk.io/api/cosmos/base/abci/v1beta1"
	stakingv1beta1 "cosmossdk.io/api/cosmos/staking/v1beta1"
	abcitypes "github.com/cometbft/cometbft/abci/types"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	comettypes "github.com/cometbft/cometbft/types"
	"github.com/pkg/errors"

	"github.com/stratosnet/sds/framework/crypto"
	"github.com/stratosnet/sds/framework/crypto/ed25519"
	fwcryptotypes "github.com/stratosnet/sds/framework/crypto/types"
	"github.com/stratosnet/sds/framework/utils"
	"github.com/stratosnet/sds/relayer/cmd/relayd/setting"
	"github.com/stratosnet/sds/relayer/stratoschain/types"
	"github.com/stratosnet/sds/sds-msg/protos"
	"github.com/stratosnet/sds/sds-msg/relay"
)

var Handlers map[string]func(coretypes.ResultEvent)
var cache *utils.AutoCleanMap // Cache with a TTL to make sure each event is only handled once

func init() {
	Handlers = make(map[string]func(coretypes.ResultEvent))
	Handlers[types.MSG_TYPE_CREATE_RESOURCE_NODE] = CreateResourceNodeMsgHandler()
	Handlers[types.MSG_TYPE_UPDATE_RESOURCE_NODE] = UpdateResourceNodeMsgHandler()
	Handlers[types.MSG_TYPE_UPDATE_RESOURCE_NODE_DEPOSIT] = UpdateResourceNodeDepositMsgHandler()
	Handlers[types.MSG_TYPE_REMOVE_RESOURCE_NODE] = UnbondingResourceNodeMsgHandler()
	//Handlers["complete_unbonding_resource_node"] = CompleteUnbondingResourceNodeMsgHandler()
	Handlers[types.MSG_TYPE_CREATE_META_NODE] = CreateMetaNodeMsgHandler()
	Handlers[types.MSG_TYPE_UPDATE_META_NODE_DEPOSIT] = UpdateMetaNodeDepositMsgHandler()
	Handlers[types.MSG_TYPE_REMOVE_META_NODE] = UnbondingMetaNodeMsgHandler()
	Handlers[types.MSG_TYPE_KICK_META_NODE_VOTE] = UnbondingMetaNodeMsgHandler()
	//Handlers["complete_unbonding_meta_node"] = CompleteUnbondingMetaNodeMsgHandler()
	Handlers[types.MSG_TYPE_META_NODE_REG_VOTE] = MetaNodeVoteMsgHandler()
	Handlers[types.MSG_TYPE_PREPAY] = PrepayMsgHandler()
	Handlers[types.MSG_TYPE_FILE_UPLOAD] = FileUploadMsgHandler()
	Handlers[types.MSG_TYPE_VOLUME_REPORT] = VolumeReportHandler()
	Handlers[types.MSG_TYPE_SLASHING_RESOURCE_NODE] = SlashingResourceNodeHandler()
	Handlers[types.MSG_TYPE_UPDATE_EFFECTIVE_DEPOSIT] = UpdateEffectiveDepositHandler()
	Handlers[types.MSG_TYPE_EVM_TX] = EvmTxHandler()

	cache = utils.NewAutoCleanMap(time.Minute)
}

func ExtractEventsFromTxResponse(response *abciv1beta1.TxResponse) []coretypes.ResultEvent {
	// Read the events from each msg in the log
	var eventsPerMsg [][]abcitypes.Event
	for _, msg := range response.Logs {
		var events []abcitypes.Event
		for _, stringEvent := range msg.Events {
			var attributes []abcitypes.EventAttribute
			for _, attrib := range stringEvent.Attributes {
				attributes = append(attributes, abcitypes.EventAttribute{
					Key:   attrib.Key,
					Value: attrib.Value,
				})
			}
			events = append(events, abcitypes.Event{
				Type:       stringEvent.Type_,
				Attributes: attributes,
			})
		}
		if len(events) > 0 {
			eventsPerMsg = append(eventsPerMsg, events)
		}
	}

	txHashEvent := make(map[string][]string)
	txHashEvent["tx.hash"] = []string{response.Txhash}

	// Convert to coretypes.ResultEvent
	var resultEvents []coretypes.ResultEvent
	for _, event := range eventsPerMsg {
		resultEvents = append(resultEvents, coretypes.ResultEvent{
			Query: "",
			Data: comettypes.EventDataTx{
				TxResult: abcitypes.TxResult{
					Height: response.Height,
					Result: abcitypes.ResponseDeliverTx{
						Code:      response.Code,
						Info:      response.Info,
						GasWanted: response.GasWanted,
						GasUsed:   response.GasUsed,
						Events:    event,
						Codespace: response.Codespace,
					},
				},
			},
			Events: txHashEvent,
		})
	}
	return resultEvents
}

func GetMsgType(result coretypes.ResultEvent) string {
	eventDataTx, ok := result.Data.(comettypes.EventDataTx)
	if !ok {
		return ""
	}
	// Find the first message.action attribute
	for _, event := range eventDataTx.Result.Events {
		if event.Type != "message" {
			continue
		}
		for _, attribute := range event.Attributes {
			if attribute.Key == "action" {
				return attribute.Value
			}
		}
	}
	return ""
}

func CreateResourceNodeMsgHandler() func(event coretypes.ResultEvent) {
	return func(result coretypes.ResultEvent) {
		txHash := getTxHash(result)
		eventDataTx, ok := result.Data.(comettypes.EventDataTx)
		if !ok {
			utils.ErrorLogf("result data is the wrong type in CreateResourceNodeMsgHandler: %T", result.Data)
			return
		}
		requiredAttributes := []EventAttribute{
			{EventTypeCreateResourceNode, AttributeKeyNetworkAddress},
			{EventTypeCreateResourceNode, AttributeKeyPubKey},
			{EventTypeCreateResourceNode, AttributeKeyOZoneLimitChanges},
			{EventTypeCreateResourceNode, AttributeKeyInitialDeposit},
		}
		processedEvents, initialEventCount := processEvents(eventDataTx.Result.Events, types.MSG_TYPE_CREATE_RESOURCE_NODE, requiredAttributes)

		key := getCacheKey(requiredAttributes, processedEvents, txHash)
		if _, ok := cache.Load(key); ok {
			utils.DebugLogf("Event create_resource_node was already handled for tx [%v]. Ignoring...", txHash)
			return
		}
		cache.Store(key, true)

		req := &relay.ActivatedPPReq{}
		for _, event := range processedEvents {
			p2pPubKey, err := processHexPubkey(event[EventAttribute{EventTypeCreateResourceNode, AttributeKeyPubKey}])
			if err != nil {
				utils.ErrorLog(err)
				continue
			}

			req.PPList = append(req.PPList, &protos.ReqActivatedPP{
				P2PAddress:        event[EventAttribute{EventTypeCreateResourceNode, AttributeKeyNetworkAddress}],
				P2PPubkey:         hex.EncodeToString(p2pPubKey.Bytes()),
				OzoneLimitChanges: event[EventAttribute{EventTypeCreateResourceNode, AttributeKeyOZoneLimitChanges}],
				TxHash:            txHash,
				InitialDeposit:    event[EventAttribute{EventTypeCreateResourceNode, AttributeKeyInitialDeposit}],
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

func UpdateResourceNodeMsgHandler() func(event coretypes.ResultEvent) {
	return func(result coretypes.ResultEvent) {
		txHash := getTxHash(result)
		eventDataTx, ok := result.Data.(comettypes.EventDataTx)
		if !ok {
			utils.ErrorLogf("result data is the wrong type in UpdateResourceNodeMsgHandler: %T", result.Data)
			return
		}
		requiredAttributes := []EventAttribute{
			{EventTypeUpdateResourceNode, AttributeKeySender},
			{EventTypeUpdateResourceNode, AttributeKeyNetworkAddress},
			{EventTypeUpdateResourceNode, AttributeKeyBeneficiaryAddress},
		}
		processedEvents, initialEventCount := processEvents(eventDataTx.Result.Events, types.MSG_TYPE_UPDATE_RESOURCE_NODE, requiredAttributes)

		key := getCacheKey(requiredAttributes, processedEvents, txHash)
		if _, ok := cache.Load(key); ok {
			utils.DebugLogf("Event update_resource_node was already handled for tx [%v]. Ignoring...", txHash)
			return
		}
		cache.Store(key, true)

		req := &relay.UpdatePPBeneficiaryAddrReq{}
		for _, event := range processedEvents {
			req.PPList = append(req.PPList, &protos.ReqUpdatePPBeneficiaryAddr{
				P2PAddress:         event[EventAttribute{EventTypeUpdateResourceNode, AttributeKeyNetworkAddress}],
				BeneficiaryAddress: event[EventAttribute{EventTypeUpdateResourceNode, AttributeKeyBeneficiaryAddress}],
			})
		}

		if len(req.PPList) != initialEventCount {
			utils.ErrorLogf("updatedInfo PP message handler couldn't process all events (success: %v  missing_attribute: %v  invalid_attribute: %v",
				len(req.PPList), initialEventCount-len(processedEvents), len(processedEvents)-len(req.PPList))
		}
		if len(req.PPList) == 0 {
			return
		}

		err := postToSP("/pp/updateBeneficiaryAddress", req)
		if err != nil {
			utils.ErrorLog(err)
			return
		}
	}
}

func UpdateResourceNodeDepositMsgHandler() func(event coretypes.ResultEvent) {
	return func(result coretypes.ResultEvent) {
		txHash := getTxHash(result)
		eventDataTx, ok := result.Data.(comettypes.EventDataTx)
		if !ok {
			utils.ErrorLogf("result data is the wrong type in UpdateResourceNodeDepositMsgHandler: %T", result.Data)
			return
		}
		requiredAttributes := []EventAttribute{
			{EventTypeUpdateResourceNodeDeposit, AttributeKeyNetworkAddress},
			{EventTypeUpdateResourceNodeDeposit, AttributeKeyOZoneLimitChanges},
			{EventTypeUpdateResourceNodeDeposit, AttributeKeyDepositDelta},
			{EventTypeUpdateResourceNodeDeposit, AttributeKeyCurrentDeposit},
			{EventTypeUpdateResourceNodeDeposit, AttributeKeyAvailableTokenBefore},
			{EventTypeUpdateResourceNodeDeposit, AttributeKeyAvailableTokenAfter},
		}
		processedEvents, initialEventCount := processEvents(eventDataTx.Result.Events, types.MSG_TYPE_UPDATE_RESOURCE_NODE_DEPOSIT, requiredAttributes)

		key := getCacheKey(requiredAttributes, processedEvents, txHash)
		if _, ok := cache.Load(key); ok {
			utils.DebugLogf("Event update_resource_node_deposit was already handled for tx [%v]. Ignoring...", txHash)
			return
		}
		cache.Store(key, true)

		req := &relay.UpdatedDepositPPReq{}
		for _, event := range processedEvents {
			req.PPList = append(req.PPList, &protos.ReqUpdatedDepositPP{
				P2PAddress:           event[EventAttribute{EventTypeUpdateResourceNodeDeposit, AttributeKeyNetworkAddress}],
				OzoneLimitChanges:    event[EventAttribute{EventTypeUpdateResourceNodeDeposit, AttributeKeyOZoneLimitChanges}],
				TxHash:               txHash,
				DepositDelta:         event[EventAttribute{EventTypeUpdateResourceNodeDeposit, AttributeKeyDepositDelta}],
				CurrentDeposit:       event[EventAttribute{EventTypeUpdateResourceNodeDeposit, AttributeKeyCurrentDeposit}],
				AvailableTokenBefore: event[EventAttribute{EventTypeUpdateResourceNodeDeposit, AttributeKeyAvailableTokenBefore}],
				AvailableTokenAfter:  event[EventAttribute{EventTypeUpdateResourceNodeDeposit, AttributeKeyAvailableTokenAfter}],
			})
		}

		if len(req.PPList) != initialEventCount {
			utils.ErrorLogf("updatedDeposit PP message handler couldn't process all events (success: %v  missing_attribute: %v  invalid_attribute: %v",
				len(req.PPList), initialEventCount-len(processedEvents), len(processedEvents)-len(req.PPList))
		}
		if len(req.PPList) == 0 {
			return
		}

		err := postToSP("/pp/updatedDeposit", req)
		if err != nil {
			utils.ErrorLog(err)
			return
		}
	}
}

func UnbondingResourceNodeMsgHandler() func(event coretypes.ResultEvent) {
	return func(result coretypes.ResultEvent) {
		txHash := getTxHash(result)
		eventDataTx, ok := result.Data.(comettypes.EventDataTx)
		if !ok {
			utils.ErrorLogf("result data is the wrong type in UnbondingResourceNodeMsgHandler: %T", result.Data)
			return
		}
		requiredAttributes := []EventAttribute{
			{EventTypeUnbondingResourceNode, AttributeKeyResourceNode},
			{EventTypeUnbondingResourceNode, AttributeKeyUnbondingMatureTime},
			{EventTypeUnbondingResourceNode, AttributeKeyDepositToRemove},
		}
		processedEvents, initialEventCount := processEvents(eventDataTx.Result.Events, types.MSG_TYPE_UNBONDING_RESOURCE_NODE, requiredAttributes)

		key := getCacheKey(requiredAttributes, processedEvents, txHash)
		if _, ok := cache.Load(key); ok {
			utils.DebugLogf("Event unbonding_resource_node was already handled for tx [%v]. Ignoring...", txHash)
			return
		}
		cache.Store(key, true)

		req := &relay.UnbondingPPReq{}
		for _, event := range processedEvents {
			req.PPList = append(req.PPList, &protos.ReqUnbondingPP{
				P2PAddress:          event[EventAttribute{EventTypeUnbondingResourceNode, AttributeKeyResourceNode}],
				UnbondingMatureTime: event[EventAttribute{EventTypeUnbondingResourceNode, AttributeKeyUnbondingMatureTime}],
				TxHash:              txHash,
				DepositToRemove:     event[EventAttribute{EventTypeUnbondingResourceNode, AttributeKeyDepositToRemove}],
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
		txHash := getTxHash(result)
		eventDataTx, ok := result.Data.(comettypes.EventDataTx)
		if !ok {
			utils.ErrorLogf("result data is the wrong type in CompleteUnbondingResourceNodeMsgHandler: %T", result.Data)
			return
		}
		requiredAttributes := []EventAttribute{
			{EventTypeCompleteUnbondingResourceNode, AttributeKeyNetworkAddress},
		}
		processedEvents, initialEventCount := processEvents(eventDataTx.Result.Events, types.MSG_TYPE_COMPLETE_UNBONDING_RESOURCE_NODE, requiredAttributes)

		key := getCacheKey(requiredAttributes, processedEvents, txHash)
		if _, ok := cache.Load(key); ok {
			utils.DebugLogf("Event complete_unbonding_resource_node was already handled for tx [%v]. Ignoring...", txHash)
			return
		}
		cache.Store(key, true)

		req := &relay.DeactivatedPPReq{}
		for _, event := range processedEvents {
			req.PPList = append(req.PPList, &protos.ReqDeactivatedPP{
				P2PAddress: event[EventAttribute{EventTypeCompleteUnbondingResourceNode, AttributeKeyNetworkAddress}],
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

func UpdateMetaNodeDepositMsgHandler() func(event coretypes.ResultEvent) {
	return func(result coretypes.ResultEvent) {
		txHash := getTxHash(result)
		eventDataTx, ok := result.Data.(comettypes.EventDataTx)
		if !ok {
			utils.ErrorLogf("result data is the wrong type in UpdateMetaNodeDepositMsgHandler: %T", result.Data)
			return
		}
		requiredAttributes := []EventAttribute{
			{EventTypeUpdateMetaNodeDeposit, AttributeKeyNetworkAddress},
			{EventTypeUpdateMetaNodeDeposit, AttributeKeyOZoneLimitChanges},
			{EventTypeUpdateMetaNodeDeposit, AttributeKeyDepositDelta},
			{EventTypeUpdateMetaNodeDeposit, AttributeKeyCurrentDeposit},
			{EventTypeUpdateMetaNodeDeposit, AttributeKeyAvailableTokenBefore},
			{EventTypeUpdateMetaNodeDeposit, AttributeKeyAvailableTokenAfter},
		}
		processedEvents, initialEventCount := processEvents(eventDataTx.Result.Events, types.MSG_TYPE_UPDATE_META_NODE_DEPOSIT, requiredAttributes)

		key := getCacheKey(requiredAttributes, processedEvents, txHash)
		if _, ok := cache.Load(key); ok {
			utils.DebugLogf("Event update_meta_node_deposit was already handled for tx [%v]. Ignoring...", txHash)
			return
		}
		cache.Store(key, true)

		req := &relay.UpdatedDepositSPReq{}
		for _, event := range processedEvents {
			req.SPList = append(req.SPList, &protos.ReqUpdatedDepositSP{
				P2PAddress:           event[EventAttribute{EventTypeUpdateMetaNodeDeposit, AttributeKeyNetworkAddress}],
				OzoneLimitChanges:    event[EventAttribute{EventTypeUpdateMetaNodeDeposit, AttributeKeyOZoneLimitChanges}],
				DepositDelta:         event[EventAttribute{EventTypeUpdateMetaNodeDeposit, AttributeKeyDepositDelta}],
				CurrentDeposit:       event[EventAttribute{EventTypeUpdateMetaNodeDeposit, AttributeKeyCurrentDeposit}],
				AvailableTokenBefore: event[EventAttribute{EventTypeUpdateMetaNodeDeposit, AttributeKeyAvailableTokenBefore}],
				AvailableTokenAfter:  event[EventAttribute{EventTypeUpdateMetaNodeDeposit, AttributeKeyAvailableTokenAfter}],
				TxHash:               txHash,
			})
		}

		if len(req.SPList) != initialEventCount {
			utils.ErrorLogf("Updated SP deposit message handler couldn't process all events (success: %v  missing_attribute: %v  invalid_attribute: %v",
				len(req.SPList), initialEventCount-len(processedEvents), len(processedEvents)-len(req.SPList))
		}
		if len(req.SPList) == 0 {
			return
		}

		err := postToSP("/chain/updatedDeposit", req)
		if err != nil {
			utils.ErrorLog(err)
			return
		}
	}
}

func UnbondingMetaNodeMsgHandler() func(event coretypes.ResultEvent) {
	return func(result coretypes.ResultEvent) {
		txHash := getTxHash(result)
		eventDataTx, ok := result.Data.(comettypes.EventDataTx)
		if !ok {
			utils.ErrorLogf("result data is the wrong type in UnbondingMetaNodeMsgHandler: %T", result.Data)
			return
		}
		requiredAttributes := []EventAttribute{
			{EventTypeUnbondingMetaNode, AttributeKeyMetaNode},
			{EventTypeUnbondingMetaNode, AttributeKeyUnbondingMatureTime},
			{EventTypeUnbondingMetaNode, AttributeKeyDepositToRemove},
		}
		processedEvents, initialEventCount := processEvents(eventDataTx.Result.Events, types.MSG_TYPE_UNBONDING_META_NODE, requiredAttributes)

		key := getCacheKey(requiredAttributes, processedEvents, txHash)
		if _, ok := cache.Load(key); ok {
			utils.DebugLogf("Event unbonding_meta_node was already handled for tx [%v]. Ignoring...", txHash)
			return
		}
		cache.Store(key, true)

		req := &relay.UnbondingSPReq{}
		for _, event := range processedEvents {
			req.SPList = append(req.SPList, &protos.ReqUnbondingSP{
				P2PAddress:          event[EventAttribute{EventTypeUnbondingMetaNode, AttributeKeyMetaNode}],
				UnbondingMatureTime: event[EventAttribute{EventTypeUnbondingMetaNode, AttributeKeyUnbondingMatureTime}],
				TxHash:              txHash,
				DepositToRemove:     event[EventAttribute{EventTypeUnbondingMetaNode, AttributeKeyDepositToRemove}],
			})
		}

		if len(req.SPList) != initialEventCount {
			utils.ErrorLogf("unbonding SP message handler couldn't process all events (success: %v  missing_attribute: %v  invalid_attribute: %v",
				len(req.SPList), initialEventCount-len(processedEvents), len(processedEvents)-len(req.SPList))
		}
		if len(req.SPList) == 0 {
			return
		}

		err := postToSP("/chain/unbonding", req)
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
		txHash := getTxHash(result)
		eventDataTx, ok := result.Data.(comettypes.EventDataTx)
		if !ok {
			utils.ErrorLogf("result data is the wrong type in MetaNodeVoteMsgHandler: %T", result.Data)
			return
		}
		requiredAttributes := []EventAttribute{
			{EventTypeMetaNodeRegistrationVote, AttributeKeyCandidateNetworkAddress},
			{EventTypeMetaNodeRegistrationVote, AttributeKeyCandidateStatus},
			{EventTypeMetaNodeRegistrationVote, AttributeKeyOZoneLimitChanges},
		}
		processedEvents, initialEventCount := processEvents(eventDataTx.Result.Events, types.MSG_TYPE_META_NODE_REG_VOTE, requiredAttributes)

		key := getCacheKey(requiredAttributes, processedEvents, txHash)
		if _, ok := cache.Load(key); ok {
			utils.DebugLogf("Event meta_node_reg_vote was already handled for tx [%v]. Ignoring...", txHash)
			return
		}
		cache.Store(key, true)

		req := &relay.ActivatedSPReq{}
		for _, event := range processedEvents {
			candidateNetworkAddr := event[EventAttribute{EventTypeMetaNodeRegistrationVote, AttributeKeyCandidateNetworkAddress}]

			if event[EventAttribute{EventTypeMetaNodeRegistrationVote, AttributeKeyCandidateStatus}] != stakingv1beta1.BondStatus_BOND_STATUS_BONDED.String() {
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
		txHash := getTxHash(result)
		eventDataTx, ok := result.Data.(comettypes.EventDataTx)
		if !ok {
			utils.ErrorLogf("result data is the wrong type in PrepayMsgHandler: %T", result.Data)
			return
		}
		requiredAttributes := []EventAttribute{
			{EventTypePrepay, AttributeKeySender},
			{EventTypePrepay, AttributeKeyBeneficiary},
			{EventTypePrepay, AttributeKeyPurchasedNoz},
			{EventTypeMerkleDataUpdated, AttributeKeyRoot},
			{EventTypeMerkleDataUpdated, AttributeKeyCommitment},
		}
		processedEvents, initialEventCount := processEvents(eventDataTx.Result.Events, types.MSG_TYPE_PREPAY, requiredAttributes)
		processPrePayEvent(requiredAttributes, processedEvents, txHash, initialEventCount)
	}
}

func FileUploadMsgHandler() func(event coretypes.ResultEvent) {
	return func(result coretypes.ResultEvent) {
		txHash := getTxHash(result)
		eventDataTx, ok := result.Data.(comettypes.EventDataTx)
		if !ok {
			utils.ErrorLogf("result data is the wrong type in FileUploadMsgHandler: %T", result.Data)
			return
		}
		requiredAttributes := []EventAttribute{
			{EventTypeFileUpload, AttributeKeyReporter},
			{EventTypeFileUpload, AttributeKeyUploader},
			{EventTypeFileUpload, AttributeKeyFileHash},
		}
		processedEvents, initialEventCount := processEvents(eventDataTx.Result.Events, types.MSG_TYPE_FILE_UPLOAD, requiredAttributes)

		key := getCacheKey(requiredAttributes, processedEvents, txHash)
		if _, ok := cache.Load(key); ok {
			utils.DebugLogf("Event FileUpload was already handled for tx [%v]. Ignoring...", txHash)
			return
		}
		cache.Store(key, true)

		req := &relay.FileUploadedReq{}
		for _, event := range processedEvents {
			req.UploadList = append(req.UploadList, &protos.Uploaded{
				ReporterAddress: event[EventAttribute{EventTypeFileUpload, AttributeKeyReporter}],
				UploaderAddress: event[EventAttribute{EventTypeFileUpload, AttributeKeyUploader}],
				FileHash:        event[EventAttribute{EventTypeFileUpload, AttributeKeyFileHash}],
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
		txHash := getTxHash(result)
		eventDataTx, ok := result.Data.(comettypes.EventDataTx)
		if !ok {
			utils.ErrorLogf("result data is the wrong type in VolumeReportHandler: %T", result.Data)
			return
		}
		requiredAttributes := []EventAttribute{
			{EventTypeVolumeReport, AttributeKeyEpoch},
		}
		processedEvents, initialEventCount := processEvents(eventDataTx.Result.Events, types.MSG_TYPE_VOLUME_REPORT, requiredAttributes)

		key := getCacheKey(requiredAttributes, processedEvents, txHash)
		if _, ok := cache.Load(key); ok {
			utils.DebugLogf("Event volume_report was already handled for tx [%v]. Ignoring...", txHash)
			return
		}
		cache.Store(key, true)

		req := &relay.VolumeReportedReq{}
		for _, event := range processedEvents {
			req.Epochs = append(req.Epochs, event[EventAttribute{EventTypeVolumeReport, AttributeKeyEpoch}])
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
		txHash := getTxHash(result)
		eventDataTx, ok := result.Data.(comettypes.EventDataTx)
		if !ok {
			utils.ErrorLogf("result data is the wrong type in SlashingResourceNodeHandler: %T", result.Data)
			return
		}
		requiredAttributes := []EventAttribute{
			{EventTypeSlashing, AttributeKeyNetworkAddress},
			{EventTypeSlashing, AttributeKeyNodeSuspended},
			{EventTypeSlashing, AttributeKeyAmount},
		}
		processedEvents, initialEventCount := processEvents(eventDataTx.Result.Events, types.MSG_TYPE_SLASHING_RESOURCE_NODE, requiredAttributes)

		key := getCacheKey(requiredAttributes, processedEvents, txHash)
		if _, ok := cache.Load(key); ok {
			utils.DebugLogf("Event slashing was already handled for tx [%v]. Ignoring...", txHash)
			return
		}
		cache.Store(key, true)
		var slashedPPs []relay.SlashedPP
		for _, event := range processedEvents {
			suspended, err := strconv.ParseBool(event[EventAttribute{EventTypeSlashing, AttributeKeyNodeSuspended}])
			if err != nil {
				utils.DebugLog("Invalid suspended boolean in the slashing message from stratos-chain", err)
				continue
			}
			slashedAmt, ok := new(big.Int).SetString(event[EventAttribute{EventTypeSlashing, AttributeKeyAmount}], 10)
			if !ok {
				utils.DebugLog("Invalid slashed amount in big integer in the slashing message from stratos-chain")
				continue
			}
			slashedPP := relay.SlashedPP{
				P2PAddress: event[EventAttribute{EventTypeSlashing, AttributeKeyNetworkAddress}],
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

		req := relay.SlashedPPReq{
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

func UpdateEffectiveDepositHandler() func(event coretypes.ResultEvent) {
	return func(result coretypes.ResultEvent) {
		txHash := getTxHash(result)
		eventDataTx, ok := result.Data.(comettypes.EventDataTx)
		if !ok {
			utils.ErrorLogf("result data is the wrong type in UpdateEffectiveDepositHandler: %T", result.Data)
			return
		}
		requiredAttributes := []EventAttribute{
			{EventTypeUpdateEffectiveDeposit, AttributeKeyNetworkAddress},
			{EventTypeUpdateEffectiveDeposit, AttributeKeyIsUnsuspended},
			{EventTypeUpdateEffectiveDeposit, AttributeKeyEffectiveDepositAfter},
		}
		processedEvents, initialEventCount := processEvents(eventDataTx.Result.Events, types.MSG_TYPE_UPDATE_EFFECTIVE_DEPOSIT, requiredAttributes)

		key := getCacheKey(requiredAttributes, processedEvents, txHash)
		if _, ok := cache.Load(key); ok {
			utils.DebugLogf("Event update_effective_deposit was already handled for tx [%v]. Ignoring...", txHash)
			return
		}
		cache.Store(key, true)
		var updatedPPs []relay.UpdatedEffectiveDepositPP
		for _, event := range processedEvents {
			isUnsuspendedDuringUpdate, err := strconv.ParseBool(event[EventAttribute{EventTypeUpdateEffectiveDeposit, AttributeKeyIsUnsuspended}])
			if err != nil {
				utils.DebugLog("Invalid is_unsuspended boolean in the update_effective_deposit message from stratos-chain", err)
				continue
			}

			effectiveDepositAfter, ok := new(big.Int).SetString(event[EventAttribute{EventTypeUpdateEffectiveDeposit, AttributeKeyEffectiveDepositAfter}], 10)
			if !ok {
				utils.DebugLog("Invalid effective_deposit_after in big integer in the update_effective_deposit message from stratos-chain")
				continue
			}
			utils.DebugLogf("network_address: %v, isUnsuspendedDuringUpdate is %v, effectiveDepositAfter: %v",
				event[EventAttribute{EventTypeUpdateEffectiveDeposit, AttributeKeyNetworkAddress}], isUnsuspendedDuringUpdate, effectiveDepositAfter.String())

			if !isUnsuspendedDuringUpdate {
				// only msg for unsuspended node will be transferred to SP
				continue
			}

			updatedPP := relay.UpdatedEffectiveDepositPP{
				P2PAddress:                event[EventAttribute{EventTypeUpdateEffectiveDeposit, AttributeKeyNetworkAddress}],
				IsUnsuspendedDuringUpdate: isUnsuspendedDuringUpdate,
				EffectiveDepositAfter:     effectiveDepositAfter,
			}
			updatedPPs = append(updatedPPs, updatedPP)
		}

		if len(updatedPPs) > 0 {
			utils.DebugLogf("updatedEffectiveDeposit message handler is processing events to unsuspend pp "+
				"(ToBeUnsuspended Events: %v, Invalid Events: %v, Total : %v",
				len(updatedPPs), initialEventCount-len(processedEvents), initialEventCount)
		}
		if len(updatedPPs) == 0 {
			return
		}

		req := relay.UpdatedEffectiveDepositPPReq{
			PPList: updatedPPs,
			TxHash: txHash,
		}
		err := postToSP("/pp/updatedEffectiveDeposit", req)
		if err != nil {
			utils.ErrorLog(err)
			return
		}
	}
}

func processPrePayEvent(requiredAttributes []EventAttribute, processedEvents []map[EventAttribute]string, txHash string, initialEventCount int) {
	key := getCacheKey(requiredAttributes, processedEvents, txHash)
	if _, ok := cache.Load(key); ok {
		utils.DebugLogf("Event Prepay was already handled for tx [%v]. Ignoring...", txHash)
		return
	}
	cache.Store(key, true)

	req := &relay.PrepaidReq{}
	for _, event := range processedEvents {
		req.WalletList = append(req.WalletList, &protos.ReqPrepaid{
			WalletAddress:      event[EventAttribute{EventTypePrepay, AttributeKeySender}],
			PurchasedUoz:       event[EventAttribute{EventTypePrepay, AttributeKeyPurchasedNoz}],
			TxHash:             txHash,
			BeneficiaryAddress: event[EventAttribute{EventTypePrepay, AttributeKeyBeneficiary}],
			MerkleRoot:         event[EventAttribute{EventTypeMerkleDataUpdated, AttributeKeyRoot}],
			Commitment:         event[EventAttribute{EventTypeMerkleDataUpdated, AttributeKeyCommitment}],
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

func EvmTxHandler() func(event coretypes.ResultEvent) {
	return func(result coretypes.ResultEvent) {
		txHash := getTxHash(result)
		eventDataTx, ok := result.Data.(comettypes.EventDataTx)
		if !ok {
			utils.ErrorLogf("result data is the wrong type in EvmTxHandler: %T", result.Data)
			return
		}

		processedMsgs, evmTxMsgType := processEvmTxEvents(eventDataTx.Result.Events)
		if len(processedMsgs) == 0 {
			if evmTxMsgType == "" {
				utils.ErrorLogf("no known msg to process in EvmTxHandler for tx %v", txHash)
			} else {
				utils.ErrorLogf("missing attributes to process msg %v in EvmTxHandler for tx %v", evmTxMsgType, txHash)
			}
			return
		}

		switch evmTxMsgType {
		case types.MSG_TYPE_PREPAY:
			processPrePayEvent(EvmTxRequiredAttributes[types.MSG_TYPE_PREPAY], processedMsgs, txHash, 1)
		}
	}
}

// Evm txs contain only 1 msg each
func processEvmTxEvents(events []abcitypes.Event) ([]map[EventAttribute]string, string) {
	msgType := ""
EventLoop:
	for _, event := range events {
		// Try to identify the msg type
		if event.Type != EventTypeMessage {
			continue
		}
		for _, attribute := range event.Attributes {
			if attribute.Key != AttributeKeyAction {
				continue
			}
			msgType = attribute.Value
			break EventLoop
		}
	}
	if msgType == "" {
		return nil, ""
	}

	processedMsgs, _ := processEvents(events, msgType, EvmTxRequiredAttributes[msgType])
	return processedMsgs, msgType
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

func getTxHash(result coretypes.ResultEvent) string {
	txHash := ""
	if len(result.Events["tx.hash"]) > 0 {
		txHash = result.Events["tx.hash"][0]
	}
	return txHash
}

func processEvents(events []abcitypes.Event, msgType string, requiredAttributes []EventAttribute) ([]map[EventAttribute]string, int) {
	var processedMsgs []map[EventAttribute]string
	initialMsgCount := 0

	inMsg := false
	currentMsg := make(map[EventAttribute]string)
	msgFinished := func() {
		if !inMsg {
			return
		}
		missingAttributes := false
		for _, attribute := range requiredAttributes {
			if _, ok := currentMsg[attribute]; !ok {
				utils.ErrorLogf("Attribute %v.%v missing in msg of type %v", attribute.EventType, attribute.Attribute, msgType)
				missingAttributes = true
				break
			}
		}

		if !missingAttributes {
			processedMsgs = append(processedMsgs, currentMsg)
		}
		currentMsg = make(map[EventAttribute]string)
		inMsg = false
	}
EventLoop:
	for _, event := range events {
		// Each msg in the tx starts with an event of type "message" that has an attribute called "action" with a value that matches the desired msgType
		if event.Type == EventTypeMessage {
			for _, attribute := range event.Attributes {
				if attribute.Key != AttributeKeyAction {
					continue
				}

				msgFinished()
				if attribute.Value != msgType {
					continue EventLoop
				}
				// This is the correct msgType. Start saving all attributes
				inMsg = true
				initialMsgCount++
				break
			}
		}
		if inMsg {
			for _, attribute := range event.Attributes {
				currentMsg[EventAttribute{event.Type, attribute.Key}] = attribute.Value
			}
		}
	}
	msgFinished()
	return processedMsgs, initialMsgCount
}

func processHexPubkey(attribute string) (fwcryptotypes.PubKey, error) {
	p2pPubkeyRaw, err := hex.DecodeString(attribute)
	if err != nil {
		return nil, errors.Wrap(err, "Error when trying to decode P2P pubkey hex")
	}
	p2pPubkey := &ed25519.PubKey{Key: p2pPubkeyRaw}

	return p2pPubkey, nil
}

func getCacheKey(requiredAttributes []EventAttribute, processedEvents []map[EventAttribute]string, txHash string) string {
	rawKey := txHash
	for _, attribute := range requiredAttributes {
		rawKey += attribute.EventType + attribute.Attribute
		for _, event := range processedEvents {
			rawKey += event[attribute]
		}
	}
	hash := crypto.Keccak256([]byte(rawKey))
	return string(hash)
}
