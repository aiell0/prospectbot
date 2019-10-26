package prospectbot

import (
	"context"
	"fmt"
)

type MyEvent struct {
	Name string `json: "name"`
}

func HandleError(ctx context.Context, name MyEvent) (string, error) {
	return fmt.Sprintf("Hello %s!", name.Name), nil
}
