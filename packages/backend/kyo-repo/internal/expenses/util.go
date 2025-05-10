package expenses

func Totals(items []Item) (float64, float64) {
	var common float64
	var uncommon float64

	for _, item := range items {
		if item.Item.Type == "COMMON" {
			common += item.Item.Amount
		} else {
			uncommon += item.Item.Amount
		}
	}

	return common, uncommon
}
