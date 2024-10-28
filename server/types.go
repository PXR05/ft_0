package server

import "fmt"

type TransferSession struct {
	SessionID  string `json:"session_id"`
	SenderID   string `json:"sender_id"`
	ReceiverID string `json:"receiver_id"`
}

type FileMetadata struct {
	Name     string
	Size     int64
	SenderIP string
}

type TransferState int

const (
	StateInitializing TransferState = iota
	StateWaitingForReceiver
	StateTransferring
	StateReceiving
	StateCompleted
	StateError
	StateCancelled
)

type SessionError struct {
	Code    string
	Message string
}

func (e SessionError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

var (
	ErrSessionNotFound = SessionError{
		Code:    "SESSION_NOT_FOUND",
		Message: "Session not found - check the ID and try again",
	}
	ErrSessionConflict = SessionError{
		Code:    "SESSION_CONFLICT",
		Message: "Session already has a receiver",
	}
	ErrConnectionTimeout = SessionError{
		Code:    "CONNECTION_TIMEOUT",
		Message: "Connection timed out - please try again",
	}
	ErrTransferRejected = SessionError{
		Code:    "TRANSFER_REJECTED",
		Message: "Transfer was rejected by receiver",
	}
	ErrRelayServerDown = SessionError{
		Code:    "RELAY_SERVER_DOWN",
		Message: "Could not connect to relay server - is it running?",
	}
)
