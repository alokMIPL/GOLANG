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

// Connect opens a MongoDB connection, verifies the server answers a ping, and
// returns the client together with a handle to the configured database.
//
// The caller owns the client and must pass it to Disconnect when finished.
func Connect(cfg config.Config) (*mongo.Client, *mongo.Database, error) {
	if err := validate(cfg); err != nil {
		return nil, nil, err
	}

	// ctx is the countdown: if connecting or pinging takes longer than
	// connectTimeout, the operation stops with a deadline-exceeded error.
	// cancel stops the countdown early and frees the timer.
	ctx, cancel := context.WithTimeout(context.Background(), connectTimeout)
	defer cancel()

	// clientOpts holds the driver settings built from the URI in the config.
	clientOpts := options.Client().
		ApplyURI(cfg.MongoURI).
		SetServerSelectionTimeout(serverSelectionTimeout).
		SetConnectTimeout(connectTimeout)

	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %w", ErrConnect, err)
	}

	// A ping is a tiny "are you there?" message. Using the primary read
	// preference confirms we can reach a node that accepts writes.
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		// Don't leak the client when the ping fails.
		_ = Disconnect(client)
		return nil, nil, fmt.Errorf("%w: %w", ErrPing, err)
	}

	// client.Database only builds a handle; cfg.MongoDB supplies its name.
	database := client.Database(cfg.MongoDB)

	return client, database, nil
}

// Disconnect closes the client's connections, waiting at most
// disconnectTimeout. It is safe to call with a nil client.
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
