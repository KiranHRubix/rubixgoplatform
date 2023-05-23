package core

import (
	"bytes"
	"fmt"
	"sync"

	ipfsnode "github.com/ipfs/go-ipfs-api"
	"github.com/rubixchain/rubixgoplatform/block"
	"github.com/rubixchain/rubixgoplatform/contract"
	"github.com/rubixchain/rubixgoplatform/core/wallet"
	"github.com/rubixchain/rubixgoplatform/did"
	"github.com/rubixchain/rubixgoplatform/token"
	"github.com/rubixchain/rubixgoplatform/util"
)

type TokenStateCheckResult struct {
	Token                 string
	Status                bool
	Error                 error
	Message               string
	tokenIDTokenStateData string
}

func (c *Core) validateTokenOwnership(cr *ConensusRequest, sc *contract.Contract) bool {
	ti := sc.GetTransTokenInfo()
	for i := range ti {
		ids, err := c.GetDHTddrs(ti[i].Token)
		if err != nil || len(ids) == 0 {
			continue
		}
	}
	address := cr.SenderPeerID + "." + sc.GetSenderDID()
	p, err := c.getPeer(address)
	if err != nil {
		c.log.Error("Failed to get peer", "err", err)
		return false
	}
	defer p.Close()
	for i := range ti {
		err := c.syncTokenChainFrom(p, ti[i].BlockID, ti[i].Token, ti[i].TokenType)
		if err != nil {
			c.log.Error("Failed to sync token chain block", "err", err)
			return false
		}
		// Check the token validation
		if !c.testNet {
			fb := c.w.GetFirstBlock(ti[i].Token, ti[i].TokenType)
			if fb == nil {
				c.log.Error("Failed to get first token chain block")
				return false
			}
			tl, tn, err := fb.GetTokenDetials(ti[i].Token)
			if err != nil {
				c.log.Error("Failed to get token detials", "err", err)
				return false
			}
			ct := token.GetTokenString(tl, tn)
			tb := bytes.NewBuffer([]byte(ct))
			tid, err := c.ipfs.Add(tb, ipfsnode.Pin(false), ipfsnode.OnlyHash(true))
			if err != nil {
				c.log.Error("Failed to validate, failed to get token hash", "err", err)
				return false
			}
			if tid != ti[i].Token {
				c.log.Error("Invalid token", "token", ti[i].Token, "exp_token", tid, "tl", tl, "tn", tn)
				return false
			}
		}
		b := c.w.GetLatestTokenBlock(ti[i].Token, ti[i].TokenType)
		if b == nil {
			c.log.Error("Invalid token chain block")
			return false
		}
		signers, err := b.GetSigner()
		if err != nil {
			c.log.Error("Failed to signers", "err", err)
			return false
		}
		for _, signer := range signers {
			var dc did.DIDCrypto
			switch b.GetTransType() {
			case block.TokenGeneratedType:
				dc, err = c.SetupForienDID(signer)
			default:
				dc, err = c.SetupForienDIDQuorum(signer)
			}
			err := b.VerifySignature(dc)
			if err != nil {
				c.log.Error("Failed to verify signature", "err", err)
				return false
			}
		}
	}
	// for i := range wt {
	// 	c.log.Debug("Requesting Token status")
	// 	ts := TokenPublish{
	// 		Token: wt[i],
	// 	}
	// 	c.ps.Publish(TokenStatusTopic, &ts)
	// 	c.log.Debug("Finding dht", "token", wt[i])
	// 	ids, err := c.GetDHTddrs(wt[i])
	// 	if err != nil || len(ids) == 0 {
	// 		c.log.Error("Failed to find token owner", "err", err)
	// 		crep.Message = "Failed to find token owner"
	// 		return c.l.RenderJSON(req, &crep, http.StatusOK)
	// 	}
	// 	if len(ids) > 1 {
	// 		// ::TODO:: to more check to findout right pwner
	// 		c.log.Error("Mutiple owner found for the token", "token", wt, "owners", ids)
	// 		crep.Message = "Mutiple owner found for the token"
	// 		return c.l.RenderJSON(req, &crep, http.StatusOK)
	// 	} else {
	// 		//:TODO:: get peer from the table
	// 		if cr.SenderPeerID != ids[0] {
	// 			c.log.Error("Token peer id mismatched", "expPeerdID", cr.SenderPeerID, "peerID", ids[0])
	// 			crep.Message = "Token peer id mismatched"
	// 			return c.l.RenderJSON(req, &crep, http.StatusOK)
	// 		}
	// 	}
	// }
	return true
}

func (c *Core) validateSignature(dc did.DIDCrypto, h string, s string) bool {
	if dc == nil {
		c.log.Error("Invalid DID setup")
		return false
	}
	sig := util.StrToHex(s)
	ok, err := dc.PvtVerify([]byte(h), sig)
	if err != nil {
		c.log.Error("Error in signature verification", "err", err)
		return false
	}
	if !ok {
		c.log.Error("Failed to verify signature")
		return false
	}
	return true
}

func (c *Core) checkTokenIsPledged(wt string) bool {
	tokenType := token.RBTTokenType
	if c.testNet {
		tokenType = token.TestTokenType
	}
	b := c.w.GetLatestTokenBlock(wt, tokenType)
	if b == nil {
		c.log.Error("Invalid token chain block")
		return true
	}
	return c.checkIsPledged(b, wt)
}

