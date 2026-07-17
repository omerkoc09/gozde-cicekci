package order

import (
	"fmt"
	"time"
)

// FormatOrderNo müşteriye söylenen sipariş numarasını üretir: "GGAA-NNNN".
//
// Gün+ay, tire, o günün sıra numarası. Örnek: 26 Temmuz'un 42. siparişi →
// "2607-0042". Esnaf "bugünün 42. siparişi" diye okuyabilir.
//
// Neden id değil: id kaç sipariş alındığını dışarı sızdırır (spec §3).
func FormatOrderNo(t time.Time, seq int) string {
	return fmt.Sprintf("%02d%02d-%04d", t.Day(), int(t.Month()), seq)
}
