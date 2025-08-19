package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	bolt "go.etcd.io/bbolt"
)

// TestMain é usado para setup e teardown dos testes
func TestMain(m *testing.M) {
	// Setup
	setupTestDB()

	// Run tests
	code := m.Run()

	// Cleanup
	cleanupTestDB()

	os.Exit(code)
}

func setupTestDB() {
	if db != nil {
		db.Close()
	}

	os.Remove("test.db")

	var err error
	db, err = bolt.Open("test.db", 0600, nil)
	if err != nil {
		panic(err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("todos"))
		return err
	})
	if err != nil {
		panic(err)
	}
}

func cleanupTestDB() {
	if db != nil {
		db.Close()
		db = nil
	}
	os.Remove("test.db")
}

func clearBucket(t *testing.T) {
	setupTestDB()
	err := db.Update(func(tx *bolt.Tx) error {
		err := tx.DeleteBucket([]byte("todos"))
		if err != nil {
			return err
		}
		_, err = tx.CreateBucket([]byte("todos"))
		return err
	})
	assert.NoError(t, err)
}

func Test_itob(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected []byte
	}{
		{
			name:     "convert positive number",
			input:    42,
			expected: []byte{0, 0, 0, 0, 0, 0, 0, 42},
		},
		{
			name:     "convert zero",
			input:    0,
			expected: []byte{0, 0, 0, 0, 0, 0, 0, 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := itob(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHealthCheck(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	healthCheck(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "healthy", response["status"])
}

func TestCreateTodo(t *testing.T) {
	clearBucket(t)

	tests := []struct {
		name           string
		payload        Todo
		expectedStatus int
		expectedError  bool
	}{
		{
			name: "valid todo",
			payload: Todo{
				Title:     "Test todo",
				Completed: false,
			},
			expectedStatus: http.StatusCreated,
			expectedError:  false,
		},
		{
			name: "empty title",
			payload: Todo{
				Title:     "",
				Completed: false,
			},
			expectedStatus: http.StatusCreated,
			expectedError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload, err := json.Marshal(tt.payload)
			assert.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/todos", bytes.NewBuffer(payload))
			w := httptest.NewRecorder()

			createTodo(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if !tt.expectedError {
				var response Todo
				err = json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.NotZero(t, response.ID)
				assert.Equal(t, tt.payload.Title, response.Title)
				assert.Equal(t, tt.payload.Completed, response.Completed)
			}
		})
	}
}

func TestGetTodos(t *testing.T) {
	clearBucket(t)

	todos := []Todo{
		{Title: "Todo 1", Completed: false},
		{Title: "Todo 2", Completed: true},
		{Title: "Todo 3", Completed: false},
	}

	for _, todo := range todos {
		payload, _ := json.Marshal(todo)
		req := httptest.NewRequest(http.MethodPost, "/todos", bytes.NewBuffer(payload))
		w := httptest.NewRecorder()
		createTodo(w, req)
	}

	tests := []struct {
		name          string
		page          string
		limit         string
		expectedCount int
		expectedPage  int
	}{
		{
			name:          "default pagination",
			page:          "",
			limit:         "",
			expectedCount: 3,
			expectedPage:  1,
		},
		{
			name:          "custom page and limit",
			page:          "1",
			limit:         "2",
			expectedCount: 2,
			expectedPage:  1,
		},
		{
			name:          "page 2 with limit 2",
			page:          "2",
			limit:         "2",
			expectedCount: 1,
			expectedPage:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/todos"
			if tt.page != "" || tt.limit != "" {
				url = fmt.Sprintf("/todos?page=%s&limit=%s", tt.page, tt.limit)
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()

			getTodos(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			var response PaginatedResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedCount, len(response.Items))
			assert.Equal(t, tt.expectedPage, response.Page)
			assert.Equal(t, 3, response.TotalItems)
		})
	}
}

func TestUpdateTodo(t *testing.T) {
	clearBucket(t)

	initial := Todo{Title: "Initial todo", Completed: false}
	payload, _ := json.Marshal(initial)
	req := httptest.NewRequest(http.MethodPost, "/todos", bytes.NewBuffer(payload))
	w := httptest.NewRecorder()
	createTodo(w, req)

	var created Todo
	json.Unmarshal(w.Body.Bytes(), &created)

	tests := []struct {
		name           string
		id             int
		payload        Todo
		expectedStatus int
		expectedError  bool
	}{
		{
			name: "valid update",
			id:   created.ID,
			payload: Todo{
				Title:     "Updated todo",
				Completed: true,
			},
			expectedStatus: http.StatusOK,
			expectedError:  false,
		},
		{
			name: "non-existent id",
			id:   99999,
			payload: Todo{
				Title:     "Updated todo",
				Completed: true,
			},
			expectedStatus: http.StatusOK,
			expectedError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload, err := json.Marshal(tt.payload)
			assert.NoError(t, err)

			req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/todos/%d", tt.id), bytes.NewBuffer(payload))
			w := httptest.NewRecorder()

			router := mux.NewRouter()
			router.HandleFunc("/todos/{id}", updateTodo).Methods("PUT")
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if !tt.expectedError {
				var response Todo
				err = json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.id, response.ID)
				assert.Equal(t, tt.payload.Title, response.Title)
				assert.Equal(t, tt.payload.Completed, response.Completed)
			}
		})
	}
}

func TestDeleteTodo(t *testing.T) {
	clearBucket(t)

	initial := Todo{Title: "Todo to delete", Completed: false}
	payload, _ := json.Marshal(initial)
	req := httptest.NewRequest(http.MethodPost, "/todos", bytes.NewBuffer(payload))
	w := httptest.NewRecorder()
	createTodo(w, req)

	var created Todo
	json.Unmarshal(w.Body.Bytes(), &created)

	tests := []struct {
		name           string
		id             int
		expectedStatus int
	}{
		{
			name:           "valid delete",
			id:             created.ID,
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "non-existent id",
			id:             99999,
			expectedStatus: http.StatusNoContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/todos/%d", tt.id), nil)
			w := httptest.NewRecorder()

			router := mux.NewRouter()
			router.HandleFunc("/todos/{id}", deleteTodo).Methods("DELETE")
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}
