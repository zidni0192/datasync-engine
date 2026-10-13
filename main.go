package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

type Event struct {
	ID      int64  `json:"id"`
	Topic   string `json:"topic"`
	Payload string `json:"payload"`
}

type EventStore struct {
	mu     sync.RWMutex
	nextID int64
	events []Event
}

func (s *EventStore) Append(topic, payload string) Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID++
	event := Event{ID: s.nextID, Topic: topic, Payload: payload}
	s.events = append(s.events, event)
	return event
}

func (s *EventStore) List(after int64, limit int) []Event {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Event, 0, limit)
	for _, event := range s.events {
		if event.ID > after {
			result = append(result, event)
			if len(result) == limit {
				break
			}
		}
	}
	return result
}

func NewHandler(store *EventStore) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			var input struct {
				Topic   string `json:"topic"`
				Payload string `json:"payload"`
			}
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				http.Error(w, "invalid JSON body", http.StatusBadRequest)
				return
			}
			input.Topic = strings.TrimSpace(input.Topic)
			if input.Topic == "" || input.Payload == "" {
				http.Error(w, "topic and payload are required", http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(store.Append(input.Topic, input.Payload))
		case http.MethodGet:
			after, err := queryInt(r, "after", 0)
			if err != nil || after < 0 {
				http.Error(w, "after must be a non-negative integer", http.StatusBadRequest)
				return
			}
			limit, err := queryInt(r, "limit", 100)
			if err != nil || limit < 1 || limit > 500 {
				http.Error(w, "limit must be between 1 and 500", http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(store.List(int64(after), limit))
		default:
			w.Header().Set("Allow", "GET, POST")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	return mux
}

func queryInt(r *http.Request, name string, fallback int) (int, error) {
	value := r.URL.Query().Get(name)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", name, err)
	}
	return parsed, nil
}

func main() {
	store := &EventStore{}
	if err := http.ListenAndServe(":8080", NewHandler(store)); err != nil && !errors.Is(err, http.ErrServerClosed) {
		panic(err)
	}
}
