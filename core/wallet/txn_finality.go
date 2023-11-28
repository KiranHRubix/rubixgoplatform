package wallet

import (
	"fmt"
	"os"

	"github.com/rubixchain/rubixgoplatform/block"
	"github.com/rubixchain/rubixgoplatform/contract"
	"github.com/rubixchain/rubixgoplatform/util"
)

func (w *Wallet) TokensFinalityStatus(did string, ti []contract.TokenInfo, b *block.Block, local bool, status int) error {
	w.l.Lock()
	defer w.l.Unlock()
	if !local {
		if status == TokenTxnFinalityPending {
			err := w.CreateTokenBlock(b)
			if err != nil {
				w.log.Error("Error at creating block due to finality pending", "err", err)
				return err
			}
		}
		for i := range ti {
			var t Token
			err := w.s.Read(TokenStorage, &t, "did=? AND token_id=?", did, ti[i].Token)
			if err != nil {
				w.log.Error("Error at reading token info", "err", err)
				return err
			}
			t.TokenValue = 1
			t.TokenStatus = status
			err = w.s.Update(TokenStorage, &t, "did=? AND token_id=?", did, ti[i].Token)
			if err != nil {
				w.log.Error("Error at updating token info", "err", err)
				return err
			}
		}
	}
	return nil
}

func (w Wallet) TokensReceivedForFinality(did string, ti []contract.TokenInfo, b *block.Block) error {
	w.l.Lock()
	defer w.l.Unlock()

	for i := range ti {
		var t Token
		err := w.s.Read(TokenStorage, &t, "token_id=?", ti[i].Token)
		if err != nil || t.TokenID == "" {
			dir := util.GetRandString()
			err := util.CreateDir(dir)
			if err != nil {
				w.log.Error("Faled to create directory", "err", err)
				return err
			}
			defer os.RemoveAll(dir)
			err = w.Get(ti[i].Token, did, OwnerRole, dir)
			if err != nil {
				w.log.Error("Faled to get token", "err", err)
				return err
			}
			t = Token{
				TokenID:    ti[i].Token,
				TokenValue: 1,
				DID:        did,
			}
			err = w.s.Write(TokenStorage, &t)
			if err != nil {
				return err
			}
		}

		t.DID = did
		t.TokenStatus = TokenIsFree
		err = w.s.Update(TokenStorage, &t, "token_id=?", ti[i].Token)
		if err != nil {
			return err
		}
		//Pinnig the whole tokens and pat tokens
		ok, err := w.Pin(ti[i].Token, OwnerRole, did)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("failed to pin token")
		}
	}
	return nil
}
