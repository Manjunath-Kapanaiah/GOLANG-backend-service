package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

// ── Models ────────────────────────────────────────────────────────────────────

type Product struct {
	ID        int     `json:"id"`
	Name      string  `json:"name"`
	Price     float64 `json:"price"`
	Stock     int     `json:"stock"`
	CreatedAt string  `json:"created_at"`
}

type APIResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

type HealthResponse struct {
	Status    string `json:"status"`
	Service   string `json:"service"`
	Version   string `json:"version"`
	Timestamp string `json:"timestamp"`
	Uptime    string `json:"uptime"`
}

// ── In-memory store ────────────────────────────────────────────────────────────

type Store struct {
	mu       sync.RWMutex
	products map[int]Product
	nextID   int
}

func NewStore() *Store {
	s := &Store{
		products: make(map[int]Product),
		nextID:   1,
	}
	s.add(Product{Name: "Widget A", Price: 9.99, Stock: 100})
	s.add(Product{Name: "Gadget B", Price: 24.99, Stock: 50})
	s.add(Product{Name: "Doohickey C", Price: 4.99, Stock: 200})
	return s
}

func (s *Store) add(p Product) Product {
	s.mu.Lock()
	defer s.mu.Unlock()
	p.ID = s.nextID
	p.CreatedAt = time.Now().Format(time.RFC3339)
	s.products[s.nextID] = p
	s.nextID++
	return p
}

func (s *Store) getAll() []Product {
	s.mu.RLock()
	defer s.mu.RUnlock()
	list := make([]Product, 0, len(s.products))
	for _, p := range s.products {
		list = append(list, p)
	}
	return list
}

func (s *Store) getByID(id int) (Product, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.products[id]
	return p, ok
}

func (s *Store) delete(id int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.products[id]; !ok {
		return false
	}
	delete(s.products, id)
	return true
}

// ── Server ────────────────────────────────────────────────────────────────────

type Server struct {
	store     *Store
	startTime time.Time
	logger    *log.Logger
}

func NewServer(store *Store) *Server {
	return &Server{
		store:     store,
		startTime: time.Now(),
		logger:    log.New(os.Stdout, "[GO-SERVICE] ", log.LstdFlags),
	}
}

func (srv *Server) respond(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}

// GET /health
func (srv *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	srv.logger.Printf("%s %s", r.Method, r.URL.Path)
	uptime := time.Since(srv.startTime).Round(time.Second)
	srv.respond(w, http.StatusOK, HealthResponse{
		Status:    "ok",
		Service:   "go-backend-service",
		Version:   "1.0.0",
		Timestamp: time.Now().Format(time.RFC3339),
		Uptime:    uptime.String(),
	})
}

// GET /products
func (srv *Server) listProducts(w http.ResponseWriter, r *http.Request) {
	srv.logger.Printf("%s %s", r.Method, r.URL.Path)
	srv.respond(w, http.StatusOK, APIResponse{
		Status: "success",
		Data:   srv.store.getAll(),
	})
}

// GET /products/{id}
func (srv *Server) getProduct(w http.ResponseWriter, r *http.Request) {
	srv.logger.Printf("%s %s", r.Method, r.URL.Path)
	idStr := r.URL.Path[len("/products/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		srv.respond(w, http.StatusBadRequest, APIResponse{Status: "error", Message: "invalid product ID"})
		return
	}
	p, ok := srv.store.getByID(id)
	if !ok {
		srv.respond(w, http.StatusNotFound, APIResponse{Status: "error", Message: fmt.Sprintf("product %d not found", id)})
		return
	}
	srv.respond(w, http.StatusOK, APIResponse{Status: "success", Data: p})
}

// POST /products
func (srv *Server) createProduct(w http.ResponseWriter, r *http.Request) {
	srv.logger.Printf("%s %s", r.Method, r.URL.Path)
	var p Product
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		srv.respond(w, http.StatusBadRequest, APIResponse{Status: "error", Message: "invalid request body"})
		return
	}
	if p.Name == "" {
		srv.respond(w, http.StatusBadRequest, APIResponse{Status: "error", Message: "name is required"})
		return
	}
	if p.Price <= 0 {
		srv.respond(w, http.StatusBadRequest, APIResponse{Status: "error", Message: "price must be greater than 0"})
		return
	}
	created := srv.store.add(p)
	srv.respond(w, http.StatusCreated, APIResponse{Status: "success", Message: "product created", Data: created})
}

// DELETE /products/{id}
func (srv *Server) deleteProduct(w http.ResponseWriter, r *http.Request) {
	srv.logger.Printf("%s %s", r.Method, r.URL.Path)
	idStr := r.URL.Path[len("/products/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		srv.respond(w, http.StatusBadRequest, APIResponse{Status: "error", Message: "invalid product ID"})
		return
	}
	if !srv.store.delete(id) {
		srv.respond(w, http.StatusNotFound, APIResponse{Status: "error", Message: fmt.Sprintf("product %d not found", id)})
		return
	}
	srv.respond(w, http.StatusOK, APIResponse{Status: "success", Message: fmt.Sprintf("product %d deleted", id)})
}

// Routes
func (srv *Server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", srv.healthHandler)
	mux.HandleFunc("/products", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			srv.listProducts(w, r)
		case http.MethodPost:
			srv.createProduct(w, r)
		default:
			srv.respond(w, http.StatusMethodNotAllowed, APIResponse{Status: "error", Message: "method not allowed"})
		}
	})
	mux.HandleFunc("/products/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			srv.getProduct(w, r)
		case http.MethodDelete:
			srv.deleteProduct(w, r)
		default:
			srv.respond(w, http.StatusMethodNotAllowed, APIResponse{Status: "error", Message: "method not allowed"})
		}
	})
	return mux
}

// ── Main ──────────────────────────────────────────────────────────────────────

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	store := NewStore()
	srv := NewServer(store)

	srv.logger.Printf("Starting go-backend-service on port %s", port)
	srv.logger.Printf("Endpoints:")
	srv.logger.Printf("  GET    /health")
	srv.logger.Printf("  GET    /products")
	srv.logger.Printf("  POST   /products")
	srv.logger.Printf("  GET    /products/{id}")
	srv.logger.Printf("  DELETE /products/{id}")

	httpSrv := &http.Server{
		Addr:         ":" + port,
		Handler:      srv.routes(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	if err := httpSrv.ListenAndServe(); err != nil {
		srv.logger.Fatalf("Server failed: %v", err)
	}
}
