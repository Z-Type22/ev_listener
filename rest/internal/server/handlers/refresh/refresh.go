package refresh

import (
	"log/slog"
	"net/http"
	ssogrpc "rest/internal/clients/sso/grpc"
	resp "rest/internal/server/statuses"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Request struct {
	RefreshToken string `json:"refresh_token"`
}

type Response struct {
	Status      string `json:"status"`
	AccessToken string `json:"access_token"`
}

// New issues a new access token for a valid refresh token.
//
// @Summary Refresh access token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body Request true "Refresh token"
// @Success 200 {object} Response
// @Failure 400 {object} statuses.Response "Invalid refresh token"
// @Failure 422 {object} statuses.Response "Malformed JSON"
// @Failure 500 {object} statuses.Response "Internal server error"
// @Router /v1/transactions/auth/refresh [post]
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

		accessToken, err := auth.Refresh(r.Context(), request.RefreshToken)
		if err != nil {
			log.Error("Fail to refresh", slog.String("error", err.Error()))

			st, ok := status.FromError(err)
			if ok {
				switch st.Code() {
				case codes.InvalidArgument:
					render.Status(r, 400)
					render.JSON(w, r, resp.Error(st.Message()))

					return
				default:
					render.Status(r, 500)
					render.JSON(w, r, resp.Error("internal server error"))

					return
				}
			}

			render.Status(r, 500)
			render.JSON(w, r, resp.Error("internal server error"))

			return
		}

		render.JSON(w, r, Response{Status: resp.StatusOK, AccessToken: accessToken})
	}
}
