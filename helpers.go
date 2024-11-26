package main

import (
	"fmt"
	"math"
	"strings"

	"github.com/bwmarrin/discordgo"
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

func calculateTotalAmount(entries []DCAEntry) float64 {
	var total float64
	for _, entry := range entries {
		// Use high-precision addition
		total = math.Round((total+entry.Amount)*100000000) / 100000000
	}
	return total
}

func calculateAverageBuyPrice(entries []DCAEntry) float64 {
	var totalCost, totalAmount float64
	for _, entry := range entries {
		// Calculate with high precision
		amount := math.Round(entry.Amount*100000000) / 100000000
		cost := math.Round(entry.Amount*entry.BuyPrice*100000000) / 100000000
		totalAmount += amount
		totalCost += cost
	}
	if totalAmount == 0 {
		return 0
	}
	// Return average with high precision
	return math.Round((totalCost/totalAmount)*100000000) / 100000000
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// helpers format
func formatCryptoAmount(amount float64) string {
	// For very small numbers (less than 0.00001), use scientific notation
	if amount < 0.00001 && amount > 0 {
		return fmt.Sprintf("%.8e", amount)
	}
	// For regular numbers, use 8 decimal places (standard crypto precision)
	return fmt.Sprintf("%.8f", amount)
}

func formatFiatPrice(price float64) string {
	// For prices under $1, show 4 decimal places
	if price < 1 {
		return fmt.Sprintf("$%.4f", price)
	}
	// For prices $1 and above, show 2 decimal places
	return fmt.Sprintf("$%.2f", price)
}

func formatFiatAmount(amount float64) string {
	// For amounts less than 1 cent, show more precision
	if math.Abs(amount) < 0.01 && amount != 0 {
		return fmt.Sprintf("$%.4f", amount)
	}
	// For regular amounts, show 2 decimal places
	return fmt.Sprintf("$%.2f", amount)
}

func formatPercentage(value float64) string {
	if math.Abs(value) < 0.01 && value != 0 {
		return fmt.Sprintf("%.4f%%", value)
	}
	return fmt.Sprintf("%.2f%%", value)
}

func formatParticipants(s *discordgo.Session, participants []string) string {
	var names []string
	for _, userID := range participants {
		if user, err := s.User(userID); err == nil {
			names = append(names, user.Username)
		} else {
			names = append(names, userID)
		}
	}
	if len(names) == 0 {
		return "No participants"
	}
	return strings.Join(names, ", ")
}
