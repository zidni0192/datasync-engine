package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAppendAndReadEvents(t *testing.T) {
	server := httptest.NewServer(NewHandler(&EventStore{}))
	defer server.Close()

	for _, body := range []string{`{"topic":"orders","payload":"created"}`, `{"topic":"orders","payload":"shipped"}`} {
		response, err := http.Post(server.URL+"/events", "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		if response.StatusCode != http.StatusCreated {
			t.Fatalf("POST status = %d, want %d", response.StatusCode, http.StatusCreated)
		}
		response.Body.Close()
	}

	response, err := http.Get(server.URL + "/events?after=1&limit=1")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var events []Event
	if err := json.NewDecoder(response.Body).Decode(&events); err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].ID != 2 || events[0].Payload != "shipped" {
		t.Fatalf("events = %#v, want second event only", events)
	}
}

func TestPostRejectsInvalidEvents(t *testing.T) {
	for _, body := range []string{`not-json`, `{"topic":" ","payload":"value"}`} {
		recorder := httptest.NewRecorder()
		NewHandler(&EventStore{}).ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/events", strings.NewReader(body)))
		if recorder.Code != http.StatusBadRequest {
			t.Errorf("body %q status = %d, want %d", body, recorder.Code, http.StatusBadRequest)
		}
	}
}

func TestGetRejectsInvalidQuery(t *testing.T) {
	for _, target := range []string{"/events?after=-1", "/events?limit=0", "/events?limit=501"} {
		recorder := httptest.NewRecorder()
		NewHandler(&EventStore{}).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, target, nil))
		if recorder.Code != http.StatusBadRequest {
			t.Errorf("target %q status = %d, want %d", target, recorder.Code, http.StatusBadRequest)
		}
	}
}
