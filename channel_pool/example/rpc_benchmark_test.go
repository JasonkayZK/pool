package example

import (
	"fmt"
	"net/rpc"
	"sync"
	"testing"
	"time"

	. "github.com/jasonkayzk/pool/channel_pool/pool"
)

const (
	benchmarkTime = 5000
)

func TestRpcBenchmark(t *testing.T) {
	p, _ := newChannelPool()
	defer p.ShutDown()

	wg := sync.WaitGroup{}
	// simpleMethod
	wg.Add(1)
	go func() {
		defer wg.Done()
		now := time.Now()
		for i := 0; i < benchmarkTime; i++ {
			simpleMethod(&Args{A: i, B: 1})
		}
		fmt.Println("simpleMethod elapsed: ", time.Since(now))
	}()
	// poolMethod
	wg.Add(1)
	go func() {
		defer wg.Done()
		now := time.Now()
		for i := 0; i < benchmarkTime; i++ {
			poolMethod(&p, &Args{A: i, B: 1})
		}
		fmt.Println("poolMethod elapsed: ", time.Since(now))
	}()

	wg.Wait()
}

func poolMethod(p *Pool, args *Args) {
	conn, _ := (*p).Get()
	cli := conn.(*rpc.Client)
	defer (*p).Put(conn)

	var resp int
	err := cli.Call("Number.Multiply", *args, &resp)
	if err != nil {
		fmt.Println(err)
	}
}

func simpleMethod(args *Args) {
	client, _ := rpc.DialHTTP(
		"tcp",
		address,
	)

	var resp int
	err := client.Call("Number.Multiply", *args, &resp)
	if err != nil {
		fmt.Println(err)
	}
}