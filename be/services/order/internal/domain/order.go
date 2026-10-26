package domain

type Line struct {
	SKU            string
	Quantity       int64
	UnitPriceCents int64
	LineCents      int64
}

type Order struct {
	ID           string
	Code         string
	Status       string
	CreatedBy    string
	DecidedBy    string
	RejectReason string
	TotalCents   int64
	CurrencyCode string
	Lines        []Line
}

func (o Order) AllowedActions(userID string, canApprove bool) []string {
	if o.Status == "PENDING_APPROVAL" && canApprove && o.CreatedBy != userID {
		return []string{"APPROVE", "REJECT"}
	}
	return []string{}
}
