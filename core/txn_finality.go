package core

import (
	"fmt"
	"time"

	"github.com/rubixchain/rubixgoplatform/contract"
	"github.com/rubixchain/rubixgoplatform/core/model"
	"github.com/rubixchain/rubixgoplatform/core/wallet"
	"github.com/rubixchain/rubixgoplatform/token"
)

func (c *Core) InitiateRBTTxnFinality(txnId string) *model.BasicResponse {
	st := time.Now()
	resp := &model.BasicResponse{
		Status: false,
	}
	//read the transactionstatusstorage DB and retrive the DATA related to txnID given
	txnDetails, err := c.GetTxnDetails(txnId)
	if err != nil {
		c.log.Error("Failed to get txndetails txnID", txnId, "err", err)
		resp.Message = "Failed to get txndetails txnID " + txnId
		return resp
	}
	// Get the receiver & do sanity check
	p, err := c.getPeer(txnDetails.ReceiverPeerId + "." + txnDetails.ReceiverDID)
	if err != nil {
		resp.Message = "Failed to get receiver peer, " + err.Error()
		return resp
	}
	defer p.Close()

	//call the finality method

	finalityTxnDetails, err := c.initiateFinlaity(txnDetails, txnId)
	if err != nil {
		c.log.Error("Consensus failed", "err", err)
		resp.Message = "Consensus failed" + err.Error()
		return resp
	}
	dif := time.Now().Sub(st)
	//check for error

	//send data to explorer
	c.w.AddTransactionHistory(finalityTxnDetails)
	etrans := &ExplorerTrans{
		TID:         txnId,
		SenderDID:   txnDetails.SenderDID,
		ReceiverDID: txnDetails.ReceiverDID,
		Amount:      float64(len(txnDetails.Tokens)),
		TrasnType:   2,
		TokenIDs:    txnDetails.Tokens,
		QuorumList:  txnDetails.QuorumList,
		TokenTime:   float64(dif.Milliseconds()),
	}
	c.ec.ExplorerTransaction(etrans)
	c.log.Info("Transaction finality achieved successfully")
	resp.Status = true
	msg := fmt.Sprintf("Transfer finality achieved successfully, TxnID :" + txnId)
	resp.Message = msg
	return resp

}

func (c *Core) initiateFinlaity(finlaityPendingDetails wallet.TxnDetails, txnId string) (*wallet.TransactionDetails, error) {
	//connect to the receiver
	rp, err := c.getPeer(finlaityPendingDetails.ReceiverPeerId + "." + finlaityPendingDetails.ReceiverDID)
	if err != nil {
		c.log.Error("Receiver not connected", "err", err)
		return nil, err
	}

	//check tokentype
	tokenType := token.RBTTokenType
	if c.testNet {
		tokenType = token.TestTokenType
	}
	//get the block from the token chain
	newBlockAfterConsensus := c.w.GetLatestTokenBlock(finlaityPendingDetails.Tokens[0], tokenType)
	if newBlockAfterConsensus == nil {
		c.log.Error("Could not fetch the Block created post successful Consesnsus")
		return nil, fmt.Errorf("Could not fetch the Block created post successful Consesnsus")
	}

	contractBlock := newBlockAfterConsensus.GetSmartContract()
	if contractBlock == nil {
		c.log.Error("Could not fetch the contract details block")
		return nil, fmt.Errorf("Could not fetch the contract details block")
	}
	ctrct := contract.InitContract(contractBlock, nil)
	if ctrct == nil {
		c.log.Error("Could not Intit the contract details")
		return nil, fmt.Errorf("Could not Init the contract details")
	}

	tokenInfo := ctrct.GetTransTokenInfo()

	//create and send token details to the receiver
	defer rp.Close()
	sendTokenFinality := SendTokenRequest{
		Address:         finlaityPendingDetails.SenderPeerId + "." + finlaityPendingDetails.SenderDID,
		TokenInfo:       tokenInfo,
		TokenChainBlock: newBlockAfterConsensus.GetBlock(),
		Finality:        true,
	}

	var br model.BasicResponse
	err = rp.SendJSONRequest("POST", APISendReceiverToken, nil, &sendTokenFinality, &br, true)
	if err != nil {
		c.log.Error("Unable to send tokens to receiver", "err", err)
		return nil, err
	}
	if !br.Status {
		c.log.Error("Unable to send tokens to receiver", "msg", br.Message)
		return nil, fmt.Errorf("unable to send tokens to receiver, " + br.Message)
	}

	c.log.Debug("updating token status")
	//update token status to transferred in DB
	err = c.w.TokensFinalityStatus(ctrct.GetSenderDID(), tokenInfo, newBlockAfterConsensus, rp.IsLocal(), wallet.TokenIsTransferred)
	if err != nil {
		c.log.Error("Failed to transfer tokens", "err", err)
		return nil, err
	}
	//remove the pins
	for _, t := range tokenInfo {
		c.w.UnPin(t.Token, wallet.PrevSenderRole, ctrct.GetSenderDID())
	}
	//call ipfs repo gc after unpinnning
	c.ipfsRepoGc()
	nbid, err := newBlockAfterConsensus.GetBlockID(finlaityPendingDetails.Tokens[0])
	if err != nil {
		c.log.Error("Failed to get block id", "err", err)
		return nil, err
	}

	td := wallet.TransactionDetails{
		TransactionID:   txnId,
		TransactionType: newBlockAfterConsensus.GetTransType(),
		BlockID:         nbid,
		Mode:            wallet.SendMode,
		SenderDID:       finlaityPendingDetails.SenderDID,
		ReceiverDID:     finlaityPendingDetails.ReceiverDID,
		Comment:         newBlockAfterConsensus.GetComment(),
		DateTime:        time.Now(),
		Status:          true,
	}
	//add db txn status - Finality Achieved
	c.updateTxnStatus(tokenInfo, wallet.FinalityAchieved, txnId)
	return &td, nil
}
