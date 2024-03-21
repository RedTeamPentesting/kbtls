package kbtls

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"net"
	"testing"
)

func TestGenerateAndParseConnectionKey(t *testing.T) {
	key, err := GenerateConnectionKey()
	if err != nil {
		t.Fatalf("generate connection key: %v", err)
	}

	parsed, err := ParseConnectionKey(key.String())
	if err != nil {
		t.Fatalf("parse connection key: %v", err)
	}

	if !bytes.Equal(key[:], parsed[:]) {
		t.Fatalf("parsed key is not equal to original key: %v", err)
	}
}

func TestSameKeySameCA(t *testing.T) {
	key, err := GenerateConnectionKey()
	if err != nil {
		t.Fatalf("generate connection key: %v", err)
	}

	ca1, _, err := GenerateCA(key)
	if err != nil {
		t.Fatalf("generate CA 1: %v", err)
	}

	ca2, _, err := GenerateCA(key)
	if err != nil {
		t.Fatalf("generate CA 2: %v", err)
	}

	if !ca1.Equal(ca2) {
		t.Fatalf("GenerateCA with the same key produced two diffent certificates")
	}
}

func TestCASerialIsPublicKey(t *testing.T) {
	key, err := GenerateConnectionKey()
	if err != nil {
		t.Fatalf("generate connection key: %v", err)
	}

	ca, _, err := GenerateCA(key)
	if err != nil {
		t.Fatalf("generate CA 1: %v", err)
	}

	if base64.RawStdEncoding.EncodeToString(ca.SerialNumber.Bytes()) != key.PublicKey() {
		t.Fatalf("CA serial did not correspond to public key")
	}
}

func TestDontParseZeroKey(t *testing.T) {
	var zeroKey ConnectionKey

	_, err := ParseConnectionKey(zeroKey.String())
	if err == nil {
		t.Fatalf("parsing all-zero key did not produce an error")
	}
}

func TestDontCreateCAWithZeroKey(t *testing.T) {
	var zeroKey ConnectionKey

	_, _, err := GenerateCA(zeroKey)
	if err == nil {
		t.Fatalf("generating CA with all-zero key did not produce an error")
	}
}

func TestDontCreateTLSClientConfigWithZeroKey(t *testing.T) {
	var zeroKey ConnectionKey

	_, err := ClientTLSConfig(zeroKey)
	if err == nil {
		t.Fatalf("generating TLS client certificate with all-zero key did not produce an error")
	}
}

func TestDontCreateTLSServerConfigWithZeroKey(t *testing.T) {
	var zeroKey ConnectionKey

	_, err := ServerTLSConfig(zeroKey)
	if err == nil {
		t.Fatalf("generating TLS server certificate with all-zero key did not produce an error")
	}
}

func TestClientServerConfigCompatibility(t *testing.T) {
	key, err := GenerateConnectionKey()
	if err != nil {
		t.Fatalf("generate connection key: %v", err)
	}

	clientConfig, err := ClientTLSConfig(key)
	if err != nil {
		t.Fatalf("generate client config: %v", err)
	}

	serverConfig, err := ServerTLSConfig(key)
	if err != nil {
		t.Fatalf("generate client config: %v", err)
	}

	wait := make(chan struct{})

	pipeA, pipeB := net.Pipe()
	defer pipeA.Close() //nolint:errcheck
	defer pipeB.Close() //nolint:errcheck
	defer func() {
		<-wait
	}()

	testData := []byte("test")

	go func() {
		server := tls.Server(pipeA, serverConfig)
		defer server.Close() //nolint:errcheck

		_, err := server.Write(testData)
		if err != nil {
			t.Errorf("write: %v", err)
		}

		close(wait)
	}()

	conn := tls.Client(pipeB, clientConfig)

	receivedData := make([]byte, len(testData))

	n, err := conn.Read(receivedData)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	if !bytes.Equal(receivedData[:n], testData) {
		t.Fatalf("received %q instead of %q", string(receivedData), string(testData))
	}
}

