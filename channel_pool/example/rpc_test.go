package example

import (
	"fmt"
	"github.com/jasonkayzk/pool/channel_pool"
	"math/rand"
	"net"
	"net/http"
	"net/rpc"
	"sync"
	"testing"
	"time"
)

var (
	InitialCap  = 5
	MaxIdleCap  = 10
	MaximumCap  = 30
	WaitTimeout = time.Second * 3
	IdleTimeout = time.Second * 20
	address     = "127.0.0.1:7777"
	factory     = func() (interface{}, error) {
		return rpc.DialHTTP(
			"tcp",
			address,
		)
	}
	closeFunc = func(v interface{}) error {
		nc := v.(*rpc.Client)
		return nc.Close()
	}
	pingFunc = func(v interface{}) error {
		cli := v.(*rpc.Client)
		var resp int
		err := cli.Call("Number.Multiply", Args{1, 2}, &resp)
		if err != nil {
			return err
		}

		if resp != 2 {
			return fmt.Errorf("rpc.err")
		}

		return nil
	}
)

func init() {
	// used for factory function
	go rpcServer()
	// wait until tcp server has been settled
	time.Sleep(time.Millisecond * 300)

	rand.Seed(time.Now().UTC().UnixNano())
}

func TestNew(t *testing.T) {
	p, err := newChannelPool()
	if err != nil {
		t.Errorf("New error: %s", err)
	}
	p.ShutDown()
}

func TestPool_Get_Impl(t *testing.T) {
	p, _ := newChannelPool()
	defer p.ShutDown()

	conn, err := p.Get()
	if err != nil {
		t.Errorf("Get error: %s", err)
	}

	_, ok := conn.(*rpc.Client)
	if !ok {
		t.Errorf("Conn is not of type poolConn")
	}

	if err = p.Put(conn); err != nil {
		t.Errorf("put conn err")
	}
}

func TestPool_Get(t *testing.T) {
	p, err := newChannelPool()
	if err != nil {
		t.Errorf("create pool error: %s", err)
	}
	defer p.ShutDown()

	_, err = p.Get()
	if err != nil {
		t.Errorf("Get error: %s", err)
	}

	// after one get, current capacity should be lowered by one.
	if p.Len() != (InitialCap - 1) {
		t.Errorf("Get error. Expecting %d, got %d", InitialCap-1, p.Len())
	}

	// get them all
	for i := 0; i < (MaximumCap - 1); i++ {
		go func() {
			_, err := p.Get()
			if err != nil {
				t.Errorf("Get error: %s", err)
			}
		}()
	}
	// wait for getting all connection
	time.Sleep(time.Second)

	if p.Len() != 0 {
		t.Errorf("Get error. Expecting %d, got %d", 0, p.Len())
	}

	_, err = p.Get()
	if !channel_pool.IsWaitConnectionTimeoutErr(err) {
		t.Errorf("Get error: %s", err)
	}
}

func TestPool_Put(t *testing.T) {
	p, _ := newChannelPool()
	defer p.ShutDown()

	// get/create from the pool
	conns := make([]interface{}, MaximumCap)
	for i := 0; i < MaximumCap; i++ {
		conn, _ := p.Get()
		conns[i] = conn
	}

	// now put them all back
	for _, conn := range conns {
		_ = p.Put(conn)
	}

	if p.Len() != MaxIdleCap {
		t.Errorf("Put error len. Expecting %d, got %d", 1, p.Len())
	}

	// close pool
	p.ShutDown()
}

func TestPool_UsedCapacity(t *testing.T) {
	p, _ := newChannelPool()
	defer p.ShutDown()

	if p.Len() != InitialCap {
		t.Errorf("InitialCap error. Expecting %d, got %d", InitialCap, p.Len())
	}
}

func TestPool_Close(t *testing.T) {
	p, _ := newChannelPool()
	// now close it and test all cases we are expecting.
	p.ShutDown()

	_, err := p.Get()
	if err == nil {
		t.Errorf("Close error, get conn should return an error")
	}

	if p.Len() != 0 {
		t.Errorf("Close error used capacity. Expecting 0, got %d", p.Len())
	}
}

func TestPoolConcurrent(t *testing.T) {
	p, _ := newChannelPool()
	pipe := make(chan interface{}, 0)

	go func() {
		p.ShutDown()
	}()

	for i := 0; i < MaximumCap; i++ {
		go func() {
			conn, _ := p.Get()

			pipe <- conn
		}()
		go func() {
			conn := <-pipe
			if conn == nil {
				return
			}
			_ = p.Put(conn)
		}()
	}
}

func TestPoolWriteRead(t *testing.T) {
	p, _ := newChannelPool()
	defer p.ShutDown()

	conn, _ := p.Get()
	cli := conn.(*rpc.Client)

	var resp int
	err := cli.Call("Number.Multiply", Args{1, 2}, &resp)
	if err != nil {
		t.Error(err)
	}

	if resp != 2 {
		t.Error("rpc.err")
	}
}

func TestPoolConcurrent2(t *testing.T) {
	p, _ := newChannelPool()
	defer p.ShutDown()

	var wg sync.WaitGroup
	go func() {
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(i int) {
				conn, _ := p.Get()
				time.Sleep(time.Millisecond * time.Duration(rand.Intn(100)))
				_ = p.CloseConn(conn)
				wg.Done()
			}(i)
		}
	}()
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			conn, _ := p.Get()
			time.Sleep(time.Millisecond * time.Duration(rand.Intn(100)))
			_ = p.CloseConn(conn)
			wg.Done()
		}(i)
	}

	wg.Wait()
}

func newChannelPool() (channel_pool.Pool, error) {
	ops := channel_pool.Options{
		InitialCap:  InitialCap,
		MaxCap:      MaximumCap,
		MaxIdle:     MaxIdleCap,
		Factory:     factory,
		Ping:        pingFunc,
		Close:       closeFunc,
		IdleTimeout: IdleTimeout,
		WaitTimeout: WaitTimeout,
	}
	return channel_pool.NewChannelPool(&ops)
}

type Number int

type Args struct {
	A, B int
}

func rpcServer() {
	number := new(Number)
	_ = rpc.Register(number)
	rpc.HandleHTTP()

	l, e := net.Listen("tcp", address)
	if e != nil {
		panic(e)
	}
	go http.Serve(l, nil)
}

func (n *Number) Multiply(args *Args, reply *int) error {
	*reply = args.A * args.B
	return nil
}
