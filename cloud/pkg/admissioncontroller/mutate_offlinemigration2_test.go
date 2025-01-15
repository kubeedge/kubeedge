package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Mock serve function to capture the arguments passed to it
var mockServe = func(w http.ResponseWriter, r *http.Request, handler func(*http.Request) ([]byte, error)) {
	// Capture the arguments or perform assertions here
	// For simplicity, we'll just call the handler and write the result to the response
	response, err := handler(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(response)
}

// Replace the original serve function with the mock for testing
func init() {
	serve = mockServe
}

func TestServeOfflineMigration(t *testing.T) {
	// Create a request to pass to our function
	req, err := http.NewRequest("GET", "/offline-migration", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create a ResponseRecorder to record the response
	rr := httptest.NewRecorder()

	// Call the function under test
	serveOfflineMigration(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Check the response body (if applicable)
	// expected := `{"message":"success"}`
	// if rr.Body.String() != expected {
	//     t.Errorf("handler returned unexpected body: got %v want %v",
	//         rr.Body.String(), expected)
	// }
}