func TestClientServerErrorIfKeyAndNameDiffers(t *testing.T) {
	clientKey, err := GenerateConnectionKey()
	if err != nil {
		t.Fatalf("generate client connection key: %v", err)
	}

	clientConfig, err := ClientTLSConfig(clientKey)
	if err != nil {
		t.Fatalf("generate client config: %v", err)
	}

	serverKey, err := GenerateConnectionKey()
	if err != nil {
		t.Fatalf("generate server connection key: %v", err)
	}

	if bytes.Equal(clientKey[:], serverKey[:]) {
		t.Fatalf("same key was generated twice")
	}

	serverConfig, err := ServerTLSConfig(serverKey)
	if err != nil {
		t.Fatalf("generate client config: %v", err)
	}

	wait := make(chan struct{})

	pipeA, pipeB := net.Pipe()
	defer pipeA.Close() //nolint:errcheck
	defer pipeB.Close() //nolint:errcheck
	defer func() {
		<-wait
	}()

	testData := []byte("test")

	go func() {
		_, err := tls.Server(pipeA, serverConfig).Write(testData)
		if err == nil {
			t.Errorf("sending data to client did not return an error")
		}

		close(wait)
	}()

	conn := tls.Client(pipeB, clientConfig)
	defer conn.Close() //nolint:errcheck

	readData := make([]byte, len(testData))

	_, err = conn.Read(readData)
	if err == nil {
		t.Fatalf("reading from connection did not return an error")
	}
}

func TestClientServerErrorIfKeyDiffers(t *testing.T) {
	clientKey, err := GenerateConnectionKey()
	if err != nil {
		t.Fatalf("generate client connection key: %v", err)
	}

	clientConfig, err := ClientTLSConfig(clientKey)
	if err != nil {
		t.Fatalf("generate client config: %v", err)
	}

	clientConfig.ServerName = "test"

	serverKey, err := GenerateConnectionKey()
	if err != nil {
		t.Fatalf("generate server connection key: %v", err)
	}

	if bytes.Equal(clientKey[:], serverKey[:]) {
		t.Fatalf("same key was generated twice")
	}

	serverConfig, err := ServerTLSConfigForServerName(serverKey, clientConfig.ServerName)
	if err != nil {
		t.Fatalf("generate client config: %v", err)
	}

	wait := make(chan struct{})

	pipeA, pipeB := net.Pipe()
	defer pipeA.Close() //nolint:errcheck
	defer pipeB.Close() //nolint:errcheck
	defer func() {
		<-wait
	}()

	testData := []byte("test")

	go func() {
		_, err := tls.Server(pipeA, serverConfig).Write(testData)
		if err == nil {
			t.Errorf("sending data to client did not return an error")
		}

		close(wait)
	}()

	conn := tls.Client(pipeB, clientConfig)

	readData := make([]byte, len(testData))

	_, err = conn.Read(readData)
	if err == nil {
		t.Fatalf("reading from connection did not return an error")
	}
}

func TestConnectionKeyValid(t *testing.T) {
	key, err := GenerateConnectionKey()
	if err != nil {
		t.Fatalf("genrate connection key: %v", err)
	}

	if !key.Valid() {
		t.Fatalf("generated connection key is not valid")
	}
}

func TestConnectionKeyInvalid(t *testing.T) {
	var key ConnectionKey

	if key.Valid() {
		t.Fatalf("all-zero connections key reports that it is valid")
	}
}

func TestDialListen(t *testing.T) {
	key, err := GenerateConnectionKey()
	if err != nil {
		t.Fatalf("generate connection key: %v", err)
	}

	listener, err := Listen("tcp", "localhost:0", key.String())
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	testData := []byte("test")

	wait := make(chan struct{})
	defer func() { <-wait }()

	go func() {
		defer close(wait)

		conn, err := Dial("tcp", listener.Addr().String(), key.String())
		if err != nil {
			t.Errorf("dial: %v", err)

			return
		}

		defer conn.Close() //nolint:errcheck

		_, err = conn.Write(testData)
		if err != nil {
			t.Errorf("write: %v", err)
		}
	}()

	conn, err := listener.Accept()
	if err != nil {
		t.Fatalf("accept: %v", err)
	}

	defer conn.Close() //nolint:errcheck

	receivedData := make([]byte, len(testData))

	n, err := conn.Read(receivedData)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	if !bytes.Equal(receivedData[:n], testData) {
		t.Fatalf("received %q instead of %q", string(receivedData), string(testData))
	}
}
