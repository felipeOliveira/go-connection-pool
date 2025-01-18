package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"math/rand"

	"github.com/foliveira/go-connection-pool/sql"
)

func main() {

	pool := sql.New(sql.PoolConfig{
		MinSize:       5,
		MaxSize:       10,
		IdleTimeout:   20 * time.Second,
		ConnectTimout: 5 * time.Second,
	})

	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go query(pool, &wg)
	}

	wg.Wait()

	for {
		time.Sleep(5 * time.Second)

		idleConns := pool.IdleConnections()
		fmt.Println("idle connection: ", idleConns)

		if idleConns == 0 {
			break
		}
	}

}

func query(pool *sql.ConnectionPool, wg *sync.WaitGroup) {

	conn, err := pool.GetConnection(context.TODO())
	defer pool.Release(conn)

	defer wg.Done()

	if err != nil {
		log.Fatal(err)
	}
	time.Sleep(time.Duration(rand.Intn(10)) * time.Second)
	fmt.Printf("active conn: %d idle conn: %d  pending conn:%d average execution time: %s\n", pool.ActiveConnections(), pool.IdleConnections(), pool.PendingConnections(), pool.ExecutionAverage())

}