func (c *Core) checkTokenIsUnpledged(wt string) bool {
	tokenType := token.RBTTokenType
	if c.testNet {
		tokenType = token.TestTokenType
	}
	b := c.w.GetLatestTokenBlock(wt, tokenType)
	if b == nil {
		c.log.Error("Invalid token chain block")
		return true
	}
	return c.checkIsUnpledged(b, wt)
}

func (c *Core) getUnpledgeId(wt string) string {
	tokenType := token.RBTTokenType
	if c.testNet {
		tokenType = token.TestTokenType
	}
	b := c.w.GetLatestTokenBlock(wt, tokenType)
	if b == nil {
		c.log.Error("Invalid token chain block")
		return ""
	}
	return b.GetUnpledgeId()
}

/*
 * Function to check whether the TokenState is pinned or not
 * Input tokenId, index, resultArray, waitgroup
 */
//func (c *Core) checkTokenState(tokenId, did string, index int, resultArray []TokenStateCheckResult, wg *sync.WaitGroup, blockID string, tokenType int, address string) {
func (c *Core) checkTokenState(tokenId, did string, index int, resultArray []TokenStateCheckResult, wg *sync.WaitGroup) {
	defer wg.Done()
	var result TokenStateCheckResult
	result.Token = tokenId
	//get the latest blockId i.e. latest token state
	c.log.Debug("token ", tokenId)
	/* p, err := c.getPeer(address)
	if err != nil {
		c.log.Error("Failed to get peer", "err", err)
		result.Status = false
		result.Error = err
		result.Message = "Failed to get peer"
		resultArray[index] = result
	}
	defer p.Close()

	err1 := c.syncTokenChainFrom(p, blockID, tokenId, tokenType)
	if err1 != nil {
		c.log.Error("Failed to sync token chain block", "err", err)
		result.Status = false
		result.Error = err
		result.Message = "Failed to sync token chain block"
		resultArray[index] = result
	} */
	b := c.w.GetLatestTokenBlock(tokenId, token.RBTTokenType)
	if b == nil {
		c.log.Error("Invalid token chain block")
		result.Status = false
		result.Error = fmt.Errorf("Invalid token chain block")
		result.Message = "Invalid token chain block"
		resultArray[index] = result
	} else {
		blockId, err := b.GetBlockID(tokenId)
		if err != nil {
			c.log.Error("Error fetching block Id", err)
			result.Status = false
			result.Error = err
			result.Message = "Error fetching block Id"
			resultArray[index] = result
		}
		//concat tokenId and BlockID
		tokenIDTokenStateData := tokenId + blockId
		tokenIDTokenStateBuffer := bytes.NewBuffer([]byte(tokenIDTokenStateData))

		//add to ipfs get only the hash of the token+tokenstate
		tokenIDTokenStateHash, err := c.ipfs.Add(tokenIDTokenStateBuffer, ipfsnode.Pin(false), ipfsnode.OnlyHash(true))
		if err != nil {
			c.log.Error("Error adding data to ipfs", err)
			result.Status = false
			result.Error = err
			result.Message = "Error adding data to ipfs"
			resultArray[index] = result
		}

		//check dht to see if any pin exist
		list, err1 := c.GetDHTddrs(tokenIDTokenStateHash)
		//try to call ipfs cat to check if any one has pinned the state i.e \
		//content, err1 := c.w.Cat(tokenIDTokenStateHash, wallet.QuorumRole, did)
		if err1 != nil {
			c.log.Error("Error fetching content for the tokenstate ipfs hash :", tokenIDTokenStateHash, "Error", err)
			result.Status = false
			result.Error = err
			result.Message = "Error fetching content for the tokenstate ipfs hash : " + tokenIDTokenStateHash
			resultArray[index] = result
		}
		//if pin exist abort
		if len(list) != 0 {
			//if len(content) != 0 || content != "" {
			c.log.Debug("Token state is exhausted, Token is being Double spent")
			result.Status = true
			result.Error = nil
			result.Message = "Token state is exhausted, Token is being Double spent"
			resultArray[index] = result
		} else {
			c.log.Debug("Token state is not exhausted, Unique Txn")
			result.Status = false
			result.Error = nil
			result.Message = "Token state is not exhausted, Unique Txn"
			resultArray[index] = result
		}
	}
}

func (c *Core) pinTokenState(tokenStateCheckResult []TokenStateCheckResult, did string) {
	for i := range tokenStateCheckResult {
		tokenIDTokenStateBuffer := bytes.NewBuffer([]byte(tokenStateCheckResult[i].tokenIDTokenStateData))
		id, err := c.w.Add(tokenIDTokenStateBuffer, did, wallet.QuorumRole)
		if err != nil {
			c.log.Error("Error triggered while pinning token state", err)
		}
		c.log.Debug("token state pinned", id)
	}
}

func (c *Core) checkNodePartOfQuorum(peerId, did string, ql []string) bool {

	qPeerIds := make([]string, 0)
	qDids := make([]string, 0)

	for i := range ql {
		pId, dId, ok := util.ParseAddress(ql[i])
		if !ok {
			c.log.Error("Error parsing addressing")
		}
		qPeerIds = append(qPeerIds, pId)
		qDids = append(qDids, dId)
	}

	for i := range qPeerIds {
		if peerId == qPeerIds[i] {
			for j := range qDids {
				if did == qDids[j] {
					return true
				}
			}
		}
	}
	return false
}
