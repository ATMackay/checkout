// Package pricing turns a basket of SKUs and a snapshot of inventory into a
// Quote: what the customer pays, what promotions give away, and what leaves
// stock. It does no I/O, so the same basket and snapshot always give the same
// quote, and pricing a basket and purchasing it cannot disagree.
package pricing

import (
	"fmt"
	"maps"
	"slices"

	"github.com/ATMackay/checkout/errors"
	"github.com/ATMackay/checkout/model"
	"github.com/shopspring/decimal"
)

// MaxBasketSize is the most units a single basket may request.
const MaxBasketSize = 100

var (
	// ErrEmptyBasket is returned when no SKUs are requested.
	ErrEmptyBasket = fmt.Errorf("%w: no items requested", errors.ErrInvalidInput)
	// ErrBasketTooLarge is returned when more than MaxBasketSize units are requested.
	ErrBasketTooLarge = fmt.Errorf("%w: too many items requested", errors.ErrInvalidInput)
	// ErrMalformedSKU is returned for a SKU that is not a valid SKU string.
	ErrMalformedSKU = fmt.Errorf("%w: malformed sku", errors.ErrInvalidInput)
	// ErrUnknownItem is returned for a well-formed SKU missing from inventory.
	ErrUnknownItem = fmt.Errorf("%w: unknown item", errors.ErrNotFound)
	// ErrOutOfStock is returned when more units are requested than are in stock.
	ErrOutOfStock = fmt.Errorf("%w: insufficient stock", errors.ErrNotFound)
)

// Inventory is a point-in-time view of stock, keyed by SKU.
type Inventory map[string]model.Item

// NewInventory builds an Inventory from store items. Nil and zero-value items
// (a store lookup that found nothing) are skipped.
func NewInventory(items ...*model.Item) Inventory {
	inv := make(Inventory, len(items))
	for _, it := range items {
		if it != nil && it.SKU != "" {
			inv[it.SKU] = *it
		}
	}
	return inv
}

func (inv Inventory) byName(name string) (model.Item, bool) {
	for _, it := range inv {
		if it.Name == name {
			return it, true
		}
	}
	return model.Item{}, false
}

// Line is a quantity of one item.
type Line struct {
	Item     model.Item
	Quantity int
}

// Quote is the priced outcome of a basket.
type Quote struct {
	// Lines are the requested items, in the order each SKU was first requested.
	Lines []Line
	// Free are the items promotions give away, already capped by stock.
	Free []Line
	// Gross is the price of the requested items before promotions.
	Gross decimal.Decimal
	// Deduction is the total promotional discount.
	Deduction decimal.Decimal
}

// Total is what the customer pays.
func (q Quote) Total() decimal.Decimal {
	return q.Gross.Sub(q.Deduction)
}

// SKUs lists one entry per unit leaving stock: requested items, then free ones.
func (q Quote) SKUs() []string {
	var skus []string
	for _, l := range slices.Concat(q.Lines, q.Free) {
		for range l.Quantity {
			skus = append(skus, l.Item.SKU)
		}
	}
	return skus
}

// StockAfter returns each affected item with its inventory reduced by every
// unit the quote takes, requested and free, ordered by SKU.
func (q Quote) StockAfter() []model.Item {
	after := make(map[string]model.Item)
	for _, l := range slices.Concat(q.Lines, q.Free) {
		it, ok := after[l.Item.SKU]
		if !ok {
			it = l.Item
		}
		it.InventoryQuantity -= l.Quantity
		after[l.Item.SKU] = it
	}
	items := make([]model.Item, 0, len(after))
	for _, sku := range slices.Sorted(maps.Keys(after)) {
		items = append(items, after[sku])
	}
	return items
}

