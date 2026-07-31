package gettran

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"rest/internal/domain/models"
	custom_middleware "rest/internal/middleware"
	"rest/internal/server/responses"
	resp "rest/internal/server/statuses"
	"rest/internal/storage"
	"strconv"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
)

type TransactionGetter interface {
	GetTransaction(ctx context.Context, id int, wallet string) (models.Transaction, error)
}

type Response struct {
	Status      string                        `json:"status"`
	Transaction responses.TransactionResponse `json:"transaction"`
}

// New returns a transaction by ID if it belongs to the authenticated wallet.
//
// @Summary Get transaction
// @Tags transactions
// @Produce json
// @Security AccessTokenCookie
// @Param id path int true "Transaction ID" minimum(1)
// @Success 200 {object} Response
// @Failure 400 {object} statuses.Response "Invalid transaction ID"
// @Failure 401 {object} statuses.Response "Unauthorized"
// @Failure 404 {object} statuses.Response "Transaction not found"
// @Failure 500 {object} statuses.Response "Internal server error"
// @Router /v1/transactions/{id} [get]
func New(log *slog.Logger, tranGetter TransactionGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.gettran.New"

		log = log.With(
			slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		idStr := chi.URLParam(r, "id")
		id, err := strconv.Atoi(idStr)
		if err != nil || id <= 0 {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.Error("invalid id"))
			return
		}

		wallet, ok := r.Context().Value(custom_middleware.WalletKey).(string)
		if !ok {
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Error("unauthorized"))

			return
		}

		transaction, err := tranGetter.GetTransaction(r.Context(), id, wallet)
		if errors.Is(err, storage.ErrTransactionNotFound) {
			log.Error("failed to get transaction",
				slog.Attr{
					Key:   "error",
					Value: slog.StringValue(err.Error()),
				},
			)

			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, resp.Error("transaction not found"))
			return
		}

		if err != nil {
			log.Error("failed to get transaction",
				slog.Attr{
					Key:   "error",
					Value: slog.StringValue(err.Error()),
				},
			)

			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.Error("internal error"))
			return
		}

		render.JSON(w, r, Response{
			Status: resp.StatusOK,
			Transaction: responses.TransactionResponse{
				ID:             transaction.ID,
				TxHash:         transaction.TxHash,
				UserWallet:     transaction.UserWallet,
				MerchantWallet: transaction.MerchantWallet,
				Price:          transaction.Price,
				Type:           transaction.Type,
				Payload:        transaction.Payload,
				CreatedAt:      transaction.CreatedAt,
			},
		})
	}
}
