package server

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type SendProgress struct {
	State      TransferState
	Speed      float64
	BytesSent  int64
	TotalBytes int64
	SessionID  string
	Error      error
}

func StartSender(filepath string, progressChan chan<- SendProgress, ctx context.Context) {
	go func() {
		defer close(progressChan)

		if err := validateFile(filepath); err != nil {
			progressChan <- SendProgress{
				State: StateError,
				Error: err,
			}
			return
		}

		progressChan <- SendProgress{State: StateInitializing}

		sm := NewSessionManager()
		session, err := sm.CreateSession(ctx)
		if err != nil {
			progressChan <- SendProgress{
				State: StateError,
				Error: err,
			}
			return
		}

		progressChan <- SendProgress{
			State:     StateWaitingForReceiver,
			SessionID: session.SessionID,
		}

		cm := NewConnectionManager()
		conn, err := waitForReceiver(ctx, cm)
		if err != nil {
			progressChan <- SendProgress{
				State: StateError,
				Error: err,
			}
			return
		}

		sendFile(filepath, conn, progressChan, ctx)
	}()
}

func validateFile(filepath string) error {
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		return fmt.Errorf("failed to access file '%s': file does not exist", filepath)
	}
	return nil
}

func waitForReceiver(ctx context.Context, cm *ConnectionManager) (*Connection, error) {
	listener, err := net.Listen("tcp", ":"+TRANSFER_PORT)
	if err != nil {
		return nil, fmt.Errorf("failed to start listener: %v", err)
	}
	defer listener.Close()

	connChan := make(chan net.Conn)
	errChan := make(chan error)

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			errChan <- err
			return
		}
		connChan <- conn
	}()

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("transfer cancelled")
	case err := <-errChan:
		return nil, fmt.Errorf("failed to accept connection: %v", err)
	case conn := <-connChan:
		return cm.NewConnection(conn), nil
	}
}

func sendFile(path string, conn net.Conn, progressChan chan<- SendProgress, ctx context.Context) {
	defer conn.Close()

	_, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	file, err := os.Open(path)
	if err != nil {
		progressChan <- SendProgress{
			State: StateError,
			Error: fmt.Errorf("failed to access file '%s': %w", path, err),
		}
		return
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		progressChan <- SendProgress{
			State: StateError,
			Error: fmt.Errorf("failed to get file info for '%s': %v", path, err),
		}
		return
	}

	conn.SetDeadline(time.Now().Add(30 * time.Second))
	defer conn.SetDeadline(time.Time{})

	defer func() {
		if r := recover(); r != nil {
			progressChan <- SendProgress{
				State: StateError,
				Error: fmt.Errorf("unexpected error: %v", r),
			}
		}
	}()

	reader := bufio.NewReader(conn)
	response, err := reader.ReadString('\n')
	if err != nil {
		progressChan <- SendProgress{
			State: StateError,
			Error: fmt.Errorf("error receiving ready signal: %v", err),
		}
		return
	}

	if strings.TrimSpace(response) != "ready" {
		progressChan <- SendProgress{
			State: StateError,
			Error: fmt.Errorf("unexpected response from receiver: %s", response),
		}
		return
	}

	metadata := fmt.Sprintf("%s|%d\n", filepath.Base(path), fileInfo.Size())
	_, err = conn.Write([]byte(metadata))
	if err != nil {
		progressChan <- SendProgress{
			State: StateError,
			Error: fmt.Errorf("failed to send metadata: %v", err),
		}
		return
	}

	response, err = reader.ReadString('\n')
	if err != nil {
		progressChan <- SendProgress{
			State: StateError,
			Error: fmt.Errorf("error receiving response: %v", err),
		}
		return
	}

	if strings.TrimSpace(response) != "accepted" {
		progressChan <- SendProgress{
			State: StateCancelled,
			Error: ErrTransferRejected,
		}
		return
	}

	progressChan <- SendProgress{
		State:      StateTransferring,
		TotalBytes: fileInfo.Size(),
	}

	buffer := make([]byte, CHUNK_SIZE)
	var sentBytes int64
	startTime := time.Now()

	for {
		select {
		case <-ctx.Done():
			progressChan <- SendProgress{
				State: StateCancelled,
				Error: fmt.Errorf("transfer cancelled"),
			}
			return
		default:
		}

		n, err := file.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			progressChan <- SendProgress{
				State: StateError,
				Error: fmt.Errorf("error reading file: %v", err),
			}
			return
		}

		_, err = conn.Write(buffer[:n])
		if err != nil {
			progressChan <- SendProgress{
				State: StateError,
				Error: fmt.Errorf("error sending file data: %v", err),
			}
			return
		}

		sentBytes += int64(n)
		speed := float64(sentBytes) / time.Since(startTime).Seconds() / 1024 / 1024

		progressChan <- SendProgress{
			State:      StateTransferring,
			Speed:      speed,
			BytesSent:  sentBytes,
			TotalBytes: fileInfo.Size(),
		}
	}

	progressChan <- SendProgress{
		State:     StateCompleted,
		BytesSent: sentBytes,
		Speed:     float64(sentBytes) / time.Since(startTime).Seconds() / 1024 / 1024,
	}
}
