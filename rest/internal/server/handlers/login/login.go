package login

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
	Email    string `json:"email"`
	Password string `json:"password"`
}

type Response struct {
	Status       string `json:"status"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// New authenticates a user and issues access and refresh tokens.
//
// @Summary Log in
// @Description Authenticates a user and returns a pair of JWT tokens.
// @Tags auth
// @Accept json
// @Produce json
// @Param request body Request true "Credentials"
// @Success 200 {object} Response
// @Failure 400 {object} statuses.Response "Invalid credentials"
// @Failure 422 {object} statuses.Response "Malformed JSON"
// @Failure 500 {object} statuses.Response "Internal server error"
// @Router /v1/transactions/auth/login [post]
func New(log *slog.Logger, auth *ssogrpc.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.login.New"

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

		tokenResponse, err := auth.Login(r.Context(), request.Email, request.Password)
		if err != nil {
			log.Error("Fail to login", slog.String("error", err.Error()))

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

		render.JSON(
			w, r, Response{
				Status:       resp.StatusOK,
				AccessToken:  tokenResponse.AccessToken,
				RefreshToken: tokenResponse.RefreshToken,
			},
		)
	}
}
