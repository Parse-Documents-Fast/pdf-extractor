package consumer

import (
	"context"
	"fmt"
	"strings"

	redisclient "github.com/PARSE-DOCUMENT-FAST/pdf-extractor/internal/redis"
)

func EnsureGroup(ctx context.Context, rdb *redisclient.Client, stream, group string) error {
	err := rdb.XGroupCreateMkStream(ctx, stream, group, "$").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		return fmt.Errorf("consumer: crear consumer group: %w", err)
	}
	return nil
}
