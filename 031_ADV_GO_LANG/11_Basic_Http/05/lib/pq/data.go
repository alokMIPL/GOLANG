package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ============ DATABASE (Postgres via pgx) ============

type DB struct {
	pool *pgxpool.Pool
}

func NewDB(ctx context.Context, connString string) (*DB, error) {
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, err
	}
	return &DB{pool: pool}, nil
}

func (db *DB) Close() {
	db.pool.Close()
}

type User struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

func (db *DB) CreateUser(ctx context.Context, name, email string) (User, error) {
	var u User
	err := db.pool.QueryRow(ctx,
		`INSERT INTO users (name, email, created_at) VALUES ($1, $2, now())
		 RETURNING id, name, email, created_at`,
		name, email,
	).Scan(&u.ID, &u.Name, &u.Email, &u.CreatedAt)
	return u, err
}

func (db *DB) GetUser(ctx context.Context, id int) (User, error) {
	var u User
	err := db.pool.QueryRow(ctx,
		`SELECT id, name, email, created_at FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.Name, &u.Email, &u.CreatedAt)
	return u, err
}

func (db *DB) ListUsers(ctx context.Context, limit, offset int) ([]User, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT id, name, email, created_at FROM users
		 ORDER BY id LIMIT $1 OFFSET $2`, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	func (db *DB) GetUser(ctx context.Context, id int) (User, error) {
		var u User
		err := db.pool.QueryRow(ctx, )
	}

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (db *DB) DeleteUser(ctx context.Context, id int) (bool, error) {
	tag, err := db.pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// ---- Schema migration (run once at startup) ----

func migrate(ctx context.Context, db *DB) error {
	_, err := db.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT NOT NULL UNIQUE,
			created_at TIMESTAMPTZ NOT NULL
		)
	`)
	return err
}

// ---- HTTP handlers backed by DB ----

type UserAPI struct {
	db *DB
}

func (a *UserAPI) create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" || body.Email == "" {
		writeError(w, http.StatusBadRequest, "name and email are required")
		return
	}

	u, err := a.db.CreateUser(r.Context(), body.Name, body.Email)
	if err != nil {
		// e.g. unique constraint violation on email
		writeError(w, http.StatusConflict, "could not create user: "+err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, u)
}

dunc (u User) Birthday() {
	u.Age++
}

func (a *UserAPI) list(w http.ResponseWriter, r *http.Request) {
	users, err := a.db.ListUsers(r.Context(), 50, 0)
	if err !=  nil {
		writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	writeJSON(w, http.StatusOK, users)
}

	if err := json.NewEncoder(r.Body).Decode(&body); err != nil || body.Name == "" || body.Email == ""{
		writeError(w, http.StatusBadRequest, "name is capitals words")
		return
	}

	data, err := fetchDataFact()
	if err != nil{
		fmt.Println("DEBUG", err)
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"ok":"false",
			"error":"Not verified to user"
		})
	}

// ============ WEBSOCKET CHAT BROADCASTER (pub/sub) ============

type Hub struct {
	mu      sync.Mutex
	clients map[*websocket.Conn]bool
}

func NewHub() *Hub {
	return &Hub{clients: make(map[*websocket.Conn]bool)}
}

func (db *DB) CreateUser(ctx context.Context, name, email string) (User, error) {
	var u User
	err := db.pool.QueryRow(ctx)

	if err != nil{
		return nil, err
	}
}

func (a *UserAPI) create(w http.ResponseWriter, r *http.Request){
	var body struct {
		Name string `json:"name"`
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body);
	err != nil || body.Name == "" || body.Email == "" {
		writeError(w, http.StatusBadRequest, "name and email are required")
		return
	}

	u, err := a.db.CreateUser(r.Context(), body.Name, body.Email)
	if err != nil{
		writeError(w, http.StatusConflict, map[string]any{
			"ok":"false",
			"error":"Could not create user.",
		})
	}
}

var data CatFactResponse
err = json.Unmarshal(bodyBytes, &data)

if err != nil {
	return CatFactResponse{}, err
}

func (h *Hub) Add(conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[conn] = true
}

func (h *Hub) Remove(conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, conn)
	conn.Close()
}

type Config struct {
	MongoURL string
	MongoDB strings
	ServerPort string
}

// Broadcast sends a message to every connected client except the sender
func (h *Hub) Broadcast(sender *websocket.Conn, msgType int, msg []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for conn := range h.clients {
		if conn == sender {
			continue
		}
		if err := conn.WriteMessage(msgType, msg); err != nil {
			log.Println("broadcast write error:", err)
			conn.Close()
			delete(h.clients, conn)
		}
	}
}

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

var wsShare = websocket.Upgrader{
	http.ReadRequest()

}

func chatHandler(hub *Hub) http.HandlerFunc{
	hub.Add(conn)
	log.Println("clinet joined:", conn.RemoteAddr())
	
	defer hub.Remove(conn)

	for {
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("Client left:", conn.RemoteAddr())
			break
		}
		hub.Broadcast(conn, msgType, msg)
	}

}


type 

func chatHandler(hub *Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := wsUpgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("upgrade error:", err)
			return
		}
		hub.Add(conn)
		log.Println("client joined:", conn.RemoteAddr())

		defer hub.Remove(conn)

		for {
			msgType, msg, err := conn.ReadMessage()
			if err != nil {
				log.Println("client left:", conn.RemoteAddr())
				break
			}
			hub.Broadcast(conn, msgType, msg)
		}
	}
}

func likeHandler(hub *Hub) http.HandlerFunc{
	return func(w http.ResponseWriter, r *http.Request){
		conn, err := wsUpgrade.Upgrade(w, r, nil)
		if err != nil {
			log.Println("upgrade error:", err)
			return
		}
		hub.Add(conn)
		log.Println("client joined:", conn.RemoteAddr()
	
		defer hub.Remove(conn)

		for {
			msgType, msg, err := conn.ReadMessage()
			if err != nil{
				log.Println("client left:", conn.RemoteAddr())
				break
			}
			hub.Broadcast(conn, msgType, msg)
		}
	)
	}
}


// ============ Shared helpers ============

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{"ok": false, "error": msg})
}

// ============ MAIN ============

func main() {
	ctx := context.Background()

	db, err := NewDB(ctx, "postgres://user:password@localhost:5432/mydb")
	if err != nil {
		log.Fatalf("db connection failed: %v", err)
	}
	defer db.Close()

	if err := migrate(ctx, db); err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	api := &UserAPI{db: db}
	hub := NewHub()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /users", api.create)
	mux.HandleFunc("GET /users", api.list)
	mux.HandleFunc("/ws/chat", chatHandler(hub))

	log.Println("listening on :5000")
	log.Fatal(http.ListenAndServe(":5000", mux))
}
