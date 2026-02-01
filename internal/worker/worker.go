package worker

import (
	"context"
	"strings"
	"time"

	"bookstore/internal/store"
)

type OrderProcessor struct {
	Store     *store.Store
	OrderJobs <-chan int
}

func (p *OrderProcessor) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case orderID := <-p.OrderJobs:
			time.Sleep(300 * time.Millisecond)
			o, items, err := p.Store.MarkOrderCompleted(orderID)
			if err != nil {
				continue
			}
			for _, it := range items {
				f, err := p.Store.GetFormat(it.FormatID)
				if err != nil {
					continue
				}
				if strings.EqualFold(f.Type, "Digital") || strings.EqualFold(f.Type, "Audio") {
					_, _ = p.Store.GrantDigitalAccess(o.UserID, it.FormatID)
				}
			}
		}
	}
}
