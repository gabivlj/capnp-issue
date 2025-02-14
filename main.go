package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"sync/atomic"
	"time"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/rpc"
	"capnproto.org/go/capnp/v3/server"

	"github.com/gabivlj/reproedgeworker/bytestream"
)

var unixSocket = flag.String("unix-socket", "./repro.sock", "")
var mode = flag.String("mode", "server", "'client' or 'server'")

func assert(err error) {
	if err != nil {
		panic(err)
	}
}

type capnpLoggerWrapper struct {
}

func (l *capnpLoggerWrapper) Debug(msg string, m ...any) {
	log.Printf("CAPNP DEBUG: "+msg, m...)
}

func (l *capnpLoggerWrapper) Info(msg string, m ...any) {
	log.Printf("CAPNP INFO: "+msg, m...)

}

func (l *capnpLoggerWrapper) Warn(msg string, m ...any) {
	log.Printf("CAPNP WARN: "+msg, m...)
}

func (l *capnpLoggerWrapper) Error(msg string, m ...any) {
	log.Printf("CAPNP ERROR: "+msg, m...)

}

type debugStream struct {
	net.Conn
}

func (d *debugStream) Read(b []byte) (int, error) {
	n, err := d.Conn.Read(b)
	return n, err
}

func main() {
	flag.Parse()
	os.RemoveAll(*unixSocket)
	ln, err := net.Listen("unix", *unixSocket)
	assert(err)

	if *mode != "server" {
		panic("-mode is 'client' or 'server'")
	}

	log.Println("Listening in", *unixSocket)
	for {
		conn, err := ln.Accept()
		assert(err)
		log.Println("New connection")

		server := bytestream.Service_ServerToClient(&service{})
		client := capnp.Client(server)
		opts := &rpc.Options{
			Logger:          &capnpLoggerWrapper{},
			BootstrapClient: client,
		}

		rpcConn := rpc.NewConn(rpc.NewStreamTransport(&debugStream{conn}), opts)
		select {
		case <-rpcConn.Done():
			log.Println("conn done, accepting another")
		}
	}
}

type service struct{}

func (s *service) Get(ctx context.Context, getParams bytestream.Service_get) (rerr error) {
	getParams.Go()
	fmt.Println("Get from service")
	res, err := getParams.AllocResults()
	if err != nil {
		return err
	}

	bServer := bytestream.ByteStreamReturner_NewServer(&byteStreamGetter{})
	bServer.HandleUnknownMethod = func(m capnp.Method) *server.Method {
		fmt.Println("Handle unknown method too rip")
		return nil
	}

	client := capnp.NewClient(bServer)
	b := bytestream.ByteStreamReturner(client)
	time.Sleep(time.Second * 3)

	return res.SetBsr(b)
}

type byteStreamGetter struct{}

func (c *byteStreamGetter) Inflighter(ctx context.Context, getParams bytestream.ByteStreamReturner_inflighter) error {
	getParams.Go()
	fmt.Println("Inflighter BS start")
	select {
	case <-ctx.Done():
		// 3s is enough for repro
	case <-time.After(time.Second * 3):
	}
	fmt.Println("Inflighter BS done")
	return nil
}

func (c *byteStreamGetter) Shutdown() {
	fmt.Println("byte stream getter Shutdown()")
}

func (c *byteStreamGetter) GetConnector(ctx context.Context, getParams bytestream.ByteStreamReturner_getConnector) (rerr error) {
	getParams.Go()
	fmt.Println("Get from byteStreamGetter")
	res, err := getParams.AllocResults()
	if err != nil {
		return err
	}

	bServer := bytestream.Connector_NewServer(&conn{})
	bServer.HandleUnknownMethod = func(m capnp.Method) *server.Method {
		fmt.Println("Unknown method:", m.String())
		// So here, if you for example pass the bytestream
		// we return in Connect and do something like this:

		/*
			    b := <-p.c
				fmt.Println("Got bytestream, returning impl")
				methods := bytestream.ByteStream_Methods([]server.Method{}, b)
				s := &server.Method{methods[m.MethodID].Method, methods[m.MethodID].Impl}
				p.c <- b
		*/

		// It will work!
		//
		// IDK why capnp-go here doesn't really understand what's going on
		return nil
	}

	// The more you sleep, the easier to repro it is. Not even sleeping
	// and you will repro this.
	time.Sleep(time.Millisecond)
	client := capnp.NewClient(bServer)
	b := bytestream.Connector(client)
	return res.SetConn(b)
}

type conn struct {
}

func (c *conn) Connect(ctx context.Context, params bytestream.Connector_connect) error {
	res, err := params.AllocResults()
	if err != nil {
		return err
	}

	bServer := bytestream.ByteStream_NewServer(&byteStreamWrite{})
	bServer.HandleUnknownMethod = func(m capnp.Method) *server.Method {
		fmt.Println("Unknown method, noooo!!!!!!!!!!")
		return nil
	}

	client := capnp.NewClient(bServer)
	b := bytestream.ByteStream(client)
	return res.SetUp(b)
}

type byteStreamWrite struct {
}

var a atomic.Int64

func (c *byteStreamWrite) Shutdown() {
	log.Println("ByteStream Shutdown()")
}

// Write writes the bytes to the connection.
func (c *byteStreamWrite) Write(ctx context.Context, params bytestream.ByteStream_write) error {
	a.Add(1)
	params.Go()
	fmt.Println("Received Write", a.Load())
	args := params.Args()
	bytes, err := args.Bytes()
	if err != nil {
		return err
	}

	fmt.Println("Write called (good!)", string(bytes))
	return nil
}
