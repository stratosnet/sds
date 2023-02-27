package tx

import (
	"fmt"

	coretypes "github.com/tendermint/tendermint/rpc/core/types"

	sdktx "github.com/cosmos/cosmos-sdk/types/tx"
)

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
