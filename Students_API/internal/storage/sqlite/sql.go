package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"notes_api/internal/config"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

const (
	// connectTimeout bounds the whole startup (connect + ping), so the app
	// can't hang forever if MongoDB is unreachable.
	connectTimeout = 10 * time.Second

	// disconnectTimeout bounds shutdown.
	disconnectTimeout = 5 * time.Second

	// serverSelectionTimeout is how long the driver searches for a usable
	// server before giving up.
	serverSelectionTimeout = 5 * time.Second
)

// Errors returned by this package. Use errors.Is to check for them.
var (
	ErrEmptyURI      = errors.New("mongo: URI is empty")
	ErrEmptyDatabase = errors.New("mongo: database name is empty")
	ErrConnect       = errors.New("mongo: connection failed")
	ErrPing          = errors.New("mongo: ping failed")
)

// validate checks that the config has what Connect needs.
func validate(cfg config.Config) error {
	if cfg.MongoURI == "" {
		return ErrEmptyURI
	}
	if cfg.MongoDB == "" {
		return ErrEmptyDatabase
	}
	return nil
}

func Connect(cfg config.Config) (*mongo.Client, *mongo.Database, error) {
	if err := validate(cfg); err != nil {
		return nil, nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), connectTimeout)
	defer cancel()

	clientOpts := options.Client().
		ApplyURI(cfg.MongoURI).
		SetServerSelectionTimeout(serverSelectionTimeout).
		SetConnectTimeout(connectTimeout)

	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %w", ErrConnect, err)
	}

	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		_ = Disconnect(client)
		return nil, nil, fmt.Errorf("%w: %w", ErrPing, err)
	}

	database := client.Database(cfg.MongoDB)

	return client, database, nil
}

func Disconnect(client *mongo.Client) error {
	if client == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), disconnectTimeout)
	defer cancel()

	if err := client.Disconnect(ctx); err != nil {
		return fmt.Errorf("mongo: disconnect: %w", err)
	}
	return nil
}
