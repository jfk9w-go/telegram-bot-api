package feed

import (
	"context"
	"crypto/md5"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jfk9w-go/flu/metrics"

	"github.com/jfk9w-go/flu"
	fluhttp "github.com/jfk9w-go/flu/http"
	telegram "github.com/jfk9w-go/telegram-bot-api"
	"github.com/jfk9w-go/telegram-bot-api/format"
	"github.com/pkg/errors"
)

type (
	MediaResolver interface {
		GetClient() *fluhttp.Client
		ResolveURL(ctx context.Context, client *fluhttp.Client, url string, maxSize int64) (string, error)
		Request(request *fluhttp.Request) *fluhttp.Request
	}

	MediaConverter interface {
		MIMETypes() []string
		Convert(ref *MediaRef) (format.MediaRef, error)
	}

	MediaDedup interface {
		Check(ctx context.Context, feedID ID, url string, blob format.Blob) error
	}
)

type DummyMediaResolver struct {
	Client *fluhttp.Client
}

func (r DummyMediaResolver) GetClient() *fluhttp.Client {
	return r.Client
}

func (r DummyMediaResolver) ResolveURL(_ context.Context, _ *fluhttp.Client, url string, _ int64) (string, error) {
	return url, nil
}

func (r DummyMediaResolver) Request(request *fluhttp.Request) *fluhttp.Request {
	return request
}

type MediaManager struct {
	DefaultClient *fluhttp.Client
	SizeBounds    [2]int64
	Storage       format.Blobs
	Converters    map[string]MediaConverter
	Dedup         MediaDedup
	RateLimiter   flu.RateLimiter
	Metrics       metrics.Registry
	ctx           context.Context
	cancel        func()
	work          sync.WaitGroup
}

func (m *MediaManager) Init(ctx context.Context) *MediaManager {
	if m.Metrics == nil {
		m.Metrics = metrics.DummyRegistry{}
	}

	ctx, cancel := context.WithCancel(ctx)
	m.ctx = ctx
	m.cancel = cancel
	return m
}

func (m *MediaManager) Submit(ref *MediaRef) format.MediaRef {
	m.work.Add(1)
	mvar := format.NewMediaVar()
	ctx, cancel := context.WithTimeout(m.ctx, 10*time.Minute)
	ref.Manager = m
	go func() {
		defer m.work.Done()
		defer cancel()
		if err := m.RateLimiter.Start(ctx); err != nil {
			log.Printf("[media > %s] failed to process: %s", ref.URL, err)
			return
		}

		defer m.RateLimiter.Complete()
		media, err := ref.Get(ctx)
		mvar.Set(media, err)
	}()

	return mvar
}

func (m *MediaManager) Converter(converter MediaConverter) *MediaManager {
	if m.Converters == nil {
		m.Converters = map[string]MediaConverter{}
	}

	for _, mimeType := range converter.MIMETypes() {
		m.Converters[mimeType] = converter
	}

	return m
}

func (m *MediaManager) Close() {
	m.cancel()
	m.work.Wait()
}

type MediaRef struct {
	MediaResolver
	Manager     *MediaManager
	URL         string
	Dedup       bool
	Blob        bool
	FeedID      ID
	ResolvedURL string
	MIMEType    string
	Size        int64
}

func (r *MediaRef) getClient() *fluhttp.Client {
	if r.GetClient() != nil {
		return r.GetClient()
	} else {
		return r.Manager.DefaultClient
	}
}

func (r *MediaRef) Handle(resp *http.Response) error {
	r.MIMEType = strings.Split(resp.Header.Get("Content-Type"), ";")[0]
	var err error
	r.Size, err = strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)
	if err != nil {
		return errors.Wrap(err, "parse content length")
	}

	return nil
}

