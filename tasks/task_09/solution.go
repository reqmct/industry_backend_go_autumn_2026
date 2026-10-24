package main

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
)

type Result[R any] struct {
	Value R
	Err   error
}

var ErrInvalidWorkers = errors.New("workers must be positive")

func ParallelMap[T any, R any](ctx context.Context, workers int, in []T, fn func(context.Context, T) (R, error)) ([]Result[R], error) {
	if workers < 1 {
		return nil, ErrInvalidWorkers
	}

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	if len(in) == 0 {
		return []Result[R]{}, nil
	}

	if workers > len(in) {
		workers = len(in)
	}

	var (
		results = make([]Result[R], len(in))
		counter atomic.Int64
		wg      sync.WaitGroup
	)

	wg.Add(workers)

	for range workers {
		go func() {
			defer wg.Done()

			for {
				i := counter.Add(1) - 1
				if ctx.Err() != nil || int(i) >= len(in) {
					return
				}

				value, err := fn(ctx, in[i])
				if err != nil {
					results[i].Err = err
				} else {
					results[i].Value = value
				}
			}
		}()
	}

	wg.Wait()

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	return results, nil
}
