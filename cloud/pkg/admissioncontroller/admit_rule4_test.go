package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Mock serve function for testing
var mockServe = func(w http.ResponseWriter, r *http.Request, admitRule func(http.ResponseWriter, *http.Request)) {
	// Simulate the behavior of the serve function
	admitRule(w, r)
}

// Mock admitRule function for testing
var mockAdmitRule = func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Admit Rule Executed"))
}

func TestServeRule(t *testing.T) {
	// Replace the actual serve function with the mockServe function
	serve = mockServe

	// Create a request to pass to our handler
	req, err := http.NewRequest("GET", "/rule", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create a ResponseRecorder to record the response
	rr := httptest.NewRecorder()

	// Call the serveRule function directly
	serveRule(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Check the response body
	expected := "Admit Rule Executed"
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), expected)
	}
}