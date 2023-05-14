package database

import (
	"context"
	"errors"

	"github.com/hungdv136/rio"
	"github.com/hungdv136/rio/internal/config"
	"github.com/hungdv136/rio/internal/log"
	"gorm.io/gorm"
)

const LatestVersion = 3

var (
	_ rio.StubStore   = (*StubDBStore)(nil)
	_ rio.StatusStore = (*StubDBStore)(nil)
)

// StubDBStore stores stub information to database
type StubDBStore struct {
	db *gorm.DB
}

// NewStubDBStore init a new instance for storage
func NewStubDBStore(ctx context.Context, config *config.MySQLConfig) (*StubDBStore, error) {
	db, err := Connect(ctx, config)
	if err != nil {
		log.Error(ctx, "cannot connect db", err)
		return nil, err
	}

	return &StubDBStore{db: db}, nil
}

// Create creates a new stub
func (s *StubDBStore) Create(ctx context.Context, stubs ...*rio.Stub) error {
	for _, stub := range stubs {
		stub.Settings.StoreVersion = LatestVersion
	}

	if err := s.db.WithContext(ctx).Create(stubs).Error; err != nil {
		log.Error(ctx, "cannot create stubs", err)
		return err
	}

	return nil
}

// CreateProto creates new protos
func (s *StubDBStore) CreateProto(ctx context.Context, protos ...*rio.Proto) error {
	if err := s.db.WithContext(ctx).Create(protos).Error; err != nil {
		log.Error(ctx, "cannot create protos", err)
		return err
	}

	return nil
}

// Delete marks record as inactive
func (s *StubDBStore) Delete(ctx context.Context, id int64) error {
	db := s.db.WithContext(ctx).Model(rio.Stub{}).Where("id = ?", id).Update("active", false)
	if err := db.Error; err != nil {
		log.Error(ctx, "cannot delete stubs", err)
		return err
	}

	return nil
}

// GetAll gets all saved records
func (s *StubDBStore) GetAll(ctx context.Context, namespace string) ([]*rio.Stub, error) {
	stubs := []*rio.Stub{}
	err := s.db.WithContext(ctx).Where("namespace = ? AND active = ?", namespace, true).Order("id DESC").Find(&stubs).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		log.Error(ctx, "cannot get all stubs", err)
		return nil, err
	}

	return stubs, nil
}

// Find finds by id
func (s *StubDBStore) Find(ctx context.Context, id int64) (*rio.Stub, error) {
	stub := rio.Stub{}
	if err := s.db.WithContext(ctx).Where("id = ?", id).Last(&stub).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		log.Error(ctx, "cannot find stubs", err)
		return nil, err
	}

	return &stub, nil
}

// CreateIncomingRequest saves the income request
func (s *StubDBStore) CreateIncomingRequest(ctx context.Context, r *rio.IncomingRequest) error {
	if err := s.db.WithContext(ctx).Create(r).Error; err != nil {
		log.Error(ctx, "cannot find income request", err)
		return err
	}

	return nil
}

// GetIncomingRequests finds a income request by id
func (s *StubDBStore) GetIncomingRequests(ctx context.Context, option *rio.IncomingQueryOption) ([]*rio.IncomingRequest, error) {
	r := []*rio.IncomingRequest{}
	db := s.db.WithContext(ctx).Where("namespace = ?", option.Namespace)
	if len(option.Ids) > 0 {
		db = db.Where("id IN (?)", option.Ids)
	}

	if err := db.Order("id DESC").Limit(option.Limit).Find(&r).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		log.Error(ctx, "cannot get incoming request", err)
		return nil, err
	}

	return r, nil
}

// GetProtos gets a list of protos
func (s *StubDBStore) GetProtos(ctx context.Context) ([]*rio.Proto, error) {
	result := []*rio.Proto{}
	if err := s.db.WithContext(ctx).Order("id DESC").Find(&result).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		log.Error(ctx, "cannot get proto", err)
		return nil, err
	}

	return result, nil
}

// Reset clear data
func (s *StubDBStore) Reset(ctx context.Context, option *rio.ResetQueryOption) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		resetQuery := s.db.WithContext(ctx)
		resetRequest := s.db.WithContext(ctx)

		if option.Namespace == rio.ResetAll {
			resetQuery = resetQuery.Where("1 = 1")
			resetRequest = resetRequest.Where("1 = 1")
		} else {
			resetQuery = resetQuery.Where("namespace = ?", option.Namespace)
			resetRequest = resetRequest.Where("namespace = ?", option.Namespace)
		}

		if len(option.Tag) > 0 {
			resetQuery = resetQuery.Where("tag = ?", option.Tag)
			resetRequest = resetRequest.Where("tag = ?", option.Tag)
		}

		if err := resetQuery.Delete(&rio.Stub{}).Error; err != nil {
			log.Error(ctx, "cannot delete stubs request", err)
			return err
		}

		if err := resetRequest.Delete(&rio.IncomingRequest{}).Error; err != nil {
			log.Error(ctx, "cannot delete incoming request", err)
			return err
		}

		return nil
	})
}

func (s *StubDBStore) GetLastUpdatedStub(ctx context.Context, namespace string) (*rio.LastUpdatedRecord, error) {
	var r rio.LastUpdatedRecord
	db := s.db.WithContext(ctx).
		Model(rio.Stub{}).
		Select("id, updated_at").
		Where("namespace = ? AND tag <> ?", namespace, rio.TagRecordedStub).
		Order("updated_at desc, id desc").
		Last(&r)
	if err := db.Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		log.Error(ctx, "cannot get stub", err)
		return nil, err
	}

	return &r, nil
}

func (s *StubDBStore) GetLastUpdatedProto(ctx context.Context) (*rio.LastUpdatedRecord, error) {
	var r rio.LastUpdatedRecord
	db := s.db.WithContext(ctx).
		Model(rio.Proto{}).
		Select("id, updated_at").
		Order("updated_at desc, id desc").
		Last(&r)
	if err := db.Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		log.Error(ctx, "cannot get proto", err)
		return nil, err
	}

	return &r, nil
}
