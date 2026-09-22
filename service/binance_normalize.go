package service

import (
	"sort"
)

// NormalizeBinanceEvents 去重并合并同一订单的多个 fill。
func NormalizeBinanceEvents(events []BinanceEvent) ([]BinanceEvent, []BinanceDiagnostic) {
	seen := make(map[string]bool)
	groups := make(map[string]BinanceEvent)
	var diagnostics []BinanceDiagnostic
	for _, event := range events {
		if event.EventID != "" && seen[event.EventID] {
			continue
		}
		if event.EventID != "" {
			seen[event.EventID] = true
		}
		key := event.EventType + "|" + event.OrderID
		if event.EventType != "spot_trade" || event.OrderID == "" {
			groups[event.EventID+"|"+event.Symbol] = event
			continue
		}
		if current, ok := groups[key]; ok {
			current.Quantity = current.Quantity.Add(event.Quantity)
			current.QuoteQuantity = current.QuoteQuantity.Add(event.QuoteQuantity)
			current.Fee = current.Fee.Add(event.Fee)
			if current.FeeAsset == "" {
				current.FeeAsset = event.FeeAsset
			}
			if event.FeeAsset != "" && current.FeeAsset != event.FeeAsset {
				diagnostics = append(diagnostics, BinanceDiagnostic{Source: event.Source, Reason: "同一订单 fill 使用了多个手续费币种，需人工复核", Raw: rawEvent(event)})
			}
			groups[key] = current
		} else {
			event.EventID = "order:" + event.OrderID
			groups[key] = event
		}
	}
	out := make([]BinanceEvent, 0, len(groups))
	for _, event := range groups {
		out = append(out, event)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Time.Before(out[j].Time) })
	return out, diagnostics
}
