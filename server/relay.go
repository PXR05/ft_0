package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type RelayServer struct {
	sessions    *sync.Map
	server      *http.Server
	stopChan    chan struct{}
	stoppedChan chan struct{}
	logChan     chan string
	Messages    []string
	IsRunning   bool
	mu          sync.Mutex
}

func NewRelayServer() *RelayServer {
	return &RelayServer{
		sessions:  &sync.Map{},
		Messages:  make([]string, 0),
		IsRunning: false,
	}
}

func (s *RelayServer) Start() {
	s.mu.Lock()
	if s.IsRunning {
		s.mu.Unlock()
		return
	}

	s.logChan = make(chan string, 100)
	s.stopChan = make(chan struct{})
	s.stoppedChan = make(chan struct{})
	s.IsRunning = true
	s.mu.Unlock()

	if s.server == nil {
		mux := http.NewServeMux()

		logRequest := func(handler http.HandlerFunc) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				s.logChan <- fmt.Sprintf("%d: %s to %s from %s", time.Now().Unix(), r.Method, r.URL.Path, r.RemoteAddr)
				handler(w, r)
			}
		}

		mux.HandleFunc("/new", logRequest(func(w http.ResponseWriter, r *http.Request) {
			sessionID := GenerateID()
			senderID := GenerateID()
			session := &TransferSession{
				SessionID: sessionID,
				SenderID:  senderID,
			}
			s.sessions.Store(sessionID, session)
			json.NewEncoder(w).Encode(session)
		}))

		mux.HandleFunc("/join/", logRequest(func(w http.ResponseWriter, r *http.Request) {
			parts := strings.Split(r.URL.Path, "/")
			if len(parts) != 3 || parts[2] == "" {
				http.Error(w, "Invalid session ID format", http.StatusBadRequest)
				return
			}

			sessionID := parts[2]
			session, exists := s.sessions.Load(sessionID)
			if !exists {
				http.Error(w, fmt.Sprintf("Session '%s' not found", sessionID), http.StatusNotFound)
				return
			}

			if session.(*TransferSession).ReceiverID != "" {
				http.Error(w, "This session already has an active receiver", http.StatusConflict)
				return
			}

			session.(*TransferSession).ReceiverID = GenerateID()
			json.NewEncoder(w).Encode(session)
		}))

		mux.HandleFunc("/leave/", logRequest(func(w http.ResponseWriter, r *http.Request) {
			parts := strings.Split(r.URL.Path, "/")
			if len(parts) != 3 {
				http.Error(w, "Invalid session ID", http.StatusBadRequest)
				return
			}

			sessionID := parts[2]
			session, exists := s.sessions.Load(sessionID)
			if !exists {
				http.Error(w, "Session not found", http.StatusNotFound)
				return
			}

			if session.(*TransferSession).ReceiverID == "" {
				http.Error(w, "Session does not have a receiver", http.StatusConflict)
				return
			}

			session.(*TransferSession).ReceiverID = ""
			json.NewEncoder(w).Encode(session)
		}))

		s.server = &http.Server{
			Addr:    RELAY_SERVER,
			Handler: mux,
		}
	}

	go func() {
		s.logChan <- "Starting server on port " + RELAY_SERVER
		if err := s.server.ListenAndServe(); err != http.ErrServerClosed {
			s.mu.Lock()
			if s.IsRunning {
				s.logChan <- fmt.Sprintf("Server error: %v", err)
			}
			s.mu.Unlock()
		}
	}()
}

func (s *RelayServer) Stop() {
	s.mu.Lock()
	if !s.IsRunning {
		s.mu.Unlock()
		return
	}
	s.IsRunning = false
	s.mu.Unlock()

	close(s.stopChan)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if s.server != nil {
		if err := s.server.Shutdown(ctx); err != nil {
			s.server.Close()
		}
	}

	s.server = nil
	close(s.logChan)
}

type RelayLogMsg string

func CheckRelayLogs(relay *RelayServer) tea.Cmd {
	return func() tea.Msg {
		if relay == nil || relay.logChan == nil {
			return nil
		}
		select {
		case msg, ok := <-relay.logChan:
			if !ok {
				return nil
			}
			return RelayLogMsg(msg)
		case <-time.After(100 * time.Millisecond):
			return nil
		}
	}
}
