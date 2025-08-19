package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func setupTestServer(t *testing.T) *httptest.Server {
	setupTestDB()

	r := setupRouter()
	return httptest.NewServer(r)
}

func TestIntegrationTodoLifecycle(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()
	defer cleanupTestDB()

	// 1. Create a new todo
	createPayload := Todo{
		Title:     "Integration Test Todo",
		Completed: false,
	}
	createBody, _ := json.Marshal(createPayload)

	createResp, err := http.Post(
		fmt.Sprintf("%s/todos", server.URL),
		"application/json",
		bytes.NewBuffer(createBody),
	)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, createResp.StatusCode)

	var createdTodo Todo
	json.NewDecoder(createResp.Body).Decode(&createdTodo)
	createResp.Body.Close()

	assert.NotZero(t, createdTodo.ID)
	assert.Equal(t, createPayload.Title, createdTodo.Title)
	assert.Equal(t, createPayload.Completed, createdTodo.Completed)

	// 2. Get the todo list and verify the created todo
	listResp, err := http.Get(fmt.Sprintf("%s/todos", server.URL))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, listResp.StatusCode)

	var listResponse PaginatedResponse
	json.NewDecoder(listResp.Body).Decode(&listResponse)
	listResp.Body.Close()

	assert.Equal(t, 1, listResponse.TotalItems)
	assert.Equal(t, createdTodo.ID, listResponse.Items[0].ID)

	// 3. Update the todo
	updatePayload := Todo{
		Title:     "Updated Integration Test Todo",
		Completed: true,
	}
	updateBody, _ := json.Marshal(updatePayload)

	updateReq, _ := http.NewRequest(
		http.MethodPut,
		fmt.Sprintf("%s/todos/%d", server.URL, createdTodo.ID),
		bytes.NewBuffer(updateBody),
	)
	updateResp, err := http.DefaultClient.Do(updateReq)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, updateResp.StatusCode)

	var updatedTodo Todo
	json.NewDecoder(updateResp.Body).Decode(&updatedTodo)
	updateResp.Body.Close()

	assert.Equal(t, createdTodo.ID, updatedTodo.ID)
	assert.Equal(t, updatePayload.Title, updatedTodo.Title)
	assert.Equal(t, updatePayload.Completed, updatedTodo.Completed)

	// 4. Test pagination
	// Create more todos for pagination testing
	for i := 0; i < 5; i++ {
		payload := Todo{
			Title:     fmt.Sprintf("Pagination Todo %d", i),
			Completed: false,
		}
		body, _ := json.Marshal(payload)
		http.Post(
			fmt.Sprintf("%s/todos", server.URL),
			"application/json",
			bytes.NewBuffer(body),
		)
	}

	// Test pagination with limit
	paginatedResp, err := http.Get(fmt.Sprintf("%s/todos?page=1&limit=3", server.URL))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, paginatedResp.StatusCode)

	var paginatedResult PaginatedResponse
	json.NewDecoder(paginatedResp.Body).Decode(&paginatedResult)
	paginatedResp.Body.Close()

	assert.Equal(t, 6, paginatedResult.TotalItems)
	assert.Equal(t, 3, len(paginatedResult.Items))
	assert.Equal(t, 1, paginatedResult.Page)
	assert.Equal(t, 3, paginatedResult.Limit)

	// 5. Delete the todo
	deleteReq, _ := http.NewRequest(
		http.MethodDelete,
		fmt.Sprintf("%s/todos/%d", server.URL, createdTodo.ID),
		nil,
	)
	deleteResp, err := http.DefaultClient.Do(deleteReq)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, deleteResp.StatusCode)
	deleteResp.Body.Close()

	// 6. Verify deletion
	getResp, err := http.Get(fmt.Sprintf("%s/todos/%d", server.URL, createdTodo.ID))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, getResp.StatusCode)
}

func TestIntegrationHealthCheck(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()
	defer cleanupTestDB()

	resp, err := http.Get(fmt.Sprintf("%s/health", server.URL))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var response map[string]string
	json.NewDecoder(resp.Body).Decode(&response)
	resp.Body.Close()

	assert.Equal(t, "healthy", response["status"])
}
