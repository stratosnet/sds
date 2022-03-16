package handlers

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	sdkTypes "github.com/cosmos/cosmos-sdk/types"
	setting "github.com/stratosnet/sds/cmd/relayd/config"
	"github.com/stratosnet/sds/msg/protos"
	"github.com/stratosnet/sds/relay"
	"github.com/stratosnet/sds/utils"
	"github.com/stratosnet/sds/utils/types"
	"github.com/tendermint/tendermint/crypto/ed25519"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
)

// TODO: update remaining handlers to be able to process multiple events
func CreateResourceNodeMsgHandler() func(event coretypes.ResultEvent) {
	return func(result coretypes.ResultEvent) {

		_, p2pAddressString, err := getP2pAddressFromEvent(result, "create_resource_node", "network_address")
		if err != nil {
			utils.ErrorLog(err.Error())
			return
		}

		nodePubkeyList := result.Events["create_resource_node.pub_key"]
		if len(nodePubkeyList) < 1 {
			utils.ErrorLog("No node pubkey was specified in the create_resource_node message from stratos-chain")
			return
		}
		p2pPubkeyRaw, err := hex.DecodeString(nodePubkeyList[0])
		if err != nil {
			utils.ErrorLog("Error when trying to decode P2P pubkey hex", err)
			return
		}
		p2pPubkey := ed25519.PubKeyEd25519{}
		err = relay.Cdc.UnmarshalBinaryBare(p2pPubkeyRaw, &p2pPubkey)
		if err != nil {
			utils.ErrorLog("Error when trying to read P2P pubkey ed25519 binary", err)
			return
		}

		ozoneLimitChangeStr := result.Events["create_resource_node.ozone_limit_changes"]

		txHashList := result.Events["tx.hash"]
		if len(txHashList) < 1 {
			utils.ErrorLog("No txHash was specified in the create_resource_node message from stratos-chain")
			return
		}

		activatedMsg := &protos.ReqActivatedPP{
			P2PAddress:        p2pAddressString,
			P2PPubkey:         hex.EncodeToString(p2pPubkey[:]),
			OzoneLimitChanges: ozoneLimitChangeStr[0],
			TxHash:            txHashList[0],
		}

		err = postToSP("/pp/activated", activatedMsg)
		if err != nil {
			utils.ErrorLog(err)
			return
		}
	}
}

func UpdateResourceNodeStakeMsgHandler() func(event coretypes.ResultEvent) {
	return func(result coretypes.ResultEvent) {

		_, p2pAddressString, err := getP2pAddressFromEvent(result, "update_resource_node_stake", "network_address")
		if err != nil {
			utils.ErrorLog(err.Error())
			return
		}

		ozoneLimitChangeStr := result.Events["update_resource_node_stake.ozone_limit_changes"]

		incrStakeBoolList := result.Events["update_resource_node_stake.incr_stake"]
		if len(incrStakeBoolList) < 1 {
			utils.ErrorLog("No incr stake status was specified in the update_resource_node_stake message from stratos-chain")
			return
		}

		txHashList := result.Events["tx.hash"]
		if len(txHashList) < 1 {
			utils.ErrorLog("No txHash was specified in the update_resource_node_stake message from stratos-chain")
			return
		}

		updatedStakeMsg := &protos.ReqUpdatedStakePP{
			P2PAddress:        p2pAddressString,
			OzoneLimitChanges: ozoneLimitChangeStr[0],
			IncrStake:         incrStakeBoolList[0],
			TxHash:            txHashList[0],
		}

		err = postToSP("/pp/updatedStake", updatedStakeMsg)
		if err != nil {
			utils.ErrorLog(err)
			return
		}
	}
}

func UnbondingResourceNodeMsgHandler() func(event coretypes.ResultEvent) {
	return func(result coretypes.ResultEvent) {
		_, p2pAddressString, err := getP2pAddressFromEvent(result, "unbonding_resource_node", "resource_node")
		if err != nil {
			utils.ErrorLog(err.Error())
			return
		}

		// get ozone limit change
		ozoneLimitChange := result.Events["unbonding_resource_node.ozone_limit_changes"]
		ozoneLimitChangeStr := ozoneLimitChange[0]
		// get mature time
		ubdMatureTime := result.Events["unbonding_resource_node.unbonding_mature_time"]
		ubdMatureTimeStr := ubdMatureTime[0]

		txHashList := result.Events["tx.hash"]
		if len(txHashList) < 1 {
			utils.ErrorLog("No txHash was specified in the unbonding_resource_node message from stratos-chain")
			return
		}

		ubdMsg := &protos.ReqUnbondingPP{
			P2PAddress:          p2pAddressString,
			OzoneLimitChanges:   ozoneLimitChangeStr,
			UnbondingMatureTime: ubdMatureTimeStr,
			TxHash:              txHashList[0],
		}

		err = postToSP("/pp/unbonding", ubdMsg)
		if err != nil {
			utils.ErrorLog(err)
			return
		}
	}
}

