package feed_test

import (
	"context"
	"testing"

	"github.com/jfk9w-go/telegram-bot-api/feed"
	"github.com/stretchr/testify/assert"
)

func newTestSQLite3(t *testing.T) *feed.SQLite3 {
	store, err := feed.NewSQLite3(":memory:")
	assert.Nil(t, err)
	return store
}

func TestSQLite3_Basic(t *testing.T) {
	store := newTestSQLite3(t)
	defer store.Close()

	ctx := context.Background()
	activeSubs, err := store.Init(ctx)
	assert.Nil(t, err)
	assert.Empty(t, activeSubs)

	sample := feed.Feed{
		ID:   feed.ID{"1", "test", 1},
		Name: "test feed",
		Data: feed.Data(`{"value": 5}`),
	}

	_, err = store.Get(ctx, sample.ID)
	assert.Equal(t, feed.ErrNotFound, err)
	err = store.Create(ctx, sample)
	assert.Nil(t, err)
	err = store.Create(ctx, sample)
	assert.Equal(t, feed.ErrExists, err)
	stored, err := store.Get(ctx, sample.ID)
	assert.Nil(t, err)
	assert.Equal(t, sample, stored)
	stored, err = store.Advance(ctx, sample.SubID)
	assert.Nil(t, err)
	assert.Equal(t, sample, stored)
	list, err := store.List(ctx, sample.SubID, true)
	assert.Nil(t, err)
	assert.Equal(t, []feed.Feed{sample}, list)
}
