package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type SessionManager struct {
	sessions sync.Map
	client   *http.Client
}

func NewSessionManager() *SessionManager {
	return &SessionManager{
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (sm *SessionManager) CreateSession(ctx context.Context) (*TransferSession, error) {
	req, err := http.NewRequestWithContext(ctx, "POST",
		RELAY_PROTOCOL+"://"+RELAY_SERVER+"/new", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	resp, err := sm.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to create session: server returned %d", resp.StatusCode)
	}

	var session TransferSession
	if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
		return nil, fmt.Errorf("failed to decode session: %v", err)
	}

	sm.sessions.Store(session.SessionID, &session)
	return &session, nil
}

func (sm *SessionManager) JoinSession(ctx context.Context, sessionID string) (*TransferSession, error) {
	if sessionID == "" {
		return nil, SessionError{
			Code:    "INVALID_SESSION",
			Message: "Please enter a valid session ID",
		}
	}

	req, err := http.NewRequestWithContext(ctx, "GET",
		RELAY_PROTOCOL+"://"+RELAY_SERVER+"/join/"+sessionID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	resp, err := sm.client.Do(req)
	if err != nil {
		return nil, ErrRelayServerDown
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusNotFound:
		return nil, ErrSessionNotFound
	case http.StatusConflict:
		return nil, ErrSessionConflict
	case http.StatusOK:
	default:
		return nil, SessionError{
			Code:    "UNEXPECTED_ERROR",
			Message: fmt.Sprintf("Unexpected error (status %d)", resp.StatusCode),
		}
	}

	var session TransferSession
	if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
		return nil, fmt.Errorf("invalid session data")
	}

	sm.sessions.Store(session.SessionID, &session)
	return &session, nil
}

func (sm *SessionManager) LeaveSession(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, "GET",
		RELAY_PROTOCOL+"://"+RELAY_SERVER+"/leave/"+sessionID, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	resp, err := sm.client.Do(req)
	if err != nil {
		return fmt.Errorf("couldn't disconnect cleanly: %v", err)
	}
	defer resp.Body.Close()

	sm.sessions.Delete(sessionID)
	return nil
}
