package test

import (
	"context"
	"encoding/json"
	"ft_0/server"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

type MockRelayServer struct {
	server    *httptest.Server
	sessions  sync.Map
	available bool
}

func NewMockRelayServer(available bool) *MockRelayServer {
	mock := &MockRelayServer{
		available: available,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/new", mock.handleNew)
	mux.HandleFunc("/join/", mock.handleJoin)
	mux.HandleFunc("/leave/", mock.handleLeave)

	mock.server = httptest.NewServer(mux)
	return mock
}

func (m *MockRelayServer) handleNew(w http.ResponseWriter, r *http.Request) {
	if !m.available {
		http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
		return
	}

	session := server.TransferSession{
		SessionID: server.GenerateID(),
		SenderID:  "test-sender",
	}
	m.sessions.Store(session.SessionID, &session)
	json.NewEncoder(w).Encode(session)
}

func (m *MockRelayServer) handleJoin(w http.ResponseWriter, r *http.Request) {
	if !m.available {
		http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) != 3 {
		http.Error(w, "Invalid session ID", http.StatusBadRequest)
		return
	}

	sessionID := parts[2]
	session, exists := m.sessions.Load(sessionID)
	if !exists {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(session)
}

func (m *MockRelayServer) handleLeave(w http.ResponseWriter, r *http.Request) {
	if !m.available {
		http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (m *MockRelayServer) URL() string {
	return m.server.URL
}

func (m *MockRelayServer) Close() {
	m.server.Close()
}

func TestSenderErrorHandling(t *testing.T) {
	originalServer := server.RELAY_SERVER
	originalProtocol := server.RELAY_PROTOCOL
	defer func() {
		server.RELAY_SERVER = originalServer
		server.RELAY_PROTOCOL = originalProtocol
	}()

	tests := []struct {
		name        string
		setupFunc   func() (string, error)
		mockServer  bool
		expectedErr string
		timeout     time.Duration
	}{
		{
			name: "non_existent_file",
			setupFunc: func() (string, error) {
				return "/path/to/nonexistent/file.txt", nil
			},
			mockServer:  true,
			expectedErr: "failed to access file",
			timeout:     time.Second,
		},
		{
			name: "relay_server_down",
			setupFunc: func() (string, error) {
				f, err := os.CreateTemp("", "test_file_*.txt")
				if err != nil {
					return "", err
				}
				f.Close()
				return f.Name(), nil
			},
			mockServer:  false,
			expectedErr: "failed to create session",
			timeout:     2 * time.Second,
		},
		{
			name: "transfer_cancelled",
			setupFunc: func() (string, error) {
				f, err := os.CreateTemp("", "test_file_*.txt")
				if err != nil {
					return "", err
				}
				f.WriteString("test data")
				f.Close()
				return f.Name(), nil
			},
			mockServer:  true,
			expectedErr: "transfer cancelled",
			timeout:     2 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testCtx, testCancel := context.WithTimeout(context.Background(), tt.timeout)
			defer testCancel()

			var mock *MockRelayServer
			if tt.mockServer {
				mock = NewMockRelayServer(true)
				server.RELAY_SERVER = mock.URL()[7:]
				server.RELAY_PROTOCOL = "http"
				defer mock.Close()
			} else {
				server.RELAY_SERVER = "localhost:0"
			}

			filepath, err := tt.setupFunc()
			if err != nil {
				t.Fatalf("Failed to setup test: %v", err)
			}
			if filepath != "/path/to/nonexistent/file.txt" {
				defer os.Remove(filepath)
			}

			progressChan := make(chan server.SendProgress)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			go server.StartSender(filepath, progressChan, ctx)

			var lastError error
			done := make(chan bool)

			go func() {
				var waitingForReceiver bool
				for progress := range progressChan {
					if progress.Error != nil {
						lastError = progress.Error
						done <- true
						return
					}

					if tt.name == "transfer_cancelled" && progress.State == server.StateWaitingForReceiver {
						waitingForReceiver = true
						cancel()
					}
				}

				if !waitingForReceiver {
					done <- true
				}
			}()

			select {
			case <-testCtx.Done():
				t.Fatal("Test timed out")
			case <-done:
				if lastError == nil {
					t.Fatal("Expected an error but got none")
				}
				if msg := lastError.Error(); !strings.Contains(msg, tt.expectedErr) {
					t.Errorf("Expected error containing %q, got %q", tt.expectedErr, msg)
				}
			}
		})
	}
}

func TestConnectionHandling(t *testing.T) {
	cm := server.NewConnectionManager()

	t.Run("connection_timeout", func(t *testing.T) {
		client, sv := net.Pipe()
		defer client.Close()
		defer sv.Close()

		conn := cm.NewConnection(client)
		_, err := conn.WaitForResponse(100 * time.Millisecond)
		if err != server.ErrConnectionTimeout {
			t.Errorf("Expected timeout error, got: %v", err)
		}
	})

	t.Run("connection_cancellation", func(t *testing.T) {
		client, server := net.Pipe()
		defer client.Close()
		defer server.Close()

		conn := cm.NewConnection(client)
		ctx, cancel := context.WithCancel(context.Background())

		go func() {
			time.Sleep(100 * time.Millisecond)
			cancel()
		}()

		err := conn.SendWithContext(ctx, []byte("test"))
		if err != context.Canceled {
			t.Errorf("Expected context.Canceled error, got: %v", err)
		}
	})
}

func TestSessionManager(t *testing.T) {
	mock := NewMockRelayServer(true)
	defer mock.Close()

	originalServer := server.RELAY_SERVER
	originalProtocol := server.RELAY_PROTOCOL
	server.RELAY_SERVER = mock.URL()[7:]
	server.RELAY_PROTOCOL = "http"
	defer func() {
		server.RELAY_SERVER = originalServer
		server.RELAY_PROTOCOL = originalProtocol
	}()

	sm := server.NewSessionManager()

	t.Run("create_session", func(t *testing.T) {
		ctx := context.Background()
		session, err := sm.CreateSession(ctx)
		if err != nil {
			t.Errorf("Failed to create session: %v", err)
		}
		if session.SessionID == "" {
			t.Error("Session ID should not be empty")
		}
	})

	t.Run("join_session", func(t *testing.T) {
		ctx := context.Background()
		session, err := sm.CreateSession(ctx)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		joinedSession, err := sm.JoinSession(ctx, session.SessionID)
		if err != nil {
			t.Errorf("Failed to join session: %v", err)
		}
		if joinedSession.SessionID != session.SessionID {
			t.Error("Joined session ID doesn't match created session")
		}
	})
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
