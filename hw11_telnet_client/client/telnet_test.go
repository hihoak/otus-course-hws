package client

import (
	"bytes"
	"io"
	"io/ioutil"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTelnetClient(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		l, err := net.Listen("tcp", "127.0.0.1:")
		require.NoError(t, err)
		defer func() { require.NoError(t, l.Close()) }()

		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()

			in := &bytes.Buffer{}
			out := &bytes.Buffer{}

			timeout, err := time.ParseDuration("10s")
			require.NoError(t, err)

			client := NewTelnetClient(l.Addr().String(), timeout, ioutil.NopCloser(in), out)
			require.NoError(t, client.Connect())
			defer func() { require.NoError(t, client.Close()) }()

			in.WriteString("hello\n")
			err = client.Send()
			require.NoError(t, err)

			err = client.Receive()
			require.NoError(t, err)
			require.Equal(t, "world\n", out.String())
		}()

		go func() {
			defer wg.Done()

			conn, err := l.Accept()
			require.NoError(t, err)
			require.NotNil(t, conn)
			defer func() { require.NoError(t, conn.Close()) }()

			request := make([]byte, 1024)
			n, err := conn.Read(request)
			require.NoError(t, err)
			require.Equal(t, "hello\n", string(request)[:n])

			n, err = conn.Write([]byte("world\n"))
			require.NoError(t, err)
			require.NotEqual(t, 0, n)
		}()

		wg.Wait()
	})
}

func TestAdditionalTelnetClient(t *testing.T) {
	t.Run("failed to connect to address", func(t *testing.T) {
		in := &bytes.Buffer{}
		out := &bytes.Buffer{}

		timeout, err := time.ParseDuration("1s")
		require.NoError(t, err)

		client := NewTelnetClient("ithinkitisnotexistforsure1322133.my.site.tests", timeout, io.NopCloser(in), out)
		err = client.Connect()
		require.ErrorIs(t, err, ErrorEstablishConnection{})
	})

	t.Run("close", func(t *testing.T) {
		l, err := net.Listen("tcp", "127.0.0.1:")
		require.NoError(t, err)
		defer func() { require.NoError(t, l.Close()) }()

		in := &bytes.Buffer{}
		out := &bytes.Buffer{}

		timeout, err := time.ParseDuration("10s")
		require.NoError(t, err)

		client := NewTelnetClient(l.Addr().String(), timeout, ioutil.NopCloser(in), out)
		require.NoError(t, client.Connect())
		receiverErrors := make(chan error)
		go func() {
			defer close(receiverErrors)
			receiverErrors <- client.Receive()
		}()
		senderErrors := make(chan error)
		go func() {
			defer close(senderErrors)
			senderErrors <- client.Send()
		}()
		require.NoError(t, client.Close())
		err, ok := <-receiverErrors
		require.NoError(t, err)
		require.True(t, ok)
		err, ok = <-senderErrors
		require.NoError(t, err)
		require.True(t, ok)
	})
}
