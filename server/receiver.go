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
	"strconv"
	"strings"
	"time"
)

type ReceiveProgress struct {
	Speed         float64
	BytesReceived int64
	Error         error
	State         TransferState
}

func LeaveSession(sessionID string) error {
	if sessionID == "" {
		return nil
	}

	resp, err := http.Get(RELAY_PROTOCOL + "://" + RELAY_SERVER + "/leave/" + sessionID)
	if err != nil {
		return fmt.Errorf("couldn't disconnect cleanly (%v)", err.Error())
	}
	defer resp.Body.Close()
	return nil
}

func StartReceiver(sessionID string) (net.Conn, error) {
	if sessionID == "" {
		return nil, SessionError{
			Code:    "INVALID_SESSION",
			Message: "Please enter a valid session ID",
		}
	}

	resp, err := http.Get(RELAY_PROTOCOL + "://" + RELAY_SERVER + "/join/" + sessionID)
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
			Message: fmt.Sprintf("Unexpected error (status %d) - please try again", resp.StatusCode),
		}
	}

	var session TransferSession
	if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
		return nil, fmt.Errorf("invalid session data - please try again")
	}

	conn, err := net.DialTimeout("tcp", "localhost:"+TRANSFER_PORT, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("couldn't connect to sender - are they still online? (%v)", err.Error())
	}

	conn.SetDeadline(time.Now().Add(10 * time.Second))
	return conn, nil
}

func ReceiveMetadata(conn net.Conn) (FileMetadata, error) {
	conn.SetDeadline(time.Now().Add(10 * time.Second))
	reader := bufio.NewReader(conn)

	_, err := conn.Write([]byte("ready\n"))
	if err != nil {
		return FileMetadata{}, fmt.Errorf("failed to send ready signal: %v", err)
	}

	fileInfo, err := reader.ReadString('\n')
	if err != nil {
		return FileMetadata{}, fmt.Errorf("failed to read file info: %v", err)
	}

	parts := strings.Split(strings.TrimSpace(fileInfo), "|")
	if len(parts) != 2 {
		return FileMetadata{}, fmt.Errorf("invalid file info: %s", fileInfo)
	}

	filesize, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return FileMetadata{}, fmt.Errorf("failed to parse file size: %v %s", err, parts[1])
	}
	metadata := FileMetadata{
		Name:     parts[0],
		Size:     filesize,
		SenderIP: conn.RemoteAddr().String(),
	}

	return metadata, nil
}

func ReceiveFile(conn net.Conn, m FileMetadata, progressChan chan<- ReceiveProgress, ctx context.Context) {
	go func() {
		defer close(progressChan)
		defer conn.Close()

		defer func() {
			if r := recover(); r != nil {
				progressChan <- ReceiveProgress{
					Error: fmt.Errorf("unexpected error: %v", r),
					State: StateError,
				}
			}
		}()

		conn.SetDeadline(time.Now().Add(30 * time.Second))
		defer conn.SetDeadline(time.Time{})

		progressChan <- ReceiveProgress{State: StateInitializing}

		reader := bufio.NewReader(conn)
		conn.Write([]byte("accepted\n"))

		safeName := m.Name

		if _, err := os.Stat(safeName); err == nil {
			safeName = fmt.Sprintf("%s_%d%s",
				strings.TrimSuffix(m.Name, filepath.Ext(m.Name)),
				time.Now().Unix(),
				filepath.Ext(m.Name),
			)
		}

		file, err := os.Create(safeName)
		if err != nil {
			progressChan <- ReceiveProgress{
				Error: fmt.Errorf("failed to create file '%s': %v", safeName, err),
				State: StateError,
			}
			return
		}
		defer file.Close()

		buffer := make([]byte, CHUNK_SIZE)
		var receivedBytes int64
		startTime := time.Now()

		progressChan <- ReceiveProgress{State: StateReceiving}

		for {
			select {
			case <-ctx.Done():
				file.Close()
				os.Remove(safeName)
				progressChan <- ReceiveProgress{
					Speed:         float64(receivedBytes) / time.Since(startTime).Seconds() / 1024 / 1024,
					BytesReceived: receivedBytes,
					State:         StateCancelled,
					Error:         fmt.Errorf("transfer cancelled"),
				}
				return
			default:
			}

			n, err := reader.Read(buffer)
			if err == io.EOF {
				break
			}
			if err != nil {
				progressChan <- ReceiveProgress{
					Error: fmt.Errorf("failed to read from connection: %v", err),
					State: StateError,
				}
				return
			}

			_, err = file.Write(buffer[:n])
			if err != nil {
				progressChan <- ReceiveProgress{
					Error: fmt.Errorf("failed to write to file '%s': %v", safeName, err),
					State: StateError,
				}
				return
			}

			receivedBytes += int64(n)
			speed := float64(receivedBytes) / time.Since(startTime).Seconds() / 1024 / 1024

			progressChan <- ReceiveProgress{
				Speed:         speed,
				BytesReceived: receivedBytes,
				State:         StateReceiving,
			}
		}

		progressChan <- ReceiveProgress{
			Speed:         float64(receivedBytes) / time.Since(startTime).Seconds() / 1024 / 1024,
			BytesReceived: receivedBytes,
			State:         StateCompleted,
		}
	}()
}
