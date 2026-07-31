package consumer

import (
	"context"
	"encoding/json"
	"rest/internal/domain/models"
)

func ProcessEvent[T any](ctx context.Context, envelope models.EventEnvelope, save func(context.Context, string, T) (int, error)) (int, error) {
	var event T
	if err := json.Unmarshal(envelope.Data, &event); err != nil {
		return 0, err
	}

	return save(ctx, envelope.Title, event)
}
