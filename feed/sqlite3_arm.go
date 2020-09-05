package feed

import (
	"context"

	"github.com/doug-martin/goqu/v9"
	"github.com/pkg/errors"
)

func selectSubs(ctx context.Context, builder *goqu.SelectDataset) ([]Sub, error) {
	rows, err := s.QuerySQLBuilder(ctx, builder.
		Select(
			goqu.C("sub_id"),
			goqu.C("vendor"),
			goqu.C("feed_id"),
			goqu.C("name"),
			goqu.C("data"),
			goqu.C("updated_at")))
	if err != nil {
		return nil, errors.Wrap(err, "query")
	}

	defer rows.Close()
	subs := make([]Sub, 0)
	for rows.Next() {
		sub := Sub{}
		if err := rows.Scan(
			&sub.SubID.ID,
			&sub.SubID.Vendor,
			&sub.SubID.FeedID,
			&sub.Name,
			&sub.Data,
			&sub.UpdatedAt); err != nil {
			return nil, errors.Wrap(err, "scan")
		}

		subs = append(subs, sub)
	}

	return subs, nil
}