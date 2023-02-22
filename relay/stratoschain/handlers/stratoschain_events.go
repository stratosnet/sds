package handlers

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"net/http"
	"strconv"
	"time"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	stakingTypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/pkg/errors"
	"github.com/stratosnet/sds/cmd/relayd/setting"
	"github.com/stratosnet/sds/msg/protos"
	relayTypes "github.com/stratosnet/sds/relay/types"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/crypto"
	"github.com/stratosnet/sds/utils/crypto/ed25519"
	//"github.com/tendermint/tendermint/crypto/ed25519"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
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
	//Handlers["complete_unbonding_meta_node"] = CompleteUnbondingMetaNodeMsgHandler()
	Handlers[MSG_TYPE_META_NODE_REG_VOTE] = MetaNodeVoteMsgHandler()
	Handlers[MSG_TYPE_PREPAY] = PrepayMsgHandler()
	Handlers[MSG_TYPE_FILE_UPLOAD] = FileUploadMsgHandler()
	Handlers[MSG_TYPE_VOLUME_REPORT] = VolumeReportHandler()
	Handlers[MSG_TYPE_SLASHING_RESOURCE_NODE] = SlashingResourceNodeHandler()
	Handlers[MSG_TYPE_UPDATE_EFFECTIVE_STAKE] = UpdateEffectiveStakeHandler()

	cache = utils.NewAutoCleanMap(time.Minute)
}

