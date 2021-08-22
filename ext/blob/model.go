package blob

import "github.com/jfk9w-go/flu"

type Blob interface {
	flu.Input
	flu.Output
}

type Storage interface {
	Alloc() (Blob, error)
}
