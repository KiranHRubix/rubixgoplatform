package client

import (
	"github.com/rubixchain/rubixgoplatform/core/model"
	"github.com/rubixchain/rubixgoplatform/setup"
)

func (c *Client) GetPendingTxn() (*model.PendingTxnIds, error) {
	var result model.PendingTxnIds
	err := c.sendJSONRequest("GET", setup.APIGetPendingTxn, nil, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) InitiateTxnFinality(txnId string) (*model.BasicResponse, error) {
	q := make(map[string]string)
	q["txnID"] = txnId
	var result model.BasicResponse
	err := c.sendJSONRequest("POST", setup.APIInitiateTxnFinality, q, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