// CheckSKUs reports whether skus is a non-empty list of at most MaxBasketSize
// well-formed SKUs. Quote
// applies the same check; callers use it to reject a request before fetching
// inventory for it.
func CheckSKUs(skus []string) error {
	if len(skus) == 0 {
		return ErrEmptyBasket
	}
	if len(skus) > MaxBasketSize {
		return fmt.Errorf("%w: %d > %d", ErrBasketTooLarge, len(skus), MaxBasketSize)
	}
	for _, sku := range skus {
		if !model.IsSKU(sku) {
			return fmt.Errorf("%w '%s'", ErrMalformedSKU, sku)
		}
	}
	return nil
}

// Pricer prices baskets under a fixed set of promotion rules.
type Pricer struct {
	rules []Rule
}

// NewPricer returns a Pricer applying rules in the given order.
func NewPricer(rules ...Rule) *Pricer {
	return &Pricer{rules: slices.Clone(rules)}
}

// RewardNames lists the items the rules may give away for a basket drawn from
// requested. Add them to the Inventory passed to Quote; a reward missing from it
// is simply not granted.
func (p *Pricer) RewardNames(requested Inventory) []string {
	var names []string
	for _, r := range p.rules {
		names = append(names, r.rewards(requested)...)
	}
	slices.Sort(names)
	return slices.Compact(names)
}

// Quote prices skus against inv. Every SKU must be in inv with enough stock for
// all its requested units. Free items are granted only from stock left after
// the requested units, so a free item never makes a purchase fail.
func (p *Pricer) Quote(skus []string, inv Inventory) (Quote, error) {
	lines, err := basketLines(skus, inv)
	if err != nil {
		return Quote{}, err
	}
	q := Quote{Lines: lines, Gross: gross(lines), Deduction: decimal.Zero}
	remaining := remainingStock(lines, inv)
	b := basket{lines: lines}
	for _, r := range p.rules {
		e := r.apply(b)
		q.Deduction = q.Deduction.Add(e.deduction)
		for _, g := range e.grants {
			if line, ok := grantFrom(g, inv, remaining); ok {
				q.Free = append(q.Free, line)
			}
		}
	}
	// Stacked rules on one item must never make the basket pay the customer.
	q.Deduction = decimal.Min(q.Deduction, q.Gross)
	return q, nil
}

func basketLines(skus []string, inv Inventory) ([]Line, error) {
	if err := CheckSKUs(skus); err != nil {
		return nil, err
	}
	var order []string
	counts := make(map[string]int)
	for _, sku := range skus {
		if counts[sku] == 0 {
			order = append(order, sku)
		}
		counts[sku]++
	}
	lines := make([]Line, 0, len(order))
	for _, sku := range order {
		it, ok := inv[sku]
		if !ok {
			return nil, fmt.Errorf("%w: sku %s", ErrUnknownItem, sku)
		}
		if counts[sku] > it.InventoryQuantity {
			return nil, fmt.Errorf("%w: sku %s, requested %d, in stock %d", ErrOutOfStock, sku, counts[sku], it.InventoryQuantity)
		}
		lines = append(lines, Line{Item: it, Quantity: counts[sku]})
	}
	return lines, nil
}

func gross(lines []Line) decimal.Decimal {
	total := decimal.Zero
	for _, l := range lines {
		total = total.Add(l.Item.Price.Mul(decimal.NewFromInt(int64(l.Quantity))))
	}
	return total
}

// remainingStock is the stock of every inventory item after the requested
// units are taken; free items are granted from it.
func remainingStock(lines []Line, inv Inventory) map[string]int {
	remaining := make(map[string]int, len(inv))
	for sku, it := range inv {
		remaining[sku] = it.InventoryQuantity
	}
	for _, l := range lines {
		remaining[l.Item.SKU] -= l.Quantity
	}
	return remaining
}

// grantFrom resolves a grant against inventory, capped by remaining stock,
// and takes the granted units out of remaining.
func grantFrom(g grant, inv Inventory, remaining map[string]int) (Line, bool) {
	it, ok := inv.byName(g.name)
	if !ok {
		return Line{}, false
	}
	n := min(g.quantity, remaining[it.SKU])
	if n <= 0 {
		return Line{}, false
	}
	remaining[it.SKU] -= n
	return Line{Item: it, Quantity: n}, true
}
