package notes

import (
	"context"
	"errors"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ---- in-memory fake repository ------------------------------------------

type fakeRepo struct {
	data map[primitive.ObjectID]Note
}

func newFakeRepo() *fakeRepo { return &fakeRepo{data: map[primitive.ObjectID]Note{}} }

func (f *fakeRepo) Create(_ context.Context, n *Note) error {
	n.ID = primitive.NewObjectID()
	f.data[n.ID] = *n
	return nil
}

func (f *fakeRepo) Get(_ context.Context, userID string, id primitive.ObjectID) (*Note, error) {
	n, ok := f.data[id]
	if !ok || n.UserID != userID {
		return nil, ErrNotFound
	}
	return &n, nil
}

func (f *fakeRepo) List(_ context.Context, userID string, p ListParams) ([]Note, int64, error) {
	var all []Note
	for _, n := range f.data {
		if n.UserID != userID {
			continue
		}
		if p.Tag != "" && !contains(n.Tags, p.Tag) {
			continue
		}
		all = append(all, n)
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].Pinned != all[j].Pinned {
			return all[i].Pinned
		}
		return all[i].UpdatedAt.After(all[j].UpdatedAt)
	})

	total := int64(len(all))
	start := min(p.Offset, total)
	end := min(start+p.Limit, total)
	return all[start:end], total, nil
}

func (f *fakeRepo) Update(_ context.Context, userID string, id primitive.ObjectID, p Patch) (*Note, error) {
	n, ok := f.data[id]
	if !ok || n.UserID != userID {
		return nil, ErrNotFound
	}
	if p.Title != nil {
		n.Title = *p.Title
	}
	if p.Content != nil {
		n.Content = *p.Content
	}
	if p.Tags != nil {
		n.Tags = *p.Tags
	}
	if p.Pinned != nil {
		n.Pinned = *p.Pinned
	}
	n.UpdatedAt = p.UpdatedAt
	f.data[id] = n
	return &n, nil
}

func (f *fakeRepo) Delete(_ context.Context, userID string, id primitive.ObjectID) error {
	n, ok := f.data[id]
	if !ok || n.UserID != userID {
		return ErrNotFound
	}
	delete(f.data, id)
	return nil
}

func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}

// ---- helpers --------------------------------------------------------------

// newTestService returns a service whose clock advances one minute per call.
func newTestService() *Service {
	t := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	return NewService(newFakeRepo(), WithClock(func() time.Time {
		t = t.Add(time.Minute)
		return t
	}))
}

func ptr[T any](v T) *T { return &v }

// ---- tests ----------------------------------------------------------------

