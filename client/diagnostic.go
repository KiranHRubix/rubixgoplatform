package client

import (
	"github.com/rubixchain/rubixgoplatform/core/model"
	"github.com/rubixchain/rubixgoplatform/server"
)

func (c *Client) DumpTokenChain(token string, blockID string) (*model.TCDumpReply, error) {
	dr := &model.TCDumpRequest{
		Token:   token,
		BlockID: blockID,
	}
	var drep model.TCDumpReply
	err := c.sendJSONRequest("POST", server.APIDumpTokenChainBlock, nil, dr, &drep)
	if err != nil {
		return nil, err
	}
	return &drep, nil
}

func (c *Client) RemoveTokenChain(token string) (*model.TCRemoveReply, error) {
	removeReq := &model.TCRemoveRequest{
		Token: token,
	}
	var removeReply model.TCRemoveReply
	err := c.sendJSONRequest("POST", server.APIRemoveTokenChain, nil, removeReq, &removeReply)
	if err != nil {
		return nil, err
	}
	return &removeReply, nil
}

func (c *Client) SyncTokenChain(token string, address string) (*model.BasicResponse, error) {
	syncReq := &model.TCSyncRequest{
		Token:   token,
		Address: address,
	}

	var syncReply model.BasicResponse
	err := c.sendJSONRequest("POST", server.APISyncTokenChainFromAddress, nil, syncReq, &syncReply)
	if err != nil {
		return nil, err
	}
	return &syncReply, nil
}
