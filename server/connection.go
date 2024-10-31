package server

import (
	"context"
	"net"
	"sync"
	"time"
)

type ConnectionManager struct {
	activeConns sync.Map
	mu          sync.RWMutex
}

func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{}
}

type Connection struct {
	net.Conn
	ctx    context.Context
	cancel context.CancelFunc
}

func (cm *ConnectionManager) NewConnection(conn net.Conn) *Connection {
	ctx, cancel := context.WithCancel(context.Background())
	c := &Connection{
		Conn:   conn,
		ctx:    ctx,
		cancel: cancel,
	}
	cm.activeConns.Store(conn.RemoteAddr().String(), c)
	return c
}

func (c *Connection) Close() error {
	c.cancel()
	return c.Conn.Close()
}

func (c *Connection) WaitForResponse(timeout time.Duration) (string, error) {
	resultCh := make(chan string, 1)
	errCh := make(chan error, 1)

	ctx, cancel := context.WithTimeout(c.ctx, timeout)
	defer cancel()

	go func() {
		buffer := make([]byte, 1024)
		n, err := c.Read(buffer)
		if err != nil {
			errCh <- err
			return
		}
		resultCh <- string(buffer[:n])
	}()

	select {
	case <-ctx.Done():
		return "", ErrConnectionTimeout
	case err := <-errCh:
		return "", err
	case result := <-resultCh:
		return result, nil
	}
}

func (c *Connection) SendWithContext(ctx context.Context, data []byte) error {
	done := make(chan error, 1)

	go func() {
		_, err := c.Write(data)
		done <- err
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		return err
	}
}
