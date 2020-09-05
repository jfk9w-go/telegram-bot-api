package format

import (
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/jfk9w-go/flu"
	"github.com/pkg/errors"
)

type Blob = flu.InputOutput

type Blobs interface {
	Alloc() (Blob, error)
}

type FileBlobStorage struct {
	Clock
	Directory     string
	TTL           time.Duration
	CleanInterval time.Duration
	lastCleanTime time.Time
	files         map[flu.File]time.Time
	work          sync.WaitGroup
	flu.Mutex
}

func (s *FileBlobStorage) Init() (*FileBlobStorage, error) {
	_ = os.RemoveAll(s.Directory)
	if err := os.MkdirAll(s.Directory, 0755); err != nil {
		return nil, errors.Wrapf(err, "create temp dir: %s", s.Directory)
	}

	s.files = make(map[flu.File]time.Time)
	if s.Clock == nil {
		s.Clock = ClockFunc(time.Now)
	}

	return s, nil
}

func (s *FileBlobStorage) clean() {
	defer s.Lock().Unlock()

}

var (
	symbols  = []rune("abcdefghijklmonpqrstuvwxyz0123456789")
	idLength = 16
)

func (s *FileBlobStorage) newID() string {
	id := make([]rune, idLength)
	for i := 0; i < idLength; i++ {
		id[i] = symbols[rand.Intn(len(symbols))]
	}

	return string(id)
}

func (s *FileBlobStorage) Alloc() (Blob, error) {
	defer s.Lock().Unlock()
	now := s.Now()
	file := flu.File(s.Directory + "/" + s.newID())
	s.files[file] = now
	if now.Sub(s.lastCleanTime) > s.CleanInterval {
		for file, createdAt := range s.files {
			if now.Sub(createdAt) > s.TTL {
				_ = os.RemoveAll(file.Path())
			}

			delete(s.files, file)
		}

		s.lastCleanTime = now
	}

	return file, nil
}

func (s *FileBlobStorage) Remove() {
	_ = os.RemoveAll(s.Directory)
}
