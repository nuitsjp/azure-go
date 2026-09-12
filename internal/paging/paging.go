// Package paging collects Azure SDK-compatible pagers without owning retries.
package paging

import "context"

type Pager[P any] interface {
	More() bool
	NextPage(context.Context) (P, error)
}

// Collect returns a non-nil slice on success, including an empty result.
// On failure it returns nil, never a partial list masquerading as a complete one.
func Collect[P, T any](ctx context.Context, pager Pager[P], values func(P) []*T) ([]*T, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	result := make([]*T, 0)
	for pager.More() {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		result = append(result, values(page)...)
	}
	return result, nil
}