func TestNormalizeTags(t *testing.T) {
	tests := []struct {
		name    string
		in      []string
		want    []string
		wantMsg bool
	}{
		{"nil becomes empty slice", nil, []string{}, false},
		{"trims, lowers, dedupes", []string{" Go ", "go", "GO", "Mongo"}, []string{"go", "mongo"}, false},
		{"drops blanks", []string{"", "  ", "a"}, []string{"a"}, false},
		{"too many", make([]string, maxTags+1), nil, true},
		{"tag too long", []string{strings.Repeat("x", maxTagLen+1)}, nil, true},
		{"multibyte counted as runes", []string{strings.Repeat("é", maxTagLen)}, []string{strings.Repeat("é", maxTagLen)}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, msg := normalizeTags(tc.in)
			if (msg != "") != tc.wantMsg {
				t.Fatalf("message = %q, wantMsg = %v", msg, tc.wantMsg)
			}
			if !tc.wantMsg && !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestPrepareCollectsAllErrors(t *testing.T) {
	n := &Note{
		Title:   "   ",
		Content: strings.Repeat("a", maxContentLen+1),
		Tags:    make([]string, maxTags+1),
	}
	err := n.Prepare()

	var verrs ValidationErrors
	if !errors.As(err, &verrs) {
		t.Fatalf("expected ValidationErrors, got %T", err)
	}
	fields := map[string]bool{}
	for _, e := range verrs {
		fields[e.Field] = true
	}
	for _, f := range []string{"user_id", "title", "content", "tags"} {
		if !fields[f] {
			t.Errorf("missing validation error for %q (got %v)", f, verrs)
		}
	}

	var single *ValidationError
	if !errors.As(err, &single) {
		t.Error("errors.As should also find an individual *ValidationError")
	}
}

func TestValidateDoesNotMutate(t *testing.T) {
	n := &Note{UserID: "u1", Title: "  Hi  ", Tags: []string{" A "}}
	if err := n.Validate(); err != nil {
		t.Fatal(err)
	}
	if n.Title != "  Hi  " || n.Tags[0] != " A " {
		t.Fatalf("Validate mutated the note: %+v", n)
	}
}

func TestCreateAndGet(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	created, err := svc.Create(ctx, "u1", CreateInput{
		Title: "  Groceries ",
		Tags:  []string{"Home", "home", "TODO"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.Title != "Groceries" {
		t.Errorf("title not trimmed: %q", created.Title)
	}
	if !reflect.DeepEqual(created.Tags, []string{"home", "todo"}) {
		t.Errorf("tags not normalized: %v", created.Tags)
	}
	if !created.CreatedAt.Equal(created.UpdatedAt) {
		t.Error("CreatedAt and UpdatedAt should match on creation")
	}

	got, err := svc.Get(ctx, "u1", created.ID.Hex())
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != created.ID {
		t.Errorf("got id %s, want %s", got.ID.Hex(), created.ID.Hex())
	}
}

func TestOwnershipIsEnforced(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	n, err := svc.Create(ctx, "alice", CreateInput{Title: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	id := n.ID.Hex()

	if _, err := svc.Get(ctx, "bob", id); !errors.Is(err, ErrNotFound) {
		t.Errorf("Get as other user: got %v, want ErrNotFound", err)
	}
	if _, err := svc.Update(ctx, "bob", id, UpdateInput{Pinned: ptr(true)}); !errors.Is(err, ErrNotFound) {
		t.Errorf("Update as other user: got %v, want ErrNotFound", err)
	}
	if err := svc.Delete(ctx, "bob", id); !errors.Is(err, ErrNotFound) {
		t.Errorf("Delete as other user: got %v, want ErrNotFound", err)
	}
	if _, err := svc.Get(ctx, "alice", id); err != nil {
		t.Errorf("owner should still see the note: %v", err)
	}
}

func TestInvalidID(t *testing.T) {
	svc := newTestService()
	if _, err := svc.Get(context.Background(), "u1", "not-hex"); !errors.Is(err, ErrInvalidID) {
		t.Fatalf("got %v, want ErrInvalidID", err)
	}
}

func TestUpdatePartialAndClearContent(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	n, _ := svc.Create(ctx, "u1", CreateInput{Title: "t", Content: "body", Tags: []string{"a"}})

	updated, err := svc.Update(ctx, "u1", n.ID.Hex(), UpdateInput{
		Content: ptr(""), // explicitly clear
		Pinned:  ptr(true),
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Content != "" {
		t.Errorf("content should be cleared, got %q", updated.Content)
	}
	if !updated.Pinned {
		t.Error("pinned should be true")
	}
	if updated.Title != "t" || !reflect.DeepEqual(updated.Tags, []string{"a"}) {
		t.Errorf("untouched fields changed: %+v", updated)
	}
	if !updated.UpdatedAt.After(n.UpdatedAt) {
		t.Error("UpdatedAt should move forward")
	}
}

func TestUpdateRejectsEmptyAndInvalid(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()
	n, _ := svc.Create(ctx, "u1", CreateInput{Title: "keep me"})

	if _, err := svc.Update(ctx, "u1", n.ID.Hex(), UpdateInput{}); !errors.Is(err, ErrEmptyUpdate) {
		t.Errorf("got %v, want ErrEmptyUpdate", err)
	}

	_, err := svc.Update(ctx, "u1", n.ID.Hex(), UpdateInput{
		Title: ptr("   "),
		Tags:  ptr(make([]string, maxTags+1)),
	})
	var verrs ValidationErrors
	if !errors.As(err, &verrs) || len(verrs) != 2 {
		t.Fatalf("expected 2 validation errors, got %v", err)
	}

	got, _ := svc.Get(ctx, "u1", n.ID.Hex())
	if got.Title != "keep me" {
		t.Errorf("invalid update must not change the note, title = %q", got.Title)
	}
}

func TestListPaginationOrderingAndFilter(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	// Created oldest -> newest: n0 .. n4. n1 is pinned.
	var ids []string
	for i, title := range []string{"n0", "n1", "n2", "n3", "n4"} {
		tags := []string{"all"}
		if i%2 == 0 {
			tags = append(tags, "even")
		}
		n, err := svc.Create(ctx, "u1", CreateInput{Title: title, Tags: tags, Pinned: i == 1})
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, n.ID.Hex())
	}
	_ = ids
	// Another user's note must never appear.
	if _, err := svc.Create(ctx, "u2", CreateInput{Title: "other"}); err != nil {
		t.Fatal(err)
	}

	titles := func(r *ListResult) []string {
		out := make([]string, len(r.Notes))
		for i, n := range r.Notes {
			out[i] = n.Title
		}
		return out
	}

	page1, err := svc.List(ctx, "u1", ListParams{Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"n1", "n4"}; !reflect.DeepEqual(titles(page1), want) { // pinned first, then newest
		t.Errorf("page1 = %v, want %v", titles(page1), want)
	}
	if page1.Total != 5 || !page1.HasMore {
		t.Errorf("page1 meta wrong: %+v", page1)
	}

	last, _ := svc.List(ctx, "u1", ListParams{Limit: 2, Offset: 4})
	if len(last.Notes) != 1 || last.HasMore {
		t.Errorf("last page wrong: %+v", last)
	}

	even, _ := svc.List(ctx, "u1", ListParams{Tag: "  EVEN "})
	if even.Total != 3 {
		t.Errorf("tag filter total = %d, want 3", even.Total)
	}

	empty, _ := svc.List(ctx, "nobody", ListParams{})
	if empty.Notes == nil || len(empty.Notes) != 0 {
		t.Error("empty result should be a non-nil empty slice")
	}
}

func TestListParamsNormalize(t *testing.T) {
	p := ListParams{Limit: 1000, Offset: -5, Tag: "  GoLang "}
	p.Normalize()
	if p.Limit != maxLimit || p.Offset != 0 || p.Tag != "golang" {
		t.Errorf("unexpected normalization: %+v", p)
	}

	p = ListParams{}
	p.Normalize()
	if p.Limit != defaultLimit {
		t.Errorf("default limit = %d, want %d", p.Limit, defaultLimit)
	}
}

func TestDelete(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()
	n, _ := svc.Create(ctx, "u1", CreateInput{Title: "bye"})

	if err := svc.Delete(ctx, "u1", n.ID.Hex()); err != nil {
		t.Fatal(err)
	}
	if err := svc.Delete(ctx, "u1", n.ID.Hex()); !errors.Is(err, ErrNotFound) {
		t.Errorf("second delete: got %v, want ErrNotFound", err)
	}
}
