export const formatTutar = (v: string) => {
  const n = Number.parseFloat(v)

  return Number.isNaN(n)
    ? v
    : `${n.toLocaleString('tr-TR', { minimumFractionDigits: 2 })} ₺`
}
