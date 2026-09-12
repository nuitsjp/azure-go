package paging

import (
	"context"
	"errors"
	"testing"
)

type page struct{ values []*int }
type fakePager struct {
	pages  []page
	index  int
	failAt int
	err    error
	after  func()
}

func (p *fakePager) More() bool { return p.index < len(p.pages) }
func (p *fakePager) NextPage(ctx context.Context) (page, error) {
	if err := ctx.Err(); err != nil {
		return page{}, err
	}
	if p.err != nil && p.index == p.failAt {
		return page{}, p.err
	}
	value := p.pages[p.index]
	p.index++
	if p.after != nil {
		p.after()
	}
	return value, nil
}
func values(p page) []*int { return p.values }
func ip(v int) *int        { return &v }

func TestAllPages(t *testing.T) {
	p := &fakePager{pages: []page{{[]*int{ip(1)}}, {nil}, {[]*int{ip(2), ip(3)}}}}
	got, err := Collect(context.Background(), p, values)
	if err != nil || len(got) != 3 || *got[0] != 1 || *got[2] != 3 || p.index != 3 {
		t.Fatalf("got=%v index=%d err=%v", got, p.index, err)
	}
}
func TestEmptyListIsNotNil(t *testing.T) {
	got, err := Collect(context.Background(), &fakePager{}, values)
	if err != nil || got == nil || len(got) != 0 {
		t.Fatalf("got=%v err=%v", got, err)
	}
}
func TestPartialFailureDiscardsResults(t *testing.T) {
	cause := errors.New("page two failed")
	p := &fakePager{pages: []page{{[]*int{ip(1)}}, {}}, failAt: 1, err: cause}
	got, err := Collect(context.Background(), p, values)
	if got != nil || !errors.Is(err, cause) {
		t.Fatalf("got=%v err=%v", got, err)
	}
}
func TestCanceledBeforeFirstPage(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	p := &fakePager{pages: []page{{}}}
	got, err := Collect(ctx, p, values)
	if got != nil || !errors.Is(err, context.Canceled) || p.index != 0 {
		t.Fatalf("got=%v err=%v", got, err)
	}
}
func TestCanceledBetweenPages(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	p := &fakePager{pages: []page{{[]*int{ip(1)}}, {}}, after: cancel}
	got, err := Collect(ctx, p, values)
	if got != nil || !errors.Is(err, context.Canceled) || p.index != 1 {
		t.Fatalf("got=%v err=%v", got, err)
	}
}
