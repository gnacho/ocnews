// Package scheduler: daemon que refresa los feeds vencidos y ejecuta la
// retención nocturna. Cada ciclo: feeds con next_update <= ahora, con
// concurrencia acotada; el intervalo de cada feed lo decide el refresher
// (adaptativo por novedad + backoff por error, con jitter).
package scheduler

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/gnacho/ocnews/backend/internal/favicon"
	"github.com/gnacho/ocnews/backend/internal/refresher"
	"github.com/gnacho/ocnews/backend/internal/store"
)

type Scheduler struct {
	store       *store.Store
	refresher   *refresher.Refresher
	favicons    *favicon.Cache
	log         *slog.Logger
	tick        time.Duration // periodo de comprobación
	concurrency int           // feeds en paralelo
	retention   time.Duration // retención de items leídos; 0 = infinita
}

func New(st *store.Store, r *refresher.Refresher, fc *favicon.Cache, log *slog.Logger,
	tick time.Duration, concurrency int, retention time.Duration) *Scheduler {
	if concurrency < 1 {
		concurrency = 4
	}
	if tick <= 0 {
		tick = 30 * time.Second
	}
	return &Scheduler{store: st, refresher: r, favicons: fc, log: log,
		tick: tick, concurrency: concurrency, retention: retention}
}

// Run bloquea hasta que ctx se cancela. Toda goroutine interna es hija del
// ctx y se drena con el WaitGroup antes de volver.
func (s *Scheduler) Run(ctx context.Context) {
	var wg sync.WaitGroup
	defer wg.Wait()

	ticker := time.NewTicker(s.tick)
	defer ticker.Stop()

	// retención: al arrancar y cada 24h
	retentionCtx, retentionCancel := context.WithCancel(ctx)
	defer retentionCancel()
	wg.Add(1)
	go func() {
		defer wg.Done()
		s.runRetention(retentionCtx)
	}()

	s.cycle(ctx) // barrido inmediato al arrancar
	for {
		select {
		case <-ctx.Done():
			s.log.Info("scheduler detenido")
			return
		case <-ticker.C:
			s.cycle(ctx)
		}
	}
}

// cycle refresca los feeds vencidos con concurrencia acotada.
func (s *Scheduler) cycle(ctx context.Context) {
	due, err := s.store.ListDueFeeds(time.Now().Unix(), 50)
	if err != nil {
		s.log.Error("listar feeds vencidos", "err", err)
		return
	}
	if len(due) == 0 {
		return
	}
	s.log.Debug("ciclo de refresco", "feeds", len(due))

	sem := make(chan struct{}, s.concurrency)
	var wg sync.WaitGroup
	for i := range due {
		f := due[i]
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				return
			}
			res := s.refresher.Refresh(ctx, &f)
			// favicon best-effort: solo cuando el feed trae novedades y no
			// está cacheado (un sitio sin favicon no se reintenta cada ciclo)
			if s.favicons != nil && res.Inserted > 0 && !s.favicons.Has(favicon.Hash(f.URL)) {
				s.favicons.Fetch(ctx, f.URL, f.Link)
			}
		}()
	}
	wg.Wait()
}

// runRetention purga items leídos no destacados. El global (s.retention) aplica
// a todos los feeds, pero los que tienen override propio (retention_days>0) se
// purgan antes (overrides más cortos) y se EXCLUYEN del corte global — un feed
// con override de retención se purga SOLO con su propia regla. 0 = desactivada.
func (s *Scheduler) runRetention(ctx context.Context) {
	if s.retention <= 0 {
		return
	}
	purge := func() {
		now := time.Now()
		// feeds con override propio
		over, err := s.store.FeedsWithRetentionOverride()
		if err != nil {
			s.log.Error("retención por feed: listar overrides", "err", err)
		} else {
			for _, f := range over {
				cutoff := now.Add(-time.Duration(f.Days) * 24 * time.Hour).Unix()
				n, err := s.store.PurgeOldItemsByFeed(f.ID, cutoff)
				if err != nil {
					s.log.Error("retención por feed", "feed", f.ID, "err", err)
					continue
				}
				if n > 0 {
					s.log.Info("retención por feed", "feed", f.ID, "días", f.Days, "borrados", n)
				}
			}
		}
		// feeds sin override: corte global
		cutoff := now.Add(-s.retention).Unix()
		n, err := s.store.PurgeOldItems(cutoff)
		if err != nil {
			s.log.Error("retención falló", "err", err)
			return
		}
		if n > 0 {
			s.log.Info("retención", "items borrados", n)
		}
	}
	purge()
	t := time.NewTicker(24 * time.Hour)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			purge()
		}
	}
}
