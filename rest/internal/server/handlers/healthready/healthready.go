package healthready

import (
	"context"
	"log/slog"
	"net/http"
	resp "rest/internal/server/statuses"
	"time"

	"github.com/go-chi/render"
)

type Checker interface {
	Check(ctx context.Context) error
}

type Response struct {
	Status string `json:"status"`
}

type Handler struct {
	log      *slog.Logger
	checkers []Checker
}

func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	for _, checker := range h.checkers {
		if err := checker.Check(ctx); err != nil {
			render.Status(r, http.StatusServiceUnavailable)
			render.JSON(w, r, resp.Error(err.Error()))

			return
		}
	}

	render.Status(r, 200)
	render.JSON(w, r, Response{Status: resp.StatusOK})
}

func New(log *slog.Logger, checkers ...Checker) http.HandlerFunc {
	const op = "handlers.healthready.New"

	log = log.With(
		slog.String("op", op),
	)

	h := &Handler{
		log:      log,
		checkers: checkers,
	}

	return h.Ready
}
