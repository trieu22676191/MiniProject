package domain

type Item struct {
	SKU            string
	Name           string
	UnitPriceCents int64
	CurrencyCode   string
	OnHand         int64
	Reserved       int64
}

func (i Item) Available() int64 {
	return i.OnHand - i.Reserved
}
