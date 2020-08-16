package feed_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jfk9w-go/telegram-bot-api/feed"
	"github.com/stretchr/testify/assert"
)

type testClock struct {
	now time.Time
}

func (c *testClock) Now() time.Time {
	return c.now
}

func newTestSQLite3(t *testing.T, clock feed.Clock) *feed.SQLite3 {
	store, err := feed.NewSQLite3(clock, ":memory:")
	assert.Nil(t, err)
	return store
}

func TestSQLite3_Basic(t *testing.T) {
	clock := new(testClock)
	store := newTestSQLite3(t, clock)
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
	clock.now = time.Date(2020, 8, 13, 13, 54, 64, 0, time.UTC)
	sample.UpdatedAt = &clock.now
	err = store.Update(ctx, sample.ID, feed.State{Error: errors.New("test error")})
	assert.Nil(t, err)
	list, err = store.List(ctx, sample.SubID, true)
	assert.Nil(t, err)
	assert.Empty(t, list)
	list, err = store.List(ctx, sample.SubID, false)
	assert.Nil(t, err)
	assert.Equal(t, []feed.Feed{sample}, list)
	stored, err = store.Advance(ctx, sample.SubID)
	assert.Equal(t, feed.ErrNotFound, err)
	stored, err = store.Get(ctx, sample.ID)
	assert.Nil(t, err)
	assert.Equal(t, sample, stored)
	cleared, err := store.Clear(ctx, sample.SubID, "%nontest%")
	assert.Nil(t, err)
	assert.Equal(t, int64(0), cleared)
	cleared, err = store.Clear(ctx, sample.SubID, "%test%")
	assert.Nil(t, err)
	assert.Equal(t, int64(1), cleared)
	stored, err = store.Get(ctx, sample.ID)
	assert.Equal(t, feed.ErrNotFound, err)
	stored, err = store.Advance(ctx, sample.SubID)
	assert.Equal(t, feed.ErrNotFound, err)
}