func (r *MediaRef) Get(ctx context.Context) (format.Media, error) {
	var err error
	r.ResolvedURL, err = r.ResolveURL(ctx, r.getClient(), r.URL, telegram.Video.AttachMaxSize())
	if err != nil {
		r.Manager.Metrics.Counter("err", metrics.Labels{
			"feed_id", PrintID(r.FeedID),
			"mime_type", "unknown",
			"err", "resolve url",
		}).Inc()
		return format.Media{}, errors.Wrapf(err, "resolve url: %s", r.URL)
	}

	if r.MIMEType == "" && r.Size == 0 {
		if err := r.Request(r.getClient().HEAD(r.ResolvedURL)).
			Context(ctx).
			Execute().
			HandleResponse(r).
			Error; err != nil {
			r.Manager.Metrics.Counter("err", metrics.Labels{
				"feed_id", PrintID(r.FeedID),
				"mime_type", "unknown",
				"err", "head",
			}).Inc()
			return format.Media{}, errors.Wrap(err, "head")
		}

		if r.Size < r.Manager.SizeBounds[0] {
			r.Manager.Metrics.Counter("err", metrics.Labels{
				"feed_id", PrintID(r.FeedID),
				"mime_type", r.MIMEType,
				"err", "too small",
			}).Inc()
			return format.Media{}, errors.Errorf("size of %db is too low", r.Size)
		} else if r.Size > r.Manager.SizeBounds[1] {
			r.Manager.Metrics.Counter("err", metrics.Labels{
				"feed_id", PrintID(r.FeedID),
				"mime_type", r.MIMEType,
				"err", "too large",
			}).Inc()
			return format.Media{}, errors.Errorf("size %dMb too large", r.Size>>20)
		}
	}

	mimeType := r.MIMEType
	if converter, ok := r.Manager.Converters[mimeType]; ok {
		ref, err := converter.Convert(r)
		if err != nil {
			r.Manager.Metrics.Counter("err", metrics.Labels{
				"feed_id", PrintID(r.FeedID),
				"mime_type", r.MIMEType,
				"err", "convert",
			}).Inc()
			return format.Media{}, errors.Wrapf(err, "convert from %s", mimeType)
		}

		return ref.Get(ctx)
	}

	mediaType := telegram.MediaTypeByMIMEType(mimeType)
	if mediaType == telegram.DefaultMediaType {
		r.Manager.Metrics.Counter("err", metrics.Labels{
			"feed_id", PrintID(r.FeedID),
			"mime_type", r.MIMEType,
			"err", "mime",
		}).Inc()
		return format.Media{}, errors.Errorf("unsupported mime type: %s", mimeType)
	}

	if r.Size <= mediaType.RemoteMaxSize() && !r.Dedup && !r.Blob {
		r.Manager.Metrics.Counter("ok", metrics.Labels{
			"feed_id", PrintID(r.FeedID),
			"mime_type", r.MIMEType,
			"method", "remote",
		}).Inc()
		return format.Media{
			MIMEType: mimeType,
			Input:    flu.URL(r.ResolvedURL),
		}, nil
	}

	if r.Size <= mediaType.AttachMaxSize() {
		blob, err := r.Manager.Storage.Alloc()
		if err != nil {
			return format.Media{}, errors.Wrap(err, "create blob")
		}

		if err := r.Request(r.getClient().GET(r.ResolvedURL)).
			Context(ctx).
			Execute().
			CheckStatus(http.StatusOK).
			DecodeBodyTo(blob).
			Error; err != nil {
			r.Manager.Metrics.Counter("err", metrics.Labels{
				"feed_id", PrintID(r.FeedID),
				"mime_type", r.MIMEType,
				"err", "download",
			}).Inc()
			return format.Media{}, errors.Wrap(err, "download")
		}

		if r.Dedup {
			if err := r.Manager.Dedup.Check(ctx, r.FeedID, r.URL, blob); err != nil {
				r.Manager.Metrics.Counter("err", metrics.Labels{
					"feed_id", PrintID(r.FeedID),
					"mime_type", r.MIMEType,
					"err", "dedup",
				}).Inc()

				log.Printf("[media > %d > %s] failed dedup check: %s", r.FeedID, r.URL, err)
				return format.Media{}, err
			}
		}

		r.Manager.Metrics.Counter("ok", metrics.Labels{
			"feed_id", PrintID(r.FeedID),
			"mime_type", r.MIMEType,
			"method", "attach",
		}).Inc()
		return format.Media{
			MIMEType: mimeType,
			Input:    blob,
		}, nil
	}

	r.Manager.Metrics.Counter("err", metrics.Labels{
		"feed_id", PrintID(r.FeedID),
		"mime_type", r.MIMEType,
		"err", "too large",
	}).Inc()
	return format.Media{}, errors.Errorf("size %dMb is too large", r.Size>>20)
}

type MD5MediaDedup struct {
	Hashes Hashes
}

func (d MD5MediaDedup) Check(ctx context.Context, feedID ID, url string, blob format.Blob) error {
	hash := md5.New()
	if err := flu.Copy(blob, flu.IO{W: hash}); err != nil {
		return errors.Wrap(err, "hash")
	}
	return d.Hashes.Check(ctx, feedID, url, fmt.Sprintf("%x", hash.Sum(nil)))
}