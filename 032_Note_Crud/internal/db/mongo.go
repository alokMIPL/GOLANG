package db

import (
	"notes_api/internal/config"

	"go.mongodb.org/mongo-driver/mongo"
)

func Connect(cfg config.Config) (*mongo.Client, *mongo.Database, error)
