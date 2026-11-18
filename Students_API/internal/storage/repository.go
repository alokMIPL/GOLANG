package notes

import (
	"context"
	"fmt"
	"strings"
	"time"

	"notes_api/internal/db"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	// CollectionName must match the collection used by db.NotesIndexes.
	CollectionName = "notes"

	queryTimeout = 5 * time.Second
)

// Repository stores notes in MongoDB.
type Repository struct {
	col *mongo.Collection
}

// NewRepository builds a repository on top of an open Store.
func NewRepository(store *db.Store) *Repository {
	return &Repository{col: store.Collection(CollectionName)}
}

// withTimeout gives ctx a deadline only if the caller did not set one.
func withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, queryTimeout)
}

// ownedBy builds the filter for one note that belongs to one user. Scoping
// every query by user_id keeps users from reading each other's notes.
func ownedBy(userID string, id primitive.ObjectID) bson.M {
	return bson.M{"_id": id, "user_id": userID}
}

// Create validates and inserts n. On success n.ID and the timestamps are set.
func (r *Repository) Create(ctx context.Context, n *Note) error {
	if err := n.Validate(); err != nil {
		return err
	}
	tags, _ := normalizeTags(n.Tags) // already validated above
	n.Tags = tags
	n.Title = strings.TrimSpace(n.Title)

	now := time.Now().UTC()
	n.ID = primitive.NilObjectID // let MongoDB generate it
	n.CreatedAt, n.UpdatedAt = now, now

	ctx, cancel := withTimeout(ctx)
	defer cancel()

	res, err := r.col.InsertOne(ctx, n)
	if err != nil {
		return fmt.Errorf("notes: insert: %w", err)
	}

	id, ok := res.InsertedID.(primitive.ObjectID)
	if !ok {
		return fmt.Errorf("notes: unexpected inserted id type %T", res.InsertedID)
	}
	n.ID = id
	return nil
}

// Get returns one note owned by userID.
func (r *Repository) Get(ctx context.Context, userID string, id primitive.ObjectID) (*Note, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	var n Note
	err := r.col.FindOne(ctx, ownedBy(userID, id)).Decode(&n)
	if err != nil {
		if db.IsNotFound(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("notes: get: %w", err)
	}
	return &n, nil
}

// List returns a page of the user's notes (pinned first, then newest first)
// and the total number of notes matching the filter.
func (r *Repository) List(ctx context.Context, userID string, p ListParams) ([]Note, int64, error) {
	p.Normalize()

	ctx, cancel := withTimeout(ctx)
	defer cancel()

	filter := bson.M{"user_id": userID}
	if p.Tag != "" {
		filter["tags"] = p.Tag
	}

	total, err := r.col.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("notes: count: %w", err)
	}

	findOpts := options.Find().
		SetSort(bson.D{{Key: "pinned", Value: -1}, {Key: "created_at", Value: -1}}).
		SetSkip(p.Offset).
		SetLimit(p.Limit)

	cur, err := r.col.Find(ctx, filter, findOpts)
	if err != nil {
		return nil, 0, fmt.Errorf("notes: find: %w", err)
	}
	defer cur.Close(ctx)

	notes := make([]Note, 0, p.Limit)
	if err := cur.All(ctx, &notes); err != nil {
		return nil, 0, fmt.Errorf("notes: decode: %w", err)
	}
	return notes, total, nil
}

// Update applies a partial update and returns the note as it is afterwards.
func (r *Repository) Update(ctx context.Context, userID string, id primitive.ObjectID, in UpdateInput) (*Note, error) {
	set := bson.M{}

	if in.Title != nil {
		if err := validateTitle(*in.Title); err != nil {
			return nil, err
		}
		set["title"] = strings.TrimSpace(*in.Title)
	}
	if in.Content != nil {
		if err := validateContent(*in.Content); err != nil {
			return nil, err
		}
		set["content"] = *in.Content
	}
	if in.Tags != nil {
		tags, err := normalizeTags(*in.Tags)
		if err != nil {
			return nil, err
		}
		set["tags"] = tags
	}
	if in.Pinned != nil {
		set["pinned"] = *in.Pinned
	}

	if len(set) == 0 {
		return nil, &ValidationError{"body", "no fields to update"}
	}
	set["updated_at"] = time.Now().UTC()

	ctx, cancel := withTimeout(ctx)
	defer cancel()

	var updated Note
	err := r.col.FindOneAndUpdate(
		ctx,
		ownedBy(userID, id),
		bson.M{"$set": set},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&updated)
	if err != nil {
		if db.IsNotFound(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("notes: update: %w", err)
	}
	return &updated, nil
}

// Delete removes one note owned by userID.
func (r *Repository) Delete(ctx context.Context, userID string, id primitive.ObjectID) error {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	res, err := r.col.DeleteOne(ctx, ownedBy(userID, id))
	if err != nil {
		return fmt.Errorf("notes: delete: %w", err)
	}
	if res.DeletedCount == 0 {
		return ErrNotFound
	}
	return nil
}

// Search runs a full-text query over title and content, best match first.
// It relies on the text index created by db.NotesIndexes.
func (r *Repository) Search(ctx context.Context, userID, query string, limit int64) ([]Note, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, &ValidationError{"q", "is required"}
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	ctx, cancel := withTimeout(ctx)
	defer cancel()

	filter := bson.M{
		"user_id": userID,
		"$text":   bson.M{"$search": query},
	}
	score := bson.M{"$meta": "textScore"}

	findOpts := options.Find().
		SetProjection(bson.M{"score": score}).
		SetSort(bson.M{"score": score}).
		SetLimit(limit)

	cur, err := r.col.Find(ctx, filter, findOpts)
	if err != nil {
		return nil, fmt.Errorf("notes: search: %w", err)
	}
	defer cur.Close(ctx)

	notes := make([]Note, 0, limit)
	if err := cur.All(ctx, &notes); err != nil {
		return nil, fmt.Errorf("notes: decode search: %w", err)
	}
	return notes, nil
}

// DeleteAllForUser removes every note of a user inside one transaction and
// returns how many were deleted. It shows how to use Store.WithTransaction;
// it needs a replica set.
func DeleteAllForUser(ctx context.Context, store *db.Store, userID string) (int64, error) {
	var deleted int64

	err := store.WithTransaction(ctx, func(txCtx context.Context) error {
		res, err := store.Collection(CollectionName).DeleteMany(txCtx, bson.M{"user_id": userID})
		if err != nil {
			return err
		}
		deleted = res.DeletedCount
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("notes: delete all for user: %w", err)
	}
	return deleted, nil
}
