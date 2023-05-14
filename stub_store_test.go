package rio

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestStubMemory_CreateAndGetAll(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := NewStubMemory()
	stub := NewStub().For("GET", Contains("animal/create"))
	require.NoError(t, store.Create(ctx, stub))
	require.NotZero(t, stub.ID)

	allItems, err := store.GetAll(ctx, "")
	require.NoError(t, err)
	require.Len(t, allItems, 1)
}

func TestStubMemory_CreateIncomingRequest(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := NewStubMemory()
	request := &IncomingRequest{
		URL:    uuid.NewString(),
		Method: uuid.NewString(),
	}

	require.NoError(t, store.CreateIncomingRequest(ctx, request))
	require.NotZero(t, request.ID)

	requests, err := store.GetIncomingRequests(ctx, &IncomingQueryOption{})
	require.NoError(t, err)
	require.Len(t, requests, 1)
}