func RemoveResourceNodeMsgHandler() func(event coretypes.ResultEvent) {
	return func(result coretypes.ResultEvent) {
		_, p2pAddressString, err := getP2pAddressFromEvent(result, "remove_resource_node", "resource_node")
		if err != nil {
			utils.ErrorLog(err.Error())
			return
		}

		deactivatedMsg := &protos.ReqDeactivatedPP{
			P2PAddress: p2pAddressString,
		}

		txHashList := result.Events["tx.hash"]
		if len(txHashList) < 1 {
			utils.ErrorLog("No txHash was specified in the remove_resource_node message from stratos-chain")
			return
		}

		err = postToSP("/pp/deactivated", deactivatedMsg)
		if err != nil {
			utils.ErrorLog(err)
			return
		}
	}
}

func CompleteUnbondingResourceNodeMsgHandler() func(event coretypes.ResultEvent) {
	return RemoveResourceNodeMsgHandler()
}

func CreateIndexingNodeMsgHandler() func(event coretypes.ResultEvent) {
	return func(result coretypes.ResultEvent) {
		// TODO
		utils.Log(fmt.Sprintf("%+v", result))
	}
}

func UpdateIndexingNodeStakeMsgHandler() func(event coretypes.ResultEvent) {
	return func(result coretypes.ResultEvent) {
		_, p2pAddressString, err := getP2pAddressFromEvent(result, "update_indexing_node_stake", "network_address")
		if err != nil {
			utils.ErrorLog(err.Error())
			return
		}

		ozoneLimitChangeStr := result.Events["update_indexing_node_stake.ozone_limit_changes"]

		incrStakeBoolList := result.Events["update_indexing_node_stake.incr_stake"]
		if len(incrStakeBoolList) < 1 {
			utils.ErrorLog("No incr stake status was specified in the update_indexing_node_stake message from stratos-chain")
			return
		}

		txHashList := result.Events["tx.hash"]
		if len(txHashList) < 1 {
			utils.ErrorLog("No txHash was specified in the update_indexing_node_stake message from stratos-chain")
			return
		}

		updatedStakeMsg := &protos.ReqUpdatedStakeSP{
			P2PAddress:        p2pAddressString,
			OzoneLimitChanges: ozoneLimitChangeStr[0],
			IncrStake:         incrStakeBoolList[0],
			TxHash:            txHashList[0],
		}
		err = postToSP("/chain/updatedStake", updatedStakeMsg)
		if err != nil {
			utils.ErrorLog(err)
			return
		}
	}
}

func UnbondingIndexingNodeMsgHandler() func(event coretypes.ResultEvent) {
	return func(result coretypes.ResultEvent) {
		// TODO
		utils.Logf("%+v\n", result)
	}
}
func RemoveIndexingNodeMsgHandler() func(event coretypes.ResultEvent) {
	return func(result coretypes.ResultEvent) {
		// TODO
		utils.Logf("%+v", result)
	}
}
func CompleteUnbondingIndexingNodeMsgHandler() func(event coretypes.ResultEvent) {
	return RemoveIndexingNodeMsgHandler()
}

func IndexingNodeVoteMsgHandler() func(event coretypes.ResultEvent) {
	return func(result coretypes.ResultEvent) {
		_, p2pAddressString, err := getP2pAddressFromEvent(result, "indexing_node_reg_vote", "candidate_network_address")
		if err != nil {
			utils.ErrorLog(err.Error())
			return
		}

		candidateStatusList := result.Events["indexing_node_reg_vote.candidate_status"]
		if len(candidateStatusList) < 1 {
			utils.ErrorLog("No candidate status was specified in the indexing_node_reg_vote message from stratos-chain")
			return
		}
		if candidateStatusList[0] != sdkTypes.BondStatusBonded {
			utils.DebugLog("Indexing node vote handler: The candidate needs more votes before being considered active")
			return
		}

		txHashList := result.Events["tx.hash"]
		if len(txHashList) < 1 {
			utils.ErrorLog("No txHash was specified in the indexing_node_reg_vote message from stratos-chain")
			return
		}

		activatedMsg := &protos.ReqActivatedSP{
			P2PAddress: p2pAddressString,
			TxHash:     txHashList[0],
		}

		err = postToSP("/chain/activated", activatedMsg)
		if err != nil {
			utils.ErrorLog(err)
			return
		}
	}
}

func PrepayMsgHandler() func(event coretypes.ResultEvent) {
	return func(result coretypes.ResultEvent) {
		utils.Log(fmt.Sprintf("%+v", result))

		reporterList := result.Events["Prepay.sender"]
		if len(reporterList) < 1 {
			utils.ErrorLog("No wallet address was specified in the prepay message from stratos-chain")
			return
		}

		purchasedUozList := result.Events["Prepay.purchased"]
		if len(purchasedUozList) < 1 {
			utils.ErrorLog("No purchased ozone amount was specified in the prepay message from stratos-chain")
			return
		}

		txHashList := result.Events["tx.hash"]
		if len(txHashList) < 1 {
			utils.ErrorLog("No txHash was specified in the prepay message from stratos-chain")
			return
		}

		prepaidMsg := &protos.ReqPrepaid{
			WalletAddress: reporterList[0],
			PurchasedUoz:  purchasedUozList[0],
			TxHash:        txHashList[0],
		}

		err := postToSP("/pp/prepaid", prepaidMsg)
		if err != nil {
			utils.ErrorLog(err)
			return
		}
	}
}

