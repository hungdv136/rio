package database

import (
	"bytes"
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hungdv136/rio"
	"github.com/hungdv136/rio/internal/config"
	"github.com/hungdv136/rio/internal/types"
	"github.com/stretchr/testify/require"
)

func TestStubDbStore_Create(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store, err := NewStubDBStore(ctx, config.NewDBConfig())
	require.NoError(t, err)

	data := types.Map{"id": uuid.NewString()}
	res := rio.NewResponse().WithBody(rio.MustToJSON(data)).WithCookie("NEW_SESSION_ID", uuid.NewString())
	stub := rio.NewStub().For("POST", rio.Contains("animal/create")).
		WithHeader("X-REQUEST-ID", rio.EqualTo(uuid.NewString())).
		WithCookie("SESSION_ID", rio.EqualTo(uuid.NewString())).
		WithRequestBody(rio.BodyJSONPath("name", rio.EqualTo(uuid.NewString()))).
		WillReturn(res)
	require.NoError(t, store.Create(ctx, stub))
	require.True(t, stub.Active)

	foundStub, err := store.Find(ctx, stub.ID)
	require.NoError(t, err)

	require.Equal(t, stub.Response.StatusCode, foundStub.Response.StatusCode)
	require.Equal(t, stub.Response.Header, foundStub.Response.Header)
	require.Equal(t, len(stub.Response.Cookies), len(foundStub.Response.Cookies))
	for i := range foundStub.Response.Cookies {
		require.Equal(t, stub.Response.Cookies[i].Name, foundStub.Response.Cookies[i].Name)
		require.Equal(t, stub.Response.Cookies[i].Value, foundStub.Response.Cookies[i].Value)
		require.WithinDuration(t, stub.Response.Cookies[i].ExpiredAt, foundStub.Response.Cookies[i].ExpiredAt, time.Millisecond)
	}

	bodyMap, err := types.CreateMapFromReader(bytes.NewReader(foundStub.Response.Body))
	require.NoError(t, err, string(foundStub.Response.Body))
	require.Equal(t, data.ForceJSON(), bodyMap.ForceJSON())
}

func TestStubDbStore_Delete(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store, err := NewStubDBStore(ctx, config.NewDBConfig())
	require.NoError(t, err)

	stub := rio.NewStub().For("POST", rio.Contains("animal/create"))
	require.NoError(t, store.Create(ctx, stub))

	err = store.Delete(ctx, stub.ID)
	require.NoError(t, err)

	foundStub, err := store.Find(ctx, stub.ID)
	require.NoError(t, err)
	require.False(t, foundStub.Active)
}

func TestStubDbStore_GetAll(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store, err := NewStubDBStore(ctx, config.NewDBConfig())
	require.NoError(t, err)

	t.Run("with_namespace", func(t *testing.T) {
		t.Parallel()

		namespace := uuid.NewString()
		activeStub := rio.NewStub().For("POST", rio.Contains("animal/create")).WithNamespace(namespace)
		inactiveStub := rio.NewStub().For("POST", rio.Contains("animal/create")).WithNamespace(namespace).WithInactive()
		require.NoError(t, store.Create(ctx, activeStub, inactiveStub))

		allStubs, err := store.GetAll(ctx, namespace)
		require.NoError(t, err)
		require.Len(t, allStubs, 1)
		require.Equal(t, activeStub.ID, allStubs[0].ID)
	})

	t.Run("without_namespace", func(t *testing.T) {
		t.Parallel()

		activeStub := rio.NewStub().For("POST", rio.Contains("animal/create"))
		inactiveStub := rio.NewStub().For("POST", rio.Contains("animal/create")).WithInactive()
		require.NoError(t, store.Create(ctx, activeStub, inactiveStub))

		allStubs, err := store.GetAll(ctx, "")
		require.NoError(t, err)
		require.NotEmpty(t, allStubs)

		for _, stub := range allStubs {
			require.True(t, stub.Active)
			require.Empty(t, stub.Namespace)
		}
	})
}

func TestStubDbStore_CreateIncomeRequest(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store, err := NewStubDBStore(ctx, config.NewDBConfig())
	require.NoError(t, err)

	body, err := os.ReadFile("stub_store_test.go")
	require.NoError(t, err)

	request := &rio.IncomingRequest{
		Namespace: uuid.New().String(),
		URL:       uuid.NewString(),
		Method:    "GET",
		Header: types.Map{
			"key": uuid.NewString(),
		},
		CURL:   uuid.NewString(),
		Body:   body,
		StubID: 1,
	}

	err = store.CreateIncomingRequest(ctx, request)
	require.NoError(t, err)

	option := &rio.IncomingQueryOption{
		Ids:       []int64{request.ID},
		Namespace: request.Namespace,
		Limit:     1,
	}
	foundRequests, err := store.GetIncomingRequests(ctx, option)
	require.NoError(t, err)
	require.Equal(t, request.ID, foundRequests[0].ID)
	require.Equal(t, request.Namespace, foundRequests[0].Namespace)
	require.Equal(t, request.Method, foundRequests[0].Method)
	require.Equal(t, request.Header, foundRequests[0].Header)
	require.Equal(t, request.CURL, foundRequests[0].CURL)
	require.Equal(t, request.StubID, foundRequests[0].StubID)
	require.Equal(t, body, foundRequests[0].Body)
}

func TestStubDbStore_GetProtos(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store, err := NewStubDBStore(ctx, config.NewDBConfig())
	require.NoError(t, err)

	proto := &rio.Proto{
		Name:    "proto",
		FileID:  uuid.NewString(),
		Methods: []string{uuid.NewString(), uuid.NewString()},
	}
	require.NoError(t, store.CreateProto(ctx, proto))

	createdProtos, err := store.GetProtos(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, createdProtos)

	for _, p := range createdProtos {
		if p.ID == proto.ID {
			require.Equal(t, proto.Name, p.Name)
			require.Equal(t, proto.FileID, p.FileID)
			require.Equal(t, proto.Methods, p.Methods)
		}
	}

	last, err := store.GetLastUpdatedProto(ctx)
	require.NoError(t, err)
	require.NotZero(t, last.UpdatedAt)
	require.NotZero(t, last.ID)
}

func TestStubDbStore_GetLastUpdatedStub(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store, err := NewStubDBStore(ctx, config.NewDBConfig())
	require.NoError(t, err)

	t.Run("with_namespace", func(t *testing.T) {
		t.Parallel()

		namespace := uuid.NewString()
		activeStub := rio.NewStub().For("POST", rio.Contains("animal/create")).WithNamespace(namespace)
		inactiveStub := rio.NewStub().For("POST", rio.Contains("animal/create")).WithNamespace(namespace).WithInactive()
		require.NoError(t, store.Create(ctx, activeStub, inactiveStub))

		last, err := store.GetLastUpdatedStub(ctx, namespace)
		require.NoError(t, err)
		require.Equal(t, inactiveStub.ID, last.ID)
		require.NotZero(t, last.UpdatedAt)
	})

	t.Run("without_namespace", func(t *testing.T) {
		t.Parallel()

		activeStub := rio.NewStub().For("POST", rio.Contains("animal/create"))
		inactiveStub := rio.NewStub().For("POST", rio.Contains("animal/create")).WithInactive()
		require.NoError(t, store.Create(ctx, activeStub, inactiveStub))

		last, err := store.GetLastUpdatedStub(ctx, "")
		require.NoError(t, err)
		require.GreaterOrEqual(t, last.ID, inactiveStub.ID)
	})
}
