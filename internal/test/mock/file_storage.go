// Code generated by MockGen. DO NOT EDIT.
// Source: file-storage.go

// Package mock is a generated GoMock package.
package mock

import (
	context "context"
	io "io"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
)

// MockFileStorage is a mock of FileStorage interface.
type MockFileStorage struct {
	ctrl     *gomock.Controller
	recorder *MockFileStorageMockRecorder
}

// MockFileStorageMockRecorder is the mock recorder for MockFileStorage.
type MockFileStorageMockRecorder struct {
	mock *MockFileStorage
}

// NewMockFileStorage creates a new mock instance.
func NewMockFileStorage(ctrl *gomock.Controller) *MockFileStorage {
	mock := &MockFileStorage{ctrl: ctrl}
	mock.recorder = &MockFileStorageMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockFileStorage) EXPECT() *MockFileStorageMockRecorder {
	return m.recorder
}

// DeleteFile mocks base method.
func (m *MockFileStorage) DeleteFile(ctx context.Context, objectKey string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteFile", ctx, objectKey)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteFile indicates an expected call of DeleteFile.
func (mr *MockFileStorageMockRecorder) DeleteFile(ctx, objectKey interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteFile", reflect.TypeOf((*MockFileStorage)(nil).DeleteFile), ctx, objectKey)
}

// DownloadFile mocks base method.
func (m *MockFileStorage) DownloadFile(ctx context.Context, objectKey string) (io.ReadCloser, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DownloadFile", ctx, objectKey)
	ret0, _ := ret[0].(io.ReadCloser)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DownloadFile indicates an expected call of DownloadFile.
func (mr *MockFileStorageMockRecorder) DownloadFile(ctx, objectKey interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DownloadFile", reflect.TypeOf((*MockFileStorage)(nil).DownloadFile), ctx, objectKey)
}

// ListFiles mocks base method.
func (m *MockFileStorage) ListFiles(ctx context.Context) ([]string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListFiles", ctx)
	ret0, _ := ret[0].([]string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListFiles indicates an expected call of ListFiles.
func (mr *MockFileStorageMockRecorder) ListFiles(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListFiles", reflect.TypeOf((*MockFileStorage)(nil).ListFiles), ctx)
}

// Reset mocks base method.
func (m *MockFileStorage) Reset(ctx context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Reset", ctx)
	ret0, _ := ret[0].(error)
	return ret0
}

// Reset indicates an expected call of Reset.
func (mr *MockFileStorageMockRecorder) Reset(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Reset", reflect.TypeOf((*MockFileStorage)(nil).Reset), ctx)
}

// UploadFile mocks base method.
func (m *MockFileStorage) UploadFile(ctx context.Context, objectKey string, reader io.Reader) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UploadFile", ctx, objectKey, reader)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UploadFile indicates an expected call of UploadFile.
func (mr *MockFileStorageMockRecorder) UploadFile(ctx, objectKey, reader interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UploadFile", reflect.TypeOf((*MockFileStorage)(nil).UploadFile), ctx, objectKey, reader)
}