func CreateResourceNodeMsgHandler() func(event coretypes.ResultEvent) {
	return func(result coretypes.ResultEvent) {
		requiredAttributes := []string{
			"create_resource_node.network_address",
			"create_resource_node.pub_key",
			"create_resource_node.ozone_limit_changes",
			"create_resource_node.initial_stake",
		}
		processedEvents, txHash, initialEventCount := processEvents(result.Events, requiredAttributes)
		key := getCacheKey(requiredAttributes, result)
		if _, ok := cache.Load(key); ok {
			utils.DebugLogf("Event create_resource_node was already handled for tx [%v]. Ignoring...", txHash)
			return
		}
		cache.Store(key, true)

		req := &relayTypes.ActivatedPPReq{}
		for _, event := range processedEvents {
			p2pPubkey, err := processHexPubkey(event["create_resource_node.pub_key"])
			if err != nil {
				utils.ErrorLog(err)
				continue
			}

			req.PPList = append(req.PPList, &protos.ReqActivatedPP{
				P2PAddress:        event["create_resource_node.network_address"],
				P2PPubkey:         hex.EncodeToString(p2pPubkey.Bytes()),
				OzoneLimitChanges: event["create_resource_node.ozone_limit_changes"],
				TxHash:            txHash,
				InitialStake:      event["create_resource_node.initial_stake"],
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
		requiredAttributes := []string{
			"update_resource_node_stake.network_address",
			"update_resource_node_stake.ozone_limit_changes",
			"update_resource_node_stake.incr_stake",
			"update_resource_node_stake.stake_delta",
			"update_resource_node_stake.current_stake",
		}
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
				P2PAddress:        event["update_resource_node_stake.network_address"],
				OzoneLimitChanges: event["update_resource_node_stake.ozone_limit_changes"],
				IncrStake:         event["update_resource_node_stake.incr_stake"],
				TxHash:            txHash,
				StakeDelta:        event["update_resource_node_stake.stake_delta"],
				CurrentStake:      event["update_resource_node_stake.current_stake"],
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
		requiredAttributes := []string{
			"unbonding_resource_node.resource_node",
			"unbonding_resource_node.ozone_limit_changes",
			"unbonding_resource_node.unbonding_mature_time",
			"unbonding_resource_node.stake_to_remove",
		}
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
				P2PAddress:          event["unbonding_resource_node.resource_node"],
				OzoneLimitChanges:   event["unbonding_resource_node.ozone_limit_changes"],
				UnbondingMatureTime: event["unbonding_resource_node.unbonding_mature_time"],
				TxHash:              txHash,
				StakeToRemove:       event["unbonding_resource_node.stake_to_remove"],
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
		requiredAttributes := []string{
			"complete_unbonding_resource_node.network_address",
		}
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
				P2PAddress: event["complete_unbonding_resource_node.network_address"],
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
		requiredAttributes := []string{
			"update_meta_node_stake.network_address",
			"update_meta_node_stake.ozone_limit_changes",
			"update_meta_node_stake.incr_stake",
		}
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
				P2PAddress:        event["update_meta_node_stake.network_address"],
				OzoneLimitChanges: event["update_meta_node_stake.ozone_limit_changes"],
				IncrStake:         event["update_meta_node_stake.incr_stake"],
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
func CompleteUnbondingMetaNodeMsgHandler() func(event coretypes.ResultEvent) {
	return func(result coretypes.ResultEvent) {
		// TODO
		utils.Logf("%+v", result)
	}
}

func MetaNodeVoteMsgHandler() func(event coretypes.ResultEvent) {
	return func(result coretypes.ResultEvent) {
		requiredAttributes := []string{
			"meta_node_reg_vote.candidate_network_address",
			"meta_node_reg_vote.candidate_status",
		}
		processedEvents, txHash, initialEventCount := processEvents(result.Events, requiredAttributes)
		key := getCacheKey(requiredAttributes, result)
		if _, ok := cache.Load(key); ok {
			utils.DebugLogf("Event meta_node_reg_vote was already handled for tx [%v]. Ignoring...", txHash)
			return
		}
		cache.Store(key, true)

		req := &relayTypes.ActivatedSPReq{}
		for _, event := range processedEvents {
			if event["meta_node_reg_vote.candidate_status"] != stakingTypes.BondStatusBonded {
				utils.ErrorLogf("Indexing node vote handler: The candidate [%v] needs more votes before being considered active", event["meta_node_reg_vote.candidate_network_address"])
				continue
			}

			req.SPList = append(req.SPList, &protos.ReqActivatedSP{
				P2PAddress: event["meta_node_reg_vote.candidate_network_address"],
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

		requiredAttributes := []string{
			"Prepay.sender",
			"Prepay.purchased",
		}
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
				WalletAddress: event["Prepay.sender"],
				PurchasedUoz:  event["Prepay.purchased"],
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
		requiredAttributes := []string{
			"FileUpload.reporter",
			"FileUpload.uploader",
			"FileUpload.file_hash",
		}
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
				ReporterAddress: event["FileUpload.reporter"],
				UploaderAddress: event["FileUpload.uploader"],
				FileHash:        event["FileUpload.file_hash"],
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
		requiredAttributes := []string{
			"volume_report.epoch",
		}
		processedEvents, txHash, initialEventCount := processEvents(result.Events, requiredAttributes)
		key := getCacheKey(requiredAttributes, result)
		if _, ok := cache.Load(key); ok {
			utils.DebugLogf("Event volume_report was already handled for tx [%v]. Ignoring...", txHash)
			return
		}
		cache.Store(key, true)

		req := &relayTypes.VolumeReportedReq{}
		for _, event := range processedEvents {
			req.Epochs = append(req.Epochs, event["volume_report.epoch"])
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
		requiredAttributes := []string{
			"slashing.p2p_address",
			"slashing.suspend",
			"slashing.amount",
			"slashing.is_effective_stake_changed",
			"slashing.effective_stake",
		}
		processedEvents, txHash, initialEventCount := processEvents(result.Events, requiredAttributes)
		key := getCacheKey(requiredAttributes, result)
		if _, ok := cache.Load(key); ok {
			utils.DebugLogf("Event slashing was already handled for tx [%v]. Ignoring...", txHash)
			return
		}
		cache.Store(key, true)
		var slashedPPs []relayTypes.SlashedPP
		for _, event := range processedEvents {
			suspended, err := strconv.ParseBool(event["slashing.suspend"])
			if err != nil {
				utils.DebugLog("Invalid suspended boolean in the slashing message from stratos-chain", err)
				continue
			}
			slashedAmt, ok := new(big.Int).SetString(event["slashing.amount"], 10)
			if !ok {
				utils.DebugLog("Invalid slashed amount in big integer in the slashing message from stratos-chain")
				continue
			}
			isEffectiveStakeChanged, err := strconv.ParseBool(event["slashing.is_effective_stake_changed"])
			if err != nil {
				utils.DebugLog("Invalid flag of is_effective_stake_changed in the slashing message from stratos-chain", err)
				continue
			}
			effectiveStake, ok := new(big.Int).SetString(event["slashing.effective_stake"], 10)
			if !ok {
				utils.DebugLog("Invalid effective stake in big integer in the slashing message from stratos-chain")
				continue
			}
			utils.DebugLogf("slashed amount is %v, is_effective_stake_changed: %v, "+
				"current effective stake: %v", slashedAmt.String(),
				isEffectiveStakeChanged, effectiveStake.String())
			slashedPP := relayTypes.SlashedPP{
				P2PAddress:              event["slashing.p2p_address"],
				QueryFirst:              false,
				Suspended:               suspended,
				SlashedAmt:              slashedAmt,
				IsEffectiveStakeChanged: isEffectiveStakeChanged,
				EffectiveStake:          effectiveStake,
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
		requiredAttributes := []string{
			"update_effective_stake.network_address",
			"update_effective_stake.is_unsuspended",
			"update_effective_stake.effective_stake_after",
		}
		processedEvents, txHash, initialEventCount := processEvents(result.Events, requiredAttributes)
		key := getCacheKey(requiredAttributes, result)
		if _, ok := cache.Load(key); ok {
			utils.DebugLogf("Event update_effective_stake was already handled for tx [%v]. Ignoring...", txHash)
			return
		}
		cache.Store(key, true)
		var updatedPPs []relayTypes.UpdatedEffectiveStakePP
		for _, event := range processedEvents {
			isUnsuspendedDuringUpdate, err := strconv.ParseBool(event["update_effective_stake.is_unsuspended"])
			if err != nil {
				utils.DebugLog("Invalid is_unsuspended boolean in the update_effective_stake message from stratos-chain", err)
				continue
			}

			effectiveStakeAfter, ok := new(big.Int).SetString(event["update_effective_stake.effective_stake_after"], 10)
			if !ok {
				utils.DebugLog("Invalid effective_stake_after in big integer in the update_effective_stake message from stratos-chain")
				continue
			}
			utils.DebugLogf("network_address: %v, isUnsuspendedDuringUpdate is %v, effectiveStakeAfter: %v",
				event["update_effective_stake.network_address"], isUnsuspendedDuringUpdate, effectiveStakeAfter.String())

			if !isUnsuspendedDuringUpdate {
				// only msg for unsuspended node will be transferred to SP
				continue
			}

			updatedPP := relayTypes.UpdatedEffectiveStakePP{
				P2PAddress:                event["update_effective_stake.network_address"],
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
