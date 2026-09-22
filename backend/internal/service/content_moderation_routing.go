package service

import (
	"context"
	"sort"

	"github.com/shopspring/decimal"
)

type moderationPricedRoute struct {
	id   string
	cost decimal.Decimal
}

// Compare the complete rendered request and maximum configured output, with no
// assumed cache hit. Identical model IDs at different channels remain independent.
func moderationChannelOrder(ctx context.Context, cfg *ModerationV2Config, text string) ([]string, string) {
	if cfg.Routing != "lowest_cost" {
		return append([]string{cfg.PrimaryID}, cfg.FallbackIDs...), ""
	}
	candidates := []moderationPricedRoute{}
	reason := "no_priced_channel"
	for _, p := range cfg.Providers {
		if !p.Enabled || !p.AuditValidated || len(p.APIKeys) == 0 || p.Prices.Input == "" || p.Prices.Output == "" || !p.OutputLimitVerified {
			continue
		}
		_, estimate, e := buildModerationV2Payload(ctx, p, text)
		if e != nil {
			reason = "invalid_payload"
			if estimate > p.MaxInputTokens {
				reason = "input_budget_exceeded"
			}
			continue
		}
		prices := p.Prices
		regular, _ := decimalValue(prices.Input)
		cached, _ := decimalValue(prices.CachedInput)
		if cached.GreaterThan(regular) {
			prices.Input = prices.CachedInput
		}
		cost, _ := decimal.NewFromString(moderationV2Cost(prices, ModerationV2Usage{Input: int64(estimate), Output: int64(p.MaxOutputTokens)}))
		candidates = append(candidates, moderationPricedRoute{id: p.ID, cost: cost})
	}
	// Configuration order breaks equal-price ties, making the route predictable.
	sort.SliceStable(candidates, func(i, j int) bool { return candidates[i].cost.LessThan(candidates[j].cost) })
	ids := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		ids = append(ids, candidate.id)
	}
	if len(ids) > 0 {
		reason = ""
	}
	return ids, reason
}
