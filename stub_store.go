package rio

//go:generate mockgen -source stub_store.go -destination internal/test/mock/stub_store.go -package mock

import (
	"context"
	"sync"
	"time"

	"github.com/hungdv136/rio/internal/util"
)

var _ StubStore = (*StubMemory)(nil)

const ResetAll = "reset_all"

// IncomingQueryOption incoming query option
type IncomingQueryOption struct {
	Namespace string  `json:"namespace" yaml:"namespace"`
	Ids       []int64 `json:"ids" yaml:"ids"`
	Limit     int     `json:"limit" yaml:"limit"`
}

type ResetQueryOption struct {
	Namespace string `json:"namespace" yaml:"namespace"`
	Tag       string `json:"tag" yaml:"tag"`
}

// StubStore stores the stub information
type StubStore interface {
	Create(ctx context.Context, stubs ...*Stub) error
	Delete(ctx context.Context, id int64) error
	GetAll(ctx context.Context, namespace string) ([]*Stub, error)
	CreateProto(ctx context.Context, protos ...*Proto) error
	GetProtos(ctx context.Context) ([]*Proto, error)
	CreateIncomingRequest(ctx context.Context, r *IncomingRequest) error
	GetIncomingRequests(ctx context.Context, option *IncomingQueryOption) ([]*IncomingRequest, error)
	Reset(ctx context.Context, option *ResetQueryOption) error
}

// LastUpdatedRecord holds the id and updated at
type LastUpdatedRecord struct {
	ID        int64     `json:"id"`
	UpdatedAt time.Time `json:"updated_at"`
}

// StatusStore defines interface to get latest updated information
type StatusStore interface {
	GetLastUpdatedStub(ctx context.Context, namespace string) (*LastUpdatedRecord, error)
	GetLastUpdatedProto(ctx context.Context) (*LastUpdatedRecord, error)
}

// StubMemory implements in memory store which using for unit test
type StubMemory struct {
	stubs          []*Stub
	protos         []*Proto
	incomeRequests []*IncomingRequest
	id             int64
	l              sync.RWMutex
}

// NewStubMemory returns a new instance
func NewStubMemory() *StubMemory {
	return &StubMemory{}
}

// Create adds to memory
func (db *StubMemory) Create(ctx context.Context, stubs ...*Stub) error {
	db.l.Lock()
	defer db.l.Unlock()

	for _, stub := range stubs {
		if stub.ID == 0 {
			db.id++
			stub.ID = db.id
		}

		db.stubs = append(db.stubs, stub)
	}

	return nil
}

// Delete deletes a stub
func (db *StubMemory) Delete(ctx context.Context, id int64) error {
	db.l.Lock()
	defer db.l.Unlock()

	records := make([]*Stub, 0, len(db.stubs))
	for _, r := range db.stubs {
		if r.ID != id {
			records = append(records, r)
		}
	}

	db.stubs = records
	return nil
}

// GetAll gets all records
func (db *StubMemory) GetAll(_ context.Context, namespace string) ([]*Stub, error) {
	db.l.RLock()
	defer db.l.RUnlock()

	records := make([]*Stub, 0, len(db.stubs))

	for i := len(db.stubs) - 1; i >= 0; i-- {
		r := db.stubs[i]
		if r.Namespace == namespace {
			records = append(records, r)
		}
	}

	return records, nil
}

// CreateIncomingRequest saves the incomes request
func (db *StubMemory) CreateIncomingRequest(ctx context.Context, r *IncomingRequest) error {
	db.l.Lock()
	defer db.l.Unlock()

	if r.ID == 0 {
		db.id++
		r.ID = db.id
	}

	db.incomeRequests = append(db.incomeRequests, r)
	return nil
}

// GetIncomingRequests returns incoming requests
func (db *StubMemory) GetIncomingRequests(ctx context.Context, option *IncomingQueryOption) ([]*IncomingRequest, error) {
	db.l.RLock()
	defer db.l.RUnlock()

	incomeRequests := make([]*IncomingRequest, 0, len(db.incomeRequests))
	for i := len(db.incomeRequests) - 1; i >= 0; i-- {
		if option.Limit > 0 && len(incomeRequests) >= option.Limit {
			break
		}

		r := db.incomeRequests[i]
		if r.Namespace != option.Namespace {
			continue
		}

		if len(option.Ids) > 0 && !util.ArrayContains(option.Ids, r.ID) {
			continue
		}

		incomeRequests = append(incomeRequests, r)
	}

	return incomeRequests, nil
}

func (db *StubMemory) CreateProto(ctx context.Context, protos ...*Proto) error {
	db.l.Lock()
	defer db.l.Unlock()

	db.protos = append(db.protos, protos...)
	return nil
}

func (db *StubMemory) GetProtos(ctx context.Context) ([]*Proto, error) {
	db.l.RLock()
	defer db.l.RUnlock()

	return db.protos, nil
}

// Reset clear data
func (db *StubMemory) Reset(ctx context.Context, option *ResetQueryOption) error {
	db.stubs = []*Stub{}
	db.incomeRequests = []*IncomingRequest{}
	return nil
}
