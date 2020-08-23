package feed

import (
	"context"
	"log"
	"time"

	telegram "github.com/jfk9w-go/telegram-bot-api"

	"github.com/jfk9w-go/telegram-bot-api/format"
	"github.com/pkg/errors"
)

type SuspendListener interface {
	OnSuspend(sub Sub, err error)
}

type HTMLWriterFactory interface {
	CreateHTMLWriter(ctx context.Context, feedIDs ...ID) (*format.HTMLWriter, error)
}

type TelegramHTML struct {
	telegram.Sender
}

func (f TelegramHTML) CreateHTMLWriter(ctx context.Context, feedIDs ...ID) (*format.HTMLWriter, error) {
	chatIDs := make([]telegram.ChatID, len(feedIDs))
	for i, feedID := range feedIDs {
		chatIDs[i] = telegram.ID(feedID)
	}

	return format.HTML(ctx, telegram.Sender(f), false, chatIDs...), nil
}

type aggregatorTask struct {
	htmlWriterFactory HTMLWriterFactory
	store             Store
	interval          time.Duration
	vendors           map[string]Vendor
	feedID            ID
	suspendListener   SuspendListener
}

func (t *aggregatorTask) Execute(ctx context.Context) error {
	html, err := t.htmlWriterFactory.CreateHTMLWriter(ctx, t.feedID)
	if err != nil {
		return errors.Wrap(err, "create HTMLWriter")
	}
	for {
		sub, err := t.store.Advance(ctx, t.feedID)
		if err != nil {
			return errors.Wrap(err, "advance")
		}
		vendor, ok := t.vendors[sub.Vendor]
		if !ok {
			return errors.Errorf("invalid vendor: %s", sub.Vendor)
		}
		if err := t.update(ctx, html, sub, vendor); err != nil {
			updateErr := err
			if ctx.Err() != nil {
				return err
			} else if err := t.store.Update(ctx, sub.SubID, err); err != nil {
				if ctx.Err() != nil {
					return err
				} else {
					log.Printf("[sub-%s] update failed: %s", sub.SubID, err)
				}
			} else {
				go t.suspendListener.OnSuspend(sub, updateErr)
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(t.interval):
			continue
		}
	}
}

func (t *aggregatorTask) update(ctx context.Context, html *format.HTMLWriter, sub Sub, vendor Vendor) error {
	queue := make(chan Update, 5)
	vctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go vendor.Load(vctx, sub.Data, queue)
	count := 0
	for update := range queue {
		if update.Error != nil {
			return errors.Wrap(update.Error, "update")
		}
		err := update.Write(html).Flush()
		if err != nil {
			return errors.Wrap(err, "write")
		}
		data, err := DataFrom(update.Data)
		if err != nil {
			return errors.Wrap(err, "wrap data")
		}
		err = t.store.Update(ctx, sub.SubID, data)
		if err != nil {
			return errors.Wrap(err, "store update")
		}
		count++
	}

	if count == 0 {
		err := t.store.Update(ctx, sub.SubID, sub.Data)
		if err != nil {
			return errors.Wrap(err, "store update")
		}
	}

	log.Printf("[sub-%s] processed %d updates", sub.SubID, count)
	return nil
}

type Aggregator struct {
	executor          TaskExecutor
	store             Store
	htmlWriterFactory HTMLWriterFactory
	vendors           map[string]Vendor
	interval          time.Duration
}

func NewAggregator(executor TaskExecutor, store Store, htmlWriterFactory HTMLWriterFactory, interval time.Duration) *Aggregator {
	return &Aggregator{
		executor:          executor,
		store:             store,
		htmlWriterFactory: htmlWriterFactory,
		vendors:           make(map[string]Vendor),
		interval:          interval,
	}
}

func (a *Aggregator) Vendor(id string, vendor Vendor) *Aggregator {
	a.vendors[id] = vendor
	return a
}

func (a *Aggregator) submitTask(feedID ID) {
	a.executor.Submit(feedID, &aggregatorTask{
		htmlWriterFactory: a.htmlWriterFactory,
		store:             a.store,
		interval:          a.interval,
		vendors:           a.vendors,
		feedID:            feedID,
	})
}

func (a *Aggregator) Subscribe(ctx context.Context, feedID ID, ref, options string) (Sub, error) {
	for vendorID, vendor := range a.vendors {
		sub, err := vendor.Parse(ctx, ref, options)
		switch err {
		case nil:
			data, err := DataFrom(sub.Data)
			sub := Sub{
				SubID: SubID{
					ID:     sub.ID,
					Vendor: vendorID,
					FeedID: feedID,
				},
				Name: sub.Name,
				Data: data,
			}

			if err != nil {
				return sub, errors.Wrap(err, "wrap data")
			}

			if err := a.store.Create(ctx, sub); err != nil {
				return sub, err
			}

			a.submitTask(feedID)
			return sub, nil

		case ErrWrongVendor:
			continue

		default:
			return Sub{}, err
		}
	}

	return Sub{}, ErrWrongVendor
}

func (a *Aggregator) Suspend(ctx context.Context, subID SubID, err error) (Sub, error) {
	if err := a.store.Update(ctx, subID, err); err != nil {
		return Sub{}, err
	}
	return a.store.Get(ctx, subID)
}

func (a *Aggregator) Resume(ctx context.Context, subID SubID) (Sub, error) {
	if err := a.store.Update(ctx, subID, nil); err != nil {
		return Sub{}, err
	}
	sub, err := a.store.Get(ctx, subID)
	if err != nil {
		return Sub{}, err
	}

	a.submitTask(subID.FeedID)
	return sub, nil
}

func (a *Aggregator) Delete(ctx context.Context, subID SubID) (Sub, error) {
	sub, err := a.store.Get(ctx, subID)
	if err != nil {
		return Sub{}, errors.Wrap(err, "get")
	}
	if err := a.store.Delete(ctx, subID); err != nil {
		return Sub{}, errors.Wrap(err, "store delete")
	}
	return sub, nil
}

func (a *Aggregator) Clear(ctx context.Context, feedID ID, pattern string) (int64, error) {
	return a.store.Clear(ctx, feedID, pattern)
}

func (a *Aggregator) List(ctx context.Context, feedID ID, active bool) ([]Sub, error) {
	return a.store.List(ctx, feedID, active)
}
