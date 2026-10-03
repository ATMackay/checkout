package pricing

import "github.com/shopspring/decimal"

// Rule is a promotion. A rule sees the requested basket and returns a discount,
// items to give away, or both. Free items are capped by stock in Quote, not by
// the rule.
type Rule interface {
	rewards(requested Inventory) []string
	apply(b basket) effect
}

type basket struct {
	lines []Line
}

// find returns the requested line for the item called name.
func (b basket) find(name string) (Line, bool) {
	for _, l := range b.lines {
		if l.Item.Name == name {
			return l, true
		}
	}
	return Line{}, false
}

type grant struct {
	name     string
	quantity int
}

type effect struct {
	deduction decimal.Decimal
	grants    []grant
}

// FreeItem gives one reward item for every unit of trigger bought.
func FreeItem(trigger, reward string) Rule {
	return freeItem{trigger: trigger, reward: reward}
}

type freeItem struct {
	trigger, reward string
}

func (r freeItem) rewards(requested Inventory) []string {
	if _, ok := requested.byName(r.trigger); !ok {
		return nil
	}
	return []string{r.reward}
}

func (r freeItem) apply(b basket) effect {
	l, ok := b.find(r.trigger)
	if !ok {
		return effect{deduction: decimal.Zero}
	}
	return effect{deduction: decimal.Zero, grants: []grant{{name: r.reward, quantity: l.Quantity}}}
}

// BuyXPayY charges for pay units out of every complete group of buy units of
// item: BuyXPayY("Google TV", 3, 2) is "3 for the price of 2". It applies only
// when 0 <= pay < buy.
func BuyXPayY(item string, buy, pay int) Rule {
	return buyXPayY{item: item, buy: buy, pay: pay}
}

type buyXPayY struct {
	item     string
	buy, pay int
}

func (r buyXPayY) rewards(Inventory) []string { return nil }

func (r buyXPayY) apply(b basket) effect {
	l, ok := b.find(r.item)
	if !ok || r.buy <= 0 || r.pay < 0 || r.pay >= r.buy {
		return effect{deduction: decimal.Zero}
	}
	freeUnits := (l.Quantity / r.buy) * (r.buy - r.pay)
	return effect{deduction: l.Item.Price.Mul(decimal.NewFromInt(int64(freeUnits)))}
}

// BulkDiscount takes percent (0 < percent <= 100) off every unit of item when
// more than over units are bought.
func BulkDiscount(item string, over int, percent decimal.Decimal) Rule {
	return bulkDiscount{item: item, over: over, percent: percent}
}

type bulkDiscount struct {
	item    string
	over    int
	percent decimal.Decimal
}

func (r bulkDiscount) rewards(Inventory) []string { return nil }

func (r bulkDiscount) apply(b basket) effect {
	l, ok := b.find(r.item)
	hundred := decimal.NewFromInt(100)
	if !ok || l.Quantity <= r.over || !r.percent.IsPositive() || r.percent.GreaterThan(hundred) {
		return effect{deduction: decimal.Zero}
	}
	unitsPrice := l.Item.Price.Mul(decimal.NewFromInt(int64(l.Quantity)))
	return effect{deduction: unitsPrice.Mul(r.percent).Div(hundred).Round(2)}
}
