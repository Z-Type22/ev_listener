package middleware

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"

	"rest/internal/lib/jwt"
	resp "rest/internal/server/statuses"
)

var WalletKey string = "wallet"

func Auth(log *slog.Logger, publicKeyPath string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			const op = "middleware.Auth"

			log := log.With(
				slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())),
			)

			cookie, err := r.Cookie("access_token")
			if err != nil {
				log.Error("failed to get access token", slog.String("error", err.Error()))

				render.Status(r, http.StatusUnauthorized)
				render.JSON(w, r, resp.Error("unauthorized"))

				return
			}

			payload, err := jwt.DecodeJWT(cookie.Value, publicKeyPath)
			if err != nil {
				if errors.Is(err, jwt.ErrRefreshTokenExpired) {
					log.Info("access token expired")

					render.Status(r, http.StatusUnauthorized)
					render.JSON(w, r, resp.Error("access token expired"))

					return
				}

				log.Error("failed to decode jwt", slog.String("error", err.Error()))

				render.Status(r, http.StatusUnauthorized)
				render.JSON(w, r, resp.Error("invalid token"))

				return
			}

			wallet, ok := payload["wallet"].(string)
			if !ok {
				log.Error("wallet is empty")

				render.Status(r, http.StatusUnauthorized)
				render.JSON(w, r, resp.Error("unauthorized"))

				return
			}

			ctx := context.WithValue(r.Context(), WalletKey, wallet)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
