package server

import (
	"net/http"

	"github.com/rubixchain/rubixgoplatform/core/model"
	"github.com/rubixchain/rubixgoplatform/wrapper/ensweb"
)

// ShowAccount godoc
// @Summary      get finality pending txn
// @Description  For a mentioned DID, check the finality pending txns
// @Tags         Account
// @Accept       json
// @Produce      json
// @Param        did      	   query      string  true  "User DID"
// @Success 200 {object} model.PendingTxnIds
// @Router /api/get-pending-txn [get]
func (s *Server) APIGetPendingTxn(req *ensweb.Request) (res *ensweb.Result) {
	did := s.GetQuerry(req, "did")
	if !s.validateDIDAccess(req, did) {
		return s.BasicResponse(req, false, "DID does not have an access", nil)
	}
	response, err := s.c.GetFinalityPendingTxns(did)
	if err != nil {
		s.log.Error("Error", err)
		result := model.PendingTxnIds{
			BasicResponse: model.BasicResponse{
				Status:  false,
				Message: "Error Triggered" + err.Error(),
			},
			TxnIds: make([]string, 0),
		}
		return s.RenderJSON(req, &result, http.StatusOK)
	}

	result := model.PendingTxnIds{
		BasicResponse: model.BasicResponse{
			Status:  true,
			Message: "Finality Pending Txns Retrieved",
		},
		TxnIds: make([]string, 0),
	}

	for i := range response {
		result.TxnIds = append(result.TxnIds, response[i])
	}
	return s.RenderJSON(req, &result, http.StatusOK)
}

// @Summary Txn Finality by Transcation ID
// @Description Initiates the process to achieve finality of pending txn.
// @ID initiate-txn-finality
// @Tags         Account
// @Accept json
// @Produce json
// @Param txnID query string true "The ID of the transaction pending finality"
// @Success 200 {object} model.BasicResponse
// @Router /api/initiate-txn-finality [post]
func (s *Server) APIInitiateTxnFinality(req *ensweb.Request) (res *ensweb.Result) {
	txnID := s.GetQuerry(req, "txnID")
	response := s.c.InitiateRBTTxnFinality(txnID)
	if !response.Status {
		s.log.Error("Error Occured", response.Message)
		return s.RenderJSON(req, &response, http.StatusOK)
	}
	return s.RenderJSON(req, &response, http.StatusOK)
}