func FileUploadMsgHandler() func(event coretypes.ResultEvent) {
	return func(result coretypes.ResultEvent) {
		reporterAddressList := result.Events["FileUpload.reporter"]
		if len(reporterAddressList) < 1 {
			utils.ErrorLog("No reporter address was specified in the FileUploadTx message from stratos-chain")
			return
		}

		uploaderAddressList := result.Events["FileUpload.uploader"]
		if len(uploaderAddressList) < 1 {
			utils.ErrorLog("No uploader address was specified in the FileUploadTx message from stratos-chain")
			return
		}

		fileHashList := result.Events["FileUpload.file_hash"]
		if len(fileHashList) < 1 {
			utils.ErrorLog("No file hash was specified in the FileUploadTx message from stratos-chain")
			return
		}

		txHashList := result.Events["tx.hash"]
		if len(txHashList) < 1 {
			utils.ErrorLog("No txHash was specified in the FileUploadTx message from stratos-chain")
			return
		}

		uploadedMsg := &protos.Uploaded{
			ReporterAddress: reporterAddressList[0],
			UploaderAddress: uploaderAddressList[0],
			FileHash:        fileHashList[0],
			TxHash:          txHashList[0],
		}

		err := postToSP("/pp/uploaded", uploadedMsg)
		if err != nil {
			utils.ErrorLog(err)
			return
		}
	}
}

type VolumeReportedReq struct {
	Epoch string `json:"epoch"`
}

func VolumeReportHandler() func(event coretypes.ResultEvent) {
	return func(result coretypes.ResultEvent) {
		epochList := result.Events["volume_report.epoch"]
		if len(epochList) < 1 {
			utils.ErrorLog("No epoch was specified in the volume_report message from stratos-chain")
			return
		}

		volumeReportedMsg := VolumeReportedReq{Epoch: epochList[0]}
		err := postToSP("/volume/reported", volumeReportedMsg)
		if err != nil {
			utils.ErrorLog(err)
			return
		}
	}
}

type SlashedPP struct {
	P2PAddress string `json:"p2p_address"`
	QueryFirst bool   `json:"query_first"`
	Suspended  bool   `json:"suspended"`
}

type SlashedPPReq struct {
	PPList []SlashedPP `json:"pp_list"`
}

func SlashingResourceNodeHandler() func(event coretypes.ResultEvent) {
	return func(result coretypes.ResultEvent) {
		requiredAttributes := []string{
			"slashing.network_address",
			"slashing.suspended",
		}
		processedEvents, initialEventCount := processEvents(result.Events, requiredAttributes)

		var slashedPPs []SlashedPP
		for _, event := range processedEvents {
			_, p2pAddressString, err := getP2pAddressFromAttribute(event["slashing.network_address"])
			if err != nil {
				utils.DebugLog("Invalid P2P address in the slashing message from stratos-chain", err)
				continue
			}

			suspended, err := strconv.ParseBool(event["slashing.suspended"])
			if err != nil {
				utils.DebugLog("Invalid suspended boolean in the slashing message from stratos-chain", err)
				continue
			}

			slashedPP := SlashedPP{
				P2PAddress: p2pAddressString,
				QueryFirst: false,
				Suspended:  suspended,
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

		slashedPPMsg := SlashedPPReq{
			PPList: slashedPPs,
		}
		err := postToSP("/pp/slashed", slashedPPMsg)
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
	json.NewDecoder(resp.Body).Decode(&res)

	utils.Log(endpoint+" endpoint response from SP node", resp.StatusCode, res["Msg"])
	return nil
}

func getP2pAddressFromEvent(result coretypes.ResultEvent, eventName, attribName string) (types.Address, string, error) {
	attribSlice := result.Events[eventName+"."+attribName]
	if len(attribSlice) < 1 {
		return types.Address{}, "", errors.New("no " + attribName + " was specified in " + eventName + " msg from st-chain")
	}
	return getP2pAddressFromAttribute(attribSlice[0])
}

func getP2pAddressFromAttribute(attribute string) (types.Address, string, error) {
	p2pAddress, err := types.P2pAddressFromBech(attribute)
	if err != nil {
		return types.Address{}, "", errors.New("error when trying to convert P2P address to bytes")
	}

	return p2pAddress, attribute, nil
}

func processEvents(eventsMap map[string][]string, attributesRequired []string) (processedEvents []map[string]string, totalEventCount int) {
	if len(attributesRequired) < 1 {
		return nil, 0
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
