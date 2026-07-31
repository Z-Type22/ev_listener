package gettrans

import (
	"context"
	"log/slog"
	"net/http"
	"rest/internal/domain/models"
	custom_middleware "rest/internal/middleware"
	"rest/internal/server/responses"
	resp "rest/internal/server/statuses"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
)

type ListTransactionGetter interface {
	GetListTransactions(ctx context.Context, wallet string) ([]models.Transaction, error)
}

type Response struct {
	Status       string                          `json:"status"`
	Transactions []responses.TransactionResponse `json:"transactions"`
}

// New returns all transactions belonging to the authenticated wallet.
//
// @Summary List transactions
// @Tags transactions
// @Produce json
// @Security AccessTokenCookie
// @Success 200 {object} Response
// @Failure 401 {object} statuses.Response "Unauthorized"
// @Failure 500 {object} statuses.Response "Internal server error"
// @Router /v1/transactions/ [get]
func New(log *slog.Logger, tranGetter ListTransactionGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.gettrans.New"

		log = log.With(
			slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		wallet, ok := r.Context().Value(custom_middleware.WalletKey).(string)
		if !ok {
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Error("unauthorized"))

			return
		}

		transactions, err := tranGetter.GetListTransactions(r.Context(), wallet)
		if err != nil {
			log.Error("failed to get list of transactions",
				slog.Attr{
					Key:   "error",
					Value: slog.StringValue(err.Error()),
				},
			)

			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.Error("internal error"))
			return
		}

		transactionsResponse := make([]responses.TransactionResponse, 0, len(transactions))

		for _, transaction := range transactions {
			transactionsResponse = append(transactionsResponse, responses.TransactionResponse{
				ID:             transaction.ID,
				TxHash:         transaction.TxHash,
				UserWallet:     transaction.UserWallet,
				MerchantWallet: transaction.MerchantWallet,
				Price:          transaction.Price,
				Type:           transaction.Type,
				Payload:        transaction.Payload,
				CreatedAt:      transaction.CreatedAt,
			})
		}

		render.JSON(w, r, Response{
			Status:       resp.StatusOK,
			Transactions: transactionsResponse,
		})
	}
}
