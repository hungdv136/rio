package grpc

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	fs "github.com/hungdv136/rio/internal/storage"
	"github.com/hungdv136/rio/internal/log"
	"github.com/hungdv136/rio/internal/util"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/desc/protoparse"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ServiceDescriptor manages descriptor for all services/projects
// Each grpc service has a different proto definitions and dependencies
// These proto files must be compressed into a zip file and uploaded to server
type ServiceDescriptor struct {
	fileStorage fs.FileStorage
	descriptors map[string]*Descriptor // project -> descriptor
	cachedDir   string

	l sync.RWMutex
}

func NewServiceDescriptor(fileStorage fs.FileStorage) *ServiceDescriptor {
	return &ServiceDescriptor{
		cachedDir:   "cached_grpc_protos",
		descriptors: map[string]*Descriptor{},
		fileStorage: fileStorage,
	}
}

// GetDescriptor loads service descriptors from a file storage
func (p *ServiceDescriptor) GetDescriptor(ctx context.Context, protoFileID string) (*Descriptor, error) {
	p.l.Lock()
	defer p.l.Unlock()

	if projectDesc, ok := p.descriptors[protoFileID]; ok {
		return projectDesc, nil
	}

	protoDir := filepath.Join(p.cachedDir, protoFileID)
	if err := p.downloadIfNotExist(ctx, protoDir, protoFileID); err != nil {
		return nil, err
	}

	projectDesc := NewDescriptor()
	if err := projectDesc.init(ctx, protoDir); err != nil {
		return nil, err
	}

	p.descriptors[protoFileID] = projectDesc
	return projectDesc, nil
}

// ClearCache clear cached files
func (p *ServiceDescriptor) ClearCache(ctx context.Context) error {
	p.l.Lock()
	defer p.l.Unlock()

	p.descriptors = map[string]*Descriptor{}

	if err := os.RemoveAll(p.cachedDir); err != nil {
		log.Error(ctx, err)
		return err
	}

	return nil
}

func (p *ServiceDescriptor) downloadIfNotExist(ctx context.Context, outputDir string, protoFileID string) error {
	zipPath := filepath.Join(outputDir, "zip-file")
	if _, err := os.Stat(zipPath); !errors.Is(err, os.ErrNotExist) {
		return err
	}

	reader, err := p.fileStorage.DownloadFile(ctx, protoFileID)
	if err != nil {
		log.Error(ctx, "cannot download file", protoFileID)
		return err
	}
	defer util.CloseSilently(ctx, reader.Close)

	if err := util.WriteToFile(ctx, reader, zipPath); err != nil {
		return err
	}

	log.Info(ctx, "downloaded for", protoFileID, "to", zipPath)
	return util.Unzip(ctx, zipPath, outputDir)
}

// Descriptor loads descriptor from a service/project
type Descriptor struct {
	sdMap map[string]*desc.ServiceDescriptor
	mdMap map[string]*desc.MessageDescriptor

	l sync.RWMutex
}

func NewDescriptor() *Descriptor {
	return &Descriptor{
		sdMap: map[string]*desc.ServiceDescriptor{},
		mdMap: map[string]*desc.MessageDescriptor{},
	}
}

// GetMethod gets method descriptor
// methodPath: full path of method, format must be /<service-name>/<method-name>
// for example: /offers.v1.OfferService/ValidateOffer
func (s *Descriptor) GetMethod(ctx context.Context, methodPath string) (*desc.MethodDescriptor, error) {
	s.l.RLock()
	defer s.l.RUnlock()

	arr := strings.Split(methodPath, "/")
	if len(arr) < 3 {
		err := status.Errorf(codes.InvalidArgument, "invalid method %s", methodPath)
		log.Error(ctx, err)
		return nil, err
	}

	serviceName := arr[1]
	methodName := arr[2]

	service, ok := s.sdMap[serviceName]
	if !ok {
		err := status.Errorf(codes.NotFound, "cannot find service %s", serviceName)
		log.Error(ctx, err)
		return nil, err
	}

	method := service.FindMethodByName(methodName)
	if method == nil {
		err := status.Errorf(codes.NotFound, "cannot find method %s", methodName)
		log.Error(ctx, err)
		return nil, err
	}

	return method, nil
}

// GetAllMethods returns all available methods
func (s *Descriptor) GetAllMethods() []string {
	s.l.RLock()
	defer s.l.RUnlock()

	var result []string
	for _, service := range s.sdMap {
		for _, m := range service.GetMethods() {
			result = append(result, getFullMethod(m))
		}
	}

	return result
}

// GetMessage gets message descriptor
func (s *Descriptor) GetMessage(ctx context.Context, name string) (*desc.MessageDescriptor, error) {
	s.l.RLock()
	defer s.l.RUnlock()

	arr := strings.Split(name, "/")
	name = arr[len(arr)-1]

	msd, ok := s.mdMap[name]
	if !ok {
		err := fmt.Errorf("message type %s not found", name)
		log.Error(ctx, err)
		return nil, err
	}

	return msd, nil
}

// GetAllMessages returns all available messages
func (s *Descriptor) GetAllMessages() []string {
	s.l.RLock()
	defer s.l.RUnlock()

	result := make([]string, 0, len(s.mdMap))
	for k := range s.mdMap {
		result = append(result, k)
	}

	return result
}

func (s *Descriptor) init(ctx context.Context, dir string) error {
	paths, err := s.getProtoFiles(ctx, dir)
	if err != nil {
		return err
	}

	log.Info(ctx, "loading spec for files", paths, "in", dir)

	p := protoparse.Parser{ImportPaths: []string{dir}}
	fdList, err := p.ParseFiles(paths...)
	if err != nil {
		log.Error(ctx, "cannot parse files in", dir, "error", err)
		return err
	}

	for _, fd := range fdList {
		for _, msd := range fd.GetMessageTypes() {
			s.l.Lock()
			s.mdMap[msd.GetFullyQualifiedName()] = msd
			s.l.Unlock()
		}

		for _, rsd := range fd.GetServices() {
			s.l.Lock()
			s.sdMap[rsd.GetFullyQualifiedName()] = rsd
			s.l.Unlock()
		}
	}

	return nil
}

func (s *Descriptor) getProtoFiles(ctx context.Context, dir string) ([]string, error) {
	var paths []string
	filter := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Error(ctx, "cannot read file", path, err)
			return err
		}

		if !strings.HasSuffix(path, ".proto") {
			return nil
		}

		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			log.Error(ctx, "cannot get rel path for", path, err)
			return err
		}

		paths = append(paths, relPath)
		return nil
	}

	if err := filepath.Walk(dir, filter); err != nil {
		log.Error(ctx, "cannot walk dir", dir, err)
		return nil, err
	}

	return paths, nil
}
