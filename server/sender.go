package server

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
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

		progressChan <- SendProgress{State: StateInitializing}

		select {
		case <-ctx.Done():
			progressChan <- SendProgress{
				State: StateCancelled,
				Error: fmt.Errorf("transfer cancelled"),
			}
			return
		default:
		}

		resp, err := http.Post(RELAY_PROTOCOL+"://"+RELAY_SERVER+"/new", "application/json", nil)
		if err != nil {
			progressChan <- SendProgress{
				State: StateError,
				Error: fmt.Errorf("failed to create session: %v", err),
			}
			return
		}

		var session TransferSession
		if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
			progressChan <- SendProgress{
				State: StateError,
				Error: fmt.Errorf("failed to decode session: %v", err),
			}
			return
		}
		resp.Body.Close()

		progressChan <- SendProgress{
			State:     StateWaitingForReceiver,
			SessionID: session.SessionID,
		}

		listener, err := net.Listen("tcp", ":"+TRANSFER_PORT)
		if err != nil {
			progressChan <- SendProgress{
				State: StateError,
				Error: fmt.Errorf("failed to start listener: %v", err),
			}
			return
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
			listener.Close()
			progressChan <- SendProgress{
				State: StateCancelled,
				Error: fmt.Errorf("transfer cancelled"),
			}
			return
		case err := <-errChan:
			progressChan <- SendProgress{
				State: StateError,
				Error: fmt.Errorf("failed to accept connection: %v", err),
			}
			return
		case conn := <-connChan:
			defer conn.Close()
			sendFile(filepath, conn, progressChan, ctx)
		}
	}()
}

func sendFile(path string, conn net.Conn, progressChan chan<- SendProgress, ctx context.Context) {
	file, err := os.Open(path)
	if err != nil {
		progressChan <- SendProgress{
			State: StateError,
			Error: fmt.Errorf("failed to access file: %w", err),
		}
		return
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		progressChan <- SendProgress{
			State: StateError,
			Error: fmt.Errorf("failed to get file info: %v", err),
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
			progressChan <- SendProgress{
				State:     StateCompleted,
				BytesSent: sentBytes,
				Speed:     float64(sentBytes) / time.Since(startTime).Seconds() / 1024 / 1024,
			}
			return
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
				Error: fmt.Errorf("error sending file: %v", err),
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
}
