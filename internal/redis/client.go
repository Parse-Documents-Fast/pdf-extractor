package redis

import (
	"context"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

type Client struct {
	*goredis.Client
}

func OptionsFromURI(uri string, timeout time.Duration) (*goredis.Options, error) {
	opts, err := goredis.ParseURL(uri)
	if err != nil {
		return nil, fmt.Errorf("redis: parse uri: %w", err)
	}
	opts.DialTimeout = timeout
	opts.ReadTimeout = timeout
	opts.WriteTimeout = timeout
	return opts, nil
}

func New(ctx context.Context, uri string, timeout time.Duration) (*Client, error) {
	opts, err := OptionsFromURI(uri, timeout)
	if err != nil {
		return nil, err
	}

	client := goredis.NewClient(opts)

	pingCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if err := client.Ping(pingCtx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("redis: connect: %w", err)
	}

	return &Client{Client: client}, nil
}
