package stratoschain

import (
	"errors"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
)

type Client interface {
	SubscribeToStratosChain(query string, handler func(coretypes.ResultEvent)) error
}

func SubscribeToEvents(c Client) error {
	err := subscribeToCreateResourceNodeMsg(c)
	if err != nil {
		return errors.New("couldn't subscribe to create resource node msg: " + err.Error())
	}
	err = subscribeToRemoveResourceNodeMsg(c)
	if err != nil {
		return errors.New("couldn't subscribe to remove resource node msg: " + err.Error())
	}
	err = subscribeToCreateIndexingNodeMsg(c)
	if err != nil {
		return errors.New("couldn't subscribe to create indexing node msg: " + err.Error())
	}
	err = subscribeToRemoveIndexingNodeMsg(c)
	if err != nil {
		return errors.New("couldn't subscribe to remove indexing node msg: " + err.Error())
	}
	err = subscribeToSPRegistrationApprovedMsg(c)
	if err != nil {
		return errors.New("couldn't subscribe to SP registration approved msg: " + err.Error())
	}
	err = subscribeToPrepayMsg(c)
	if err != nil {
		return errors.New("couldn't subscribe to prepay msg: " + err.Error())
	}
	return nil
}

func subscribeToCreateResourceNodeMsg(c Client) error {
	err := c.SubscribeToStratosChain("tm.event='Tx' AND event.type='create_resource_node'", func(result coretypes.ResultEvent) {
		// TODO
	})
	return err
}

func subscribeToRemoveResourceNodeMsg(c Client) error {
	err := c.SubscribeToStratosChain("tm.event='Tx' AND event.type='remove_resource_node'", func(result coretypes.ResultEvent) {
		// TODO
	})
	return err
}

func subscribeToCreateIndexingNodeMsg(c Client) error {
	err := c.SubscribeToStratosChain("tm.event='Tx' AND event.type='create_indexing_node'", func(result coretypes.ResultEvent) {
		// TODO
	})
	return err
}

func subscribeToRemoveIndexingNodeMsg(c Client) error {
	err := c.SubscribeToStratosChain("tm.event='Tx' AND event.type='remove_indexing_node'", func(result coretypes.ResultEvent) {
		// TODO
	})
	return err
}

func subscribeToSPRegistrationApprovedMsg(c Client) error {
	// TODO: name will probably change when this is implemented in stratos-chain
	err := c.SubscribeToStratosChain("tm.event='Tx' AND event.type='sp_registration_approved'", func(result coretypes.ResultEvent) {
		// TODO
	})
	return err
}

func subscribeToPrepayMsg(c Client) error {
	// TODO: name will probably change when this is implemented in stratos-chain
	err := c.SubscribeToStratosChain("tm.event='Tx' AND event.type='prepay'", func(result coretypes.ResultEvent) {
		// TODO
	})
	return err
}
