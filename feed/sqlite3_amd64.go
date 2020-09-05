package feed

import (
	"context"

	"github.com/doug-martin/goqu/v9"
)

func selectSubs(ctx context.Context, sd *goqu.SelectDataset) ([]Sub, error) {
	subs := make([]Sub, 0)
	return subs, sd.Select(Sub{}).ScanStructsContext(ctx, &subs)
}
