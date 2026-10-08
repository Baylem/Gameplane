package notify

import (
	"context"
	"time"

	"github.com/GameplanePanel/gameplane/api/internal/kube"
)

// RunWithRegistry keeps credentials central while watching remote workloads in a
// standalone panel. Removing or replacing a client cancels its old informers.
func (n *Notifier) RunWithRegistry(ctx context.Context, reg *kube.Registry) {
	if !n.k.IsStandalone() {
		n.Run(ctx)
		return
	}
	go n.watchRegistry(ctx, reg)
	for {
		select {
		case <-ctx.Done():
			return
		case e := <-n.ch:
			n.fanOut(ctx, e)
		}
	}
}

func (n *Notifier) watchRegistry(ctx context.Context, reg *kube.Registry) {
	type observer struct {
		client *kube.Client
		cancel context.CancelFunc
	}
	watchers := map[string]observer{}
	defer func() {
		for _, w := range watchers {
			w.cancel()
		}
	}()
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		clients := map[string]*kube.Client{}
		for _, id := range reg.IDs() {
			if k, ok := reg.Get(id); ok && k != nil {
				clients[id] = k
			}
		}
		for id, w := range watchers {
			if clients[id] != w.client {
				w.cancel()
				delete(watchers, id)
			}
		}
		for id, k := range clients {
			if _, ok := watchers[id]; ok || k.Dynamic == nil {
				continue
			}
			watchCtx, cancel := context.WithCancel(ctx)
			watchers[id] = observer{client: k, cancel: cancel}
			go n.runClusterWatchers(watchCtx, k, id)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
