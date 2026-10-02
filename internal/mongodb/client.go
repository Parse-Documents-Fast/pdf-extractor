package mongodb

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

type Client struct {
	*mongo.Client
}

func OptionsFromURI(uri string, timeout time.Duration) (*options.ClientOptions, error) {
	opts := options.Client().ApplyURI(uri).
		SetConnectTimeout(timeout).
		SetServerSelectionTimeout(timeout)

	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("mongodb: parse uri: %w", err)
	}

	return opts, nil
}

func New(ctx context.Context, uri string, timeout time.Duration) (*Client, error) {
	opts, err := OptionsFromURI(uri, timeout)
	if err != nil {
		return nil, err
	}

	client, err := mongo.Connect(opts)
	if err != nil {
		return nil, fmt.Errorf("mongodb: connect: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if err := client.Ping(pingCtx, readpref.Primary()); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, fmt.Errorf("mongodb: ping: %w", err)
	}

	return &Client{Client: client}, nil
}

func (c *Client) Database(name string) *mongo.Database {
	return c.Client.Database(name)
}

func (c *Client) Close(ctx context.Context) error {
	return c.Client.Disconnect(ctx)
}
