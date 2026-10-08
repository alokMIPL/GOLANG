package db

import (
	"context"
	"fmt"
	"notes_api/internal/config"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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

	// Now apply all of our Configs()

	//clientOpts === It contains settings for creating a client, and the MongoURI
	clientOpts := options.Client().ApplyURI(cfg.MongoURI)

	// Here we creata cleint variable that stores the clientOpts inside this variable.
	// And in error we do the same approach.
	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, nil, fmt.Errorf("mongo connection failed.")
	}

	// A ping is a tiny "are you there?" message you send to a server to check that it's alive and responding.
	err = client.Ping(ctx, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("mongo ping failed")
	}
	// client.Database(...) creates the handle, and cfg.MongoDB only supplies the name for it.
	// And in database variable we store the created handler.
	database := client.Database(cfg.MongoDB)

	return client, database, nil

}

/*
Summary for this Connect() function

Connect() is a function that takes a config and promises to return a client, a database handle, and an error. First I create a ctx with a 10-second timeout, so Connect and Ping give up with an error if they take too long. Then I build clientOpts, which holds the settings parsed from cfg.MongoURI. I pass those to mongo.Connect to create the client, and check for an error. Next I ping the server to confirm it's reachable, and check that error too. Then I create database using client.Database(cfg.MongoDB), which gives me a handle to the database named in the config. Finally I return client, database, and nil (no error).
*/
