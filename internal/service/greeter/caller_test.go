// Copyright (c) 2026 H0llyW00dzZ All rights reserved.
//
// By accessing or using this software, you agree to be bound by the terms
// of the License Agreement, which you can find at LICENSE files.

package greeter_test

import (
	"context"
	"io"
	"testing"

	"github.com/H0llyW00dzZ/grpc-template/internal/logging"
	"github.com/H0llyW00dzZ/grpc-template/internal/service/greeter"
	"github.com/H0llyW00dzZ/grpc-template/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

func newGreeterConn(t *testing.T, lis *bufconn.Listener) (*grpc.ClientConn, error) {
	t.Helper()
	conn, err := testutil.DialBufNet(context.Background(), lis)
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() { conn.Close() })
	return conn, nil
}

func TestCallerSayHello(t *testing.T) {
	lis := startGreeterServer(t)
	conn, err := newGreeterConn(t, lis)
	require.NoError(t, err)

	caller := greeter.NewCaller(conn, logging.Default())

	resp, err := caller.SayHello(context.Background(), "World")
	require.NoError(t, err)
	assert.Equal(t, "Hello, World!", resp.GetMessage())
}

func TestCallerSayHello_EmptyName(t *testing.T) {
	lis := startGreeterServer(t)
	conn, err := newGreeterConn(t, lis)
	require.NoError(t, err)

	caller := greeter.NewCaller(conn, logging.Default())

	resp, err := caller.SayHello(context.Background(), "")
	require.NoError(t, err)
	assert.Equal(t, "Hello, !", resp.GetMessage())
}

func TestCallerSayHelloServerStream(t *testing.T) {
	lis := startGreeterServer(t)
	conn, err := newGreeterConn(t, lis)
	require.NoError(t, err)

	caller := greeter.NewCaller(conn, logging.Default())

	stream, err := caller.SayHelloServerStream(context.Background(), "Alice")
	require.NoError(t, err)

	want := []string{
		"Hello, Alice!",
		"How are you, Alice?",
		"Good to see you, Alice!",
	}

	var got []string
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		got = append(got, resp.GetMessage())
	}

	require.Len(t, got, len(want))
	assert.Equal(t, want, got)
}
