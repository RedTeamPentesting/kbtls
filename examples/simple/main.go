// Package main implements a simple kbtls client and server.
package main

import (
	"fmt"
	"os"

	"github.com/RedTeamPentesting/kbtls"
)

var message = []byte("hello")

func client() error {
	conn, err := kbtls.Dial("tcp", "localhost:8443", os.Args[2])
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}

	buf := make([]byte, len(message))

	_, err = conn.Read(buf)
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}

	fmt.Printf("TLS connection with %s was established successfully, received message: %s\n",
		conn.RemoteAddr(), string(buf))

	err = conn.Close()
	if err != nil {
		return fmt.Errorf("close: %w", err)
	}

	return nil
}

func server() error {
	var key string

	if len(os.Args) > 2 {
		key = os.Args[2]
	} else {
		k, err := kbtls.GenerateConnectionKey()
		if err != nil {
			return fmt.Errorf("generate connection key: %w", err)
		}

		key = k.String()
	}

	fmt.Println("Listening on localhost:8443")

	listener, err := kbtls.Listen("tcp", "localhost:8443", key)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	defer listener.Close() //nolint:errcheck,gosec

	for {
		conn, err := listener.Accept()
		if err != nil {
			return fmt.Errorf("accept: %w", err)
		}

		fmt.Printf("TLS connection with %s established successfully, sending message: %s\n",
			conn.RemoteAddr(), string(message))

		_, err = conn.Write(message)
		if err != nil {
			return fmt.Errorf("write: %w", err)
		}

		err = conn.Close()
		if err != nil {
			return fmt.Errorf("close conn: %w", err)
		}
	}
}

func run() error {
	switch {
	case len(os.Args) > 2 && os.Args[1] == "client":
		return client()
	case len(os.Args) > 1 && os.Args[1] == "server":
		return server()
	default:
		executable, err := os.Executable()
		if err != nil {
			executable = "example"
		}

		return fmt.Errorf("usage: %s (client|server) <connection key>", executable)
	}
}

func main() {
	err := run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)

		os.Exit(1)
	}
}
