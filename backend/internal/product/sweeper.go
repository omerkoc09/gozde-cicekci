package product

import (
	"context"
	"fmt"
	"log"
	"time"
)

// Sweeper ödemesi tamamlanmayan siparişlerin stok rezervasyonlarını serbest
// bırakır. Yarım kalan ödeme stoğu sonsuza kadar tutmamalı (spec §4.3).
type Sweeper struct {
	store    *Store
	ttl      time.Duration
	interval time.Duration
}

func NewSweeper(store *Store, ttl, interval time.Duration) *Sweeper {
	return &Sweeper{store: store, ttl: ttl, interval: interval}
}

// Run ctx iptal edilene kadar periyodik olarak süpürür.
func (s *Sweeper) Run(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := s.SweepOnce(ctx); err != nil {
				log.Printf("stok süpürücü hatası: %v", err)
			}
		}
	}
}

// SweepOnce tek tur süpürür, serbest bırakılan sipariş sayısını döner.
//
// stock_swept işareti olmasaydı aynı sipariş her turda tekrar süpürülür ve
// rezerve sayacı yanlış yere düşerdi.
func (s *Sweeper) SweepOnce(ctx context.Context) (int, error) {
	rows, err := s.store.pool.Query(ctx, `
		SELECT o.id, oi.product_id, oi.quantity
		  FROM orders o
		  JOIN order_items oi ON oi.order_id = o.id
		 WHERE o.status = 'awaiting_payment'
		   AND NOT o.stock_swept
		   AND o.created_at < now() - $1::interval
		   AND oi.product_id IS NOT NULL`,
		s.ttl.String())
	if err != nil {
		return 0, fmt.Errorf("süresi geçen siparişler: %w", err)
	}

	type satir struct {
		orderID   int64
		productID int64
		qty       int
	}
	var satirlar []satir
	for rows.Next() {
		var r satir
		if err := rows.Scan(&r.orderID, &r.productID, &r.qty); err != nil {
			rows.Close()
			return 0, fmt.Errorf("süpürme scan: %w", err)
		}
		satirlar = append(satirlar, r)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}

	siparisler := map[int64]bool{}
	for _, r := range satirlar {
		// Tek satırdaki hata tüm süpürmeyi durdurmamalı — logla, devam et.
		// Bir sonraki tur tekrar dener (spec §8).
		if err := s.store.Release(ctx, r.productID, r.qty, &r.orderID); err != nil {
			log.Printf("rezervasyon serbest bırakılamadı (order=%d, product=%d): %v",
				r.orderID, r.productID, err)
			continue
		}
		siparisler[r.orderID] = true
	}

	for orderID := range siparisler {
		if _, err := s.store.pool.Exec(ctx,
			`UPDATE orders SET stock_swept = true WHERE id = $1`, orderID); err != nil {
			log.Printf("süpürme işareti yazılamadı (order=%d): %v", orderID, err)
		}
	}

	return len(siparisler), nil
}
