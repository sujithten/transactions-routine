package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/ts_project/transactions_routine/internal/repository"
	"github.com/ts_project/transactions_routine/internal/service"
)

type AccountHandler struct {
	svc *service.AccountService
}

func NewAccountHandler(svc *service.AccountService) *AccountHandler {
	return &AccountHandler{svc: svc}
}

type createAccountRequest struct {
	DocumentNumber string `json:"document_number" example:"12345678900"`
}

// CreateAccount godoc
// @Summary      Create an account
// @Description  Creates a new cardholder account with a unique document number
// @Tags         accounts
// @Accept       json
// @Produce      json
// @Param        body  body      createAccountRequest  true  "Account payload"
// @Success      201   {object}  domain.Account
// @Failure      400   {object}  ErrorResponse
// @Failure      409   {object}  ErrorResponse  "Document number already registered"
// @Router       /accounts [post]
func (h *AccountHandler) CreateAccount(w http.ResponseWriter, r *http.Request) {
	var body createAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.DocumentNumber == "" {
		writeError(w, http.StatusBadRequest, "document_number is required")
		return
	}

	account, err := h.svc.CreateAccount(r.Context(), body.DocumentNumber)
	if err != nil {
		if errors.Is(err, repository.ErrDuplicateDocument) {
			writeError(w, http.StatusConflict, "document number already registered")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusCreated, account)
}

// GetAccount godoc
// @Summary      Get an account
// @Description  Retrieves a cardholder account by its UUID
// @Tags         accounts
// @Produce      json
// @Param        accountId  path      string  true  "Account UUID"
// @Success      200        {object}  domain.Account
// @Failure      404        {object}  ErrorResponse
// @Router       /accounts/{accountId} [get]
func (h *AccountHandler) GetAccount(w http.ResponseWriter, r *http.Request) {
	accountID := r.PathValue("accountId")
	if err := uuid.Validate(accountID); err != nil {
		writeError(w, http.StatusBadRequest, "invalid account id")
		return
	}

	account, err := h.svc.GetAccount(r.Context(), accountID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "account not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, account)
}
