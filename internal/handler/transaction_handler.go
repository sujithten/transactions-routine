package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/ts_project/transactions_routine/internal/repository"
	"github.com/ts_project/transactions_routine/internal/service"
)

type TransactionHandler struct {
	svc *service.TransactionService
}

func NewTransactionHandler(svc *service.TransactionService) *TransactionHandler {
	return &TransactionHandler{svc: svc}
}

type createTransactionRequest struct {
	AccountID       string  `json:"account_id"        example:"550e8400-e29b-41d4-a716-446655440000"`
	OperationTypeID int     `json:"operation_type_id" example:"2"`
	Amount          float64 `json:"amount"            example:"12000.00"`
	Installments    int     `json:"installments"      example:"12"`
}

// CreateTransaction godoc
// @Summary      Create a transaction
// @Description  Creates a new transaction. For operation_type_id=2 (Purchase with Installments) the installments field is required and must be > 0. Purchase and withdrawal amounts are stored as negative; credit vouchers as positive.
// @Tags         transactions
// @Accept       json
// @Produce      json
// @Param        body  body      createTransactionRequest  true  "Transaction payload"
// @Success      201   {object}  domain.Transaction
// @Failure      400   {object}  ErrorResponse
// @Failure      422   {object}  ErrorResponse  "Account or operation type not found"
// @Router       /transactions [post]
func (h *TransactionHandler) CreateTransaction(w http.ResponseWriter, r *http.Request) {
	var body createTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.AccountID == "" || body.OperationTypeID == 0 {
		writeError(w, http.StatusBadRequest, "account_id and operation_type_id are required")
		return
	}

	tx, err := h.svc.CreateTransaction(r.Context(), body.AccountID, body.OperationTypeID, body.Amount, body.Installments)
	if err != nil {
		if errors.Is(err, service.ErrInvalidInput) {
			writeError(w, http.StatusBadRequest, "installments must be greater than 0 for purchase with installments")
			return
		}
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusUnprocessableEntity, "account or operation type not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusCreated, tx)
}
