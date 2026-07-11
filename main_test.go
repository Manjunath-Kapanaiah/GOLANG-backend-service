package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ── Store Tests ───────────────────────────────────────────────────────────────

func TestStore_AddAndGetAll(t *testing.T) {
	s := NewStore()
	products := s.getAll()
	if len(products) != 3 {
		t.Errorf("expected 3 seeded products, got %d", len(products))
	}
}

func TestStore_GetByID(t *testing.T) {
	s := NewStore()
	p, ok := s.getByID(1)
	if !ok {
		t.Fatal("expected product with ID 1 to exist")
	}
	if p.ID != 1 {
		t.Errorf("expected ID 1, got %d", p.ID)
	}
}

func TestStore_GetByID_NotFound(t *testing.T) {
	s := NewStore()
	_, ok := s.getByID(999)
	if ok {
		t.Error("expected product 999 to not exist")
	}
}

func TestStore_Delete(t *testing.T) {
	s := NewStore()
	ok := s.delete(1)
	if !ok {
		t.Fatal("expected delete of product 1 to succeed")
	}
	_, exists := s.getByID(1)
	if exists {
		t.Error("expected product 1 to be gone after delete")
	}
}

func TestStore_Delete_NotFound(t *testing.T) {
	s := NewStore()
	ok := s.delete(999)
	if ok {
		t.Error("expected delete of non-existent product to return false")
	}
}

// ── HTTP Handler Tests ────────────────────────────────────────────────────────

func newTestServer() *Server {
	return NewServer(NewStore())
}

func TestHealthHandler(t *testing.T) {
	srv := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	srv.healthHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var resp HealthResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Status != "ok" {
		t.Errorf("expected status ok, got %s", resp.Status)
	}
	if resp.Service != "go-backend-service" {
		t.Errorf("unexpected service name: %s", resp.Service)
	}
}

func TestListProducts(t *testing.T) {
	srv := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/products", nil)
	w := httptest.NewRecorder()
	srv.listProducts(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestGetProduct_Found(t *testing.T) {
	srv := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/products/1", nil)
	w := httptest.NewRecorder()
	srv.getProduct(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestGetProduct_NotFound(t *testing.T) {
	srv := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/products/999", nil)
	w := httptest.NewRecorder()
	srv.getProduct(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestGetProduct_InvalidID(t *testing.T) {
	srv := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/products/abc", nil)
	w := httptest.NewRecorder()
	srv.getProduct(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCreateProduct_Success(t *testing.T) {
	srv := newTestServer()
	body := `{"name":"New Widget","price":14.99,"stock":30}`
	req := httptest.NewRequest(http.MethodPost, "/products", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.createProduct(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", w.Code)
	}
}

func TestCreateProduct_MissingName(t *testing.T) {
	srv := newTestServer()
	body := `{"price":14.99,"stock":30}`
	req := httptest.NewRequest(http.MethodPost, "/products", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	srv.createProduct(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCreateProduct_InvalidPrice(t *testing.T) {
	srv := newTestServer()
	body := `{"name":"Bad Widget","price":-5}`
	req := httptest.NewRequest(http.MethodPost, "/products", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	srv.createProduct(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestDeleteProduct_Success(t *testing.T) {
	srv := newTestServer()
	req := httptest.NewRequest(http.MethodDelete, "/products/1", nil)
	w := httptest.NewRecorder()
	srv.deleteProduct(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestDeleteProduct_NotFound(t *testing.T) {
	srv := newTestServer()
	req := httptest.NewRequest(http.MethodDelete, "/products/999", nil)
	w := httptest.NewRecorder()
	srv.deleteProduct(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}
