package server

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