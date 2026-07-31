package logout

import (
	"log/slog"
	"net/http"
	ssogrpc "rest/internal/clients/sso/grpc"
	resp "rest/internal/server/statuses"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
)

type Request struct {
	RefreshToken string `json:"refresh_token"`
}

type Response struct {
	Status string `json:"status"`
}

// New revokes the supplied refresh token.
//
// @Summary Log out
// @Tags auth
// @Accept json
// @Produce json
// @Param request body Request true "Refresh token"
// @Success 200 {object} Response
// @Failure 400 {object} statuses.Response "Invalid refresh token"
// @Failure 422 {object} statuses.Response "Malformed JSON"
// @Router /v1/transactions/auth/logout [post]
func New(log *slog.Logger, auth *ssogrpc.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.refresh.New"

		log = log.With(
			slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		var request Request

		if err := render.DecodeJSON(r.Body, &request); err != nil {
			log.Error("Fail decode JSON", slog.Attr{
				Key:   "error",
				Value: slog.StringValue(err.Error()),
			})

			render.Status(r, 422)
			render.JSON(w, r, resp.Error("failed to decode request"))
			return
		}

		err := auth.Logout(r.Context(), request.RefreshToken)
		if err != nil {
			log.Error("Fail to logout", slog.Attr{
				Key:   "error",
				Value: slog.StringValue(err.Error()),
			})

			render.Status(r, 400)
			render.JSON(w, r, resp.Error("failed to refresh request"))
			return
		}

		render.JSON(w, r, Response{Status: resp.StatusOK})
	}
}
