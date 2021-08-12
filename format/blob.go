package format

import (
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/jfk9w-go/flu"
	"github.com/pkg/errors"
)

type Blob interface {
	flu.Input
	flu.Output
}

type Blobs interface {
	Alloc() (Blob, error)
}

type FileBlobStorage struct {
	flu.Clock
	Directory     string
	TTL           time.Duration
	CleanInterval time.Duration
	lastCleanTime time.Time
	files         map[flu.File]time.Time
	work          sync.WaitGroup
	flu.Mutex
}

func (s *FileBlobStorage) Init() (*FileBlobStorage, error) {
	s.Remove()
	if err := os.MkdirAll(s.Directory, 0755); err != nil {
		return nil, errors.Wrapf(err, "create temp dir: %s", s.Directory)
	}

	s.files = make(map[flu.File]time.Time)
	if s.Clock == nil {
		s.Clock = flu.DefaultClock
	}

	return s, nil
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
		count := 0
		for file, createdAt := range s.files {
			if now.Sub(createdAt) > s.TTL {
				if err := os.RemoveAll(file.Path()); err != nil {
					logrus.WithFields(logrus.Fields{
						"file": file,
					}).Warnf("failed to remove blob file: %s", err)
					continue
				}

				delete(s.files, file)
				count++
			}
		}

		s.lastCleanTime = now
		logrus.Debugf("removed %s blob files", count)
	}

	return file, nil
}

func (s *FileBlobStorage) Remove() {
	_ = os.RemoveAll(s.Directory)
}
