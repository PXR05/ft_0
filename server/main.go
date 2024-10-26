package server

import (
	"crypto/rand"
	"encoding/hex"
)

const (
	CHUNK_SIZE     = 1024 * 32
	RELAY_PROTOCOL = "http"
	RELAY_SERVER   = "localhost:3000"
	TRANSFER_PORT  = "3001"
)

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

func generateID() string {
	bytes := make([]byte, 3)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
