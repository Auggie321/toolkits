package app

type GeminiUsage struct {
	Rows []GeminiUsageRow
}

func (u GeminiUsage) TotalCost() (string, float64) {
	var currency string
	var total float64
	for _, row := range u.Rows {
		if currency == "" {
			currency = row.Currency
		}
		total += row.Cost
	}
	return currency, total
}

type GeminiUsageRow struct {
	Currency    string
	UsageUnit   string
	UsageAmount float64
	Cost        float64
}

type DeepSeekBalance struct {
	IsAvailable bool                  `json:"is_available"`
	BalanceInfo []DeepSeekBalanceInfo `json:"balance_infos"`
}

type DeepSeekBalanceInfo struct {
	Currency        string `json:"currency"`
	TotalBalance    string `json:"total_balance"`
	GrantedBalance  string `json:"granted_balance"`
	ToppedUpBalance string `json:"topped_up_balance"`
}

type CopilotUsage struct {
	Used      float64
	Limit     float64
	Percent   float64
	Source    string
	IsPercent bool
	Items     []CopilotUsageItem
}

type CopilotUsageItem struct {
	Product          string  `json:"product"`
	SKU              string  `json:"sku"`
	Model            string  `json:"model"`
	UnitType         string  `json:"unitType"`
	GrossQuantity    float64 `json:"grossQuantity"`
	DiscountQuantity float64 `json:"discountQuantity"`
	NetQuantity      float64 `json:"netQuantity"`
	NetAmount        float64 `json:"netAmount"`
}
