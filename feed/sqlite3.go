package feed

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/sqlite3"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
)

type Clock interface {
	Now() time.Time
}

type ClockFunc func() time.Time

func (fun ClockFunc) Now() time.Time {
	return fun()
}

var SQLite3TableName = "feed"

type SQLBuilder interface {
	ToSQL() (string, []interface{}, error)
}

type SQLite3 struct {
	*goqu.Database
	Clock
	mu sync.RWMutex
}

func NewSQLite3(clock Clock, datasource string) (*SQLite3, error) {
	dialect := "sqlite3"
	db, err := sql.Open(dialect, datasource)
	if err != nil {
		return nil, err
	}

	if clock == nil {
		clock = ClockFunc(time.Now)
	}

	return &SQLite3{Database: goqu.New(dialect, db), Clock: clock}, nil
}

func (s *SQLite3) Init(ctx context.Context) ([]SubID, error) {
	sql := fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS %s (
	  id VARCHAR(63) NOT NULL,
	  type VARCHAR(31) NOT NULL,
	  sub_id INTEGER NOT NULL,
      name VARCHAR(255) NOT NULL,
	  data TEXT,
	  updated_at TIMESTAMP,
	  error VARCHAR(255)
	)`, SQLite3TableName)
	if _, err := s.Database.ExecContext(ctx, sql); err != nil {
		return nil, errors.Wrap(err, "create table")
	}
	sql = fmt.Sprintf(`
	CREATE UNIQUE INDEX IF NOT EXISTS i__%s__id 
	ON %s(id, type, sub_id)`, SQLite3TableName, SQLite3TableName)
	if _, err := s.Database.ExecContext(ctx, sql); err != nil {
		return nil, errors.Wrap(err, "create index")
	}
	activeSubs := make([]SubID, 0)
	err := s.Select(goqu.DISTINCT("sub_id")).From(SQLite3TableName).ScanValsContext(ctx, &activeSubs)
	if err != nil {
		return nil, errors.Wrap(err, "select active subs")
	}
	return activeSubs, nil
}

func (s *SQLite3) ExecuteSQLBuilder(ctx context.Context, builder SQLBuilder) (int64, error) {
	sql, args, err := builder.ToSQL()
	if err != nil {
		return 0, errors.Wrap(err, "build sql")
	}
	result, err := s.Database.ExecContext(ctx, sql, args...)
	if err != nil {
		return 0, errors.Wrap(err, "execute")
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, errors.Wrap(err, "rows affected")
	}
	return affected, nil
}

func (s *SQLite3) UpdateSQLBuilder(ctx context.Context, builder SQLBuilder) (bool, error) {
	affected, err := s.ExecuteSQLBuilder(ctx, builder)
	return affected > 0, err
}

func (s *SQLite3) Create(ctx context.Context, feed Feed) error {
	defer s.WLock().Unlock()
	ok, err := s.UpdateSQLBuilder(ctx, s.Insert(SQLite3TableName).Rows(feed).OnConflict(goqu.DoNothing()))
	if err == nil && !ok {
		err = ErrExists
	}

	return err
}

func (s *SQLite3) Get(ctx context.Context, id ID) (Feed, error) {
	defer s.RLock().Unlock()
	var feed Feed
	ok, err := s.Select(Feed{}).
		From(SQLite3TableName).
		Where(s.ByID(id)).
		Limit(1).
		ScanStructContext(ctx, &feed)
	if err == nil && !ok {
		err = ErrNotFound
	}

	return feed, err
}

func (s *SQLite3) Advance(ctx context.Context, subID SubID) (Feed, error) {
	defer s.RLock().Unlock()
	var feed Feed
	ok, err := s.Select(Feed{}).
		From(SQLite3TableName).
		Where(goqu.And(
			goqu.C("sub_id").Eq(subID),
			goqu.C("error").IsNull(),
		)).
		Order(goqu.I("updated_at").Asc().NullsFirst()).
		Limit(1).
		ScanStructContext(ctx, &feed)
	if err == nil && !ok {
		err = ErrNotFound
	}

	return feed, err
}

func (s *SQLite3) List(ctx context.Context, subID SubID, active bool) ([]Feed, error) {
	defer s.RLock().Unlock()
	feeds := make([]Feed, 0)
	err := s.Select(Feed{}).
		From(SQLite3TableName).
		Where(goqu.And(
			goqu.C("sub_id").Eq(subID),
			goqu.Literal("error IS NULL").Eq(active),
		)).
		ScanStructsContext(ctx, &feeds)
	return feeds, err
}

func (s *SQLite3) Clear(ctx context.Context, subID SubID, pattern string) (int64, error) {
	defer s.WLock().Unlock()
	return s.ExecuteSQLBuilder(ctx, s.Database.Delete(SQLite3TableName).
		Where(goqu.And(
			goqu.C("sub_id").Eq(subID),
			goqu.C("error").Like(pattern),
		)))
}

func (s *SQLite3) Delete(ctx context.Context, id ID) error {
	defer s.WLock().Unlock()
	ok, err := s.UpdateSQLBuilder(ctx, s.Database.Delete(SQLite3TableName).Where(s.ByID(id)))
	if err == nil && !ok {
		err = ErrNotFound
	}

	return err
}

func (s *SQLite3) Update(ctx context.Context, id ID, state State) error {
	defer s.WLock().Unlock()
	where := s.ByID(id)
	update := map[string]interface{}{"updated_at": s.Now()}
	switch {
	case state.Data != ZeroData:
		where = goqu.And(where, goqu.C("error").IsNull())
		update["data"] = state.Data
	case state.Error == nil:
		where = goqu.And(where, goqu.C("error").IsNotNull())
		update["error"] = nil
	default:
		where = goqu.And(where, goqu.C("error").IsNull())
		update["error"] = state.Error.Error()
	}

	ok, err := s.UpdateSQLBuilder(ctx, s.Database.Update(SQLite3TableName).Set(update).Where(where))
	if err == nil && !ok {
		err = ErrNotFound
	}

	return err
}

func (s *SQLite3) Close() error {
	return s.Db.(*sql.DB).Close()
}

func (s *SQLite3) ByID(id ID) goqu.Expression {
	return goqu.Ex{
		"id":     id.ID,
		"type":   id.Type,
		"sub_id": id.SubID,
	}
}

func (s *SQLite3) RLock() UnlockFunc {
	s.mu.RLock()
	return s.mu.RUnlock
}

func (s *SQLite3) WLock() UnlockFunc {
	s.mu.Lock()
	return s.mu.Unlock
}

type UnlockFunc func()

func (fun UnlockFunc) Unlock() {
	fun()
}
