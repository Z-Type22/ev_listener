package healthcheck

import (
	"log/slog"
	"net/http"
	resp "rest/internal/server/statuses"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
)

type Response struct {
	Status string `json:"status"`
}

func New(log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.healthcheck.New"

		log = log.With(
			slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		render.Status(r, 200)
		render.JSON(w, r, Response{Status: resp.StatusOK})
	}
}
