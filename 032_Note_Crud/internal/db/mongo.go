package db

import (
	"context"
	"notes_api/internal/config"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
)

/*
cfg config.Config means the function takes one parameter named cfg, whose type is Config from your config package. It's usually a struct holding settings like the MongoDB URI and the database name. So cfg is the settings box, and config.Config is its type.

The return values are the three things you listed:

*mongo.Client is the connection to the MongoDB server
*mongo.Database is the specific database you'll work in
error is nil if everything worked, or the problem if it didn't

The * means these are pointers, so the function hands back a reference to the client and database objects instead of copies. You use them the same way either way.

When you call it, you receive all three:

*/

func Connect(cfg config.Config) (*mongo.Client, *mongo.Database, error) {

	// Prevent Our app from freezing in startup

	/*
		ctx holds the 10-second countdown (the deadline). You pass it to operations like mongo.Connect(ctx, ...) and client.Ping(ctx, nil). If 10 seconds pass first, they stop and return a context deadline exceeded error.
	*/

	/*
		cancel is not a value and does not record whether the connection failed. It is a function. Calling it stops the countdown early and releases the timer's resources. That's what defer cancel() does when your function exits.
	*/
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
}
