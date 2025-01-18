package sql

import (
	"context"
	"fmt"
	"sync"

	"time"

	"golang.org/x/sync/semaphore"
)

type Connection struct {
	Id       int
	lastUsed time.Time
}

type PoolConfig struct {
	MinSize       int
	MaxSize       int
	IdleTimeout   time.Duration
	ConnectTimout time.Duration
}

type ConnectionPool struct {
	conf               PoolConfig
	sem                *semaphore.Weighted
	idleConns          chan *Connection
	activeConns        int
	execCount          int
	pendingConnections int
	execDuration       time.Duration
	mu                 sync.Mutex
}

func New(config PoolConfig) *ConnectionPool {

	pool := &ConnectionPool{
		conf:      config,
		sem:       semaphore.NewWeighted(int64(config.MaxSize)),
		idleConns: make(chan *Connection, config.MinSize),
	}

	go pool.trackIdleConnections()

	return pool
}

func (p *ConnectionPool) GetConnection(ctx context.Context) (*Connection, error) {

	p.mu.Lock()
	p.pendingConnections++
	p.mu.Unlock()

	if ctx.Err() != nil {
		return nil, fmt.Errorf("connectionpool: could not get connection: %w", ctx.Err())
	}

	c, cancelFn := context.WithTimeout(ctx, time.Duration(p.conf.ConnectTimout))
	defer cancelFn()

	select {
	case <-c.Done():
		return nil, fmt.Errorf("connectionpool: could not get connection: %w", c.Err())
	default:

		err := p.sem.Acquire(ctx, 1)

		p.mu.Lock()
		p.pendingConnections--
		p.mu.Unlock()

		if err != nil {

			return nil, fmt.Errorf("connectionpool: could not acquire new connection: %w", err)
		}

		select {

		case conn := <-p.idleConns:
			p.mu.Lock()
			p.activeConns++
			conn.lastUsed = time.Now()
			p.mu.Unlock()

			return conn, nil
		default:
		}

		if p.activeConns >= p.conf.MaxSize {
			select {
			case <-ctx.Done():
				p.sem.Release(1)
				return nil, ctx.Err()
			case conn := <-p.idleConns:
				p.mu.Lock()
				p.activeConns++
				conn.lastUsed = time.Now()
				p.mu.Unlock()
				return conn, nil
			}
		}
		p.mu.Lock()
		conn := &Connection{
			Id:       p.activeConns + 1,
			lastUsed: time.Now(),
		}
		p.activeConns++
		p.mu.Unlock()

		return conn, nil
	}

}

func (p *ConnectionPool) Release(conn *Connection) error {

	defer p.sem.Release(1)

	p.mu.Lock()
	p.activeConns--
	p.execCount++
	p.execDuration += time.Since(conn.lastUsed)
	p.mu.Unlock()

	select {
	case p.idleConns <- conn:
		return nil
	default:
		return conn.Close()
	}
}

func (p *ConnectionPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	close(p.idleConns)

	for conn := range p.idleConns {
		if err := conn.Close(); err != nil {
			return err
		}
	}

	return nil
}

func (p *ConnectionPool) ActiveConnections() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.activeConns
}

func (p *ConnectionPool) IdleConnections() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.idleConns)
}

func (p *ConnectionPool) ExecutionAverage() time.Duration {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.execCount == 0 {
		return 0
	}
	return p.execDuration / time.Duration(p.execCount)
}

func (p *ConnectionPool) PendingConnections() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.pendingConnections
}
func (p *ConnectionPool) trackIdleConnections() {
	for {
		time.Sleep(p.conf.IdleTimeout / 2)

		p.mu.Lock()
		for i := 0; i < len(p.idleConns); i++ {
			conn := <-p.idleConns
			if time.Since(conn.lastUsed) > p.conf.IdleTimeout {
				conn.Close()
				continue
			}

			p.idleConns <- conn
		}
		p.mu.Unlock()
	}
}

func (c *Connection) Close() error {
	return nil
}
