package main

import (
	"fmt"
	"strings"
)

func convertToCMCID(geckoID string) string {
	if id, ok := convertToCMCSymbolMap[geckoID]; ok {
		return id
	}
	return geckoID
}

func convertToBinanceSymbol(geckoID string) string {
	if symbol, ok := convertToBinanceSymbolMap[geckoID]; ok {
		return symbol
	}
	return ""
}

func formatNickname(symbol string, price float64) string {
	cryptoInfo, exists := commonCryptos[symbol]
	if !exists {
		return fmt.Sprintf("%s $%.2f", strings.ToUpper(symbol), price)
	}

	if cryptoInfo.Symbol == "BTC" {
		return fmt.Sprintf("BTC $%.0f", price)
	}
	return fmt.Sprintf("%s $%.2f", cryptoInfo.Symbol, price)
}

func formatPrice(symbol string, price float64) string {
	cryptoInfo, exists := commonCryptos[symbol]
	if !exists {
		return fmt.Sprintf("$%.2f", price)
	}

	if cryptoInfo.Symbol == "BTC" {
		return fmt.Sprintf("$%.0f", price)
	}
	return fmt.Sprintf("$%.2f", price)
}
func getColorForChange(change float64) int {
	if change > 0 {
		return 0x00ff00 // Green
	}
	return 0xff0000 // Red
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
