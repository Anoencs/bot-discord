package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

type Investment struct {
	Amount   float64 `json:"amount"`
	Symbol   string  `json:"symbol"`
	BuyPrice float64 `json:"buy_price"`
}

var (
	investments = make(map[string]*Investment)
	investMutex sync.RWMutex
	saveFile    = "investments.json"
)

// Load investments from file
func loadInvestments() error {
	investMutex.Lock()
	defer investMutex.Unlock()

	data, err := os.ReadFile(saveFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	return json.Unmarshal(data, &investments)
}

// Save investments to file
func saveInvestments() error {
	investMutex.RLock()
	defer investMutex.RUnlock()

	data, err := json.MarshalIndent(investments, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(saveFile, data, 0644)
}

func handleSetInvestCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	symbol := strings.ToLower(options[0].StringValue())
	amount := options[1].FloatValue()
	buyPrice := options[2].FloatValue()

	// Extract geckoID from symbol if it contains parentheses
	// Example: "WLD (worldcoin-wld)" -> "worldcoin-wld"
	geckoID := symbol
	if strings.Contains(symbol, "(") {
		parts := strings.Split(symbol, "(")
		geckoID = strings.TrimSpace(strings.TrimRight(parts[1], ")"))
	}

	investMutex.Lock()
	investments[geckoID] = &Investment{
		Amount:   amount,
		Symbol:   geckoID,
		BuyPrice: buyPrice,
	}
	investMutex.Unlock()

	if err := saveInvestments(); err != nil {
		log.Printf("Error saving investments: %v", err)
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Investment set: %.4f %s at $%.2f per coin",
				amount, strings.ToUpper(geckoID), buyPrice),
		},
	})
}

func handleAssetsCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	investMutex.RLock()
	if len(investments) == 0 {
		investMutex.RUnlock()
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "No investments set. Use /setinvest first.",
			},
		})
		return
	}

	var totalValue, totalCost float64
	var fields []*discordgo.MessageEmbedField

	for _, inv := range investments {
		price, err := getCryptoPrice(inv.Symbol)
		if err != nil {
			log.Printf("Error getting price for %s: %v", inv.Symbol, err)
			continue
		}

		// Calculate values
		currentValue := price.Price * inv.Amount
		initialCost := inv.BuyPrice * inv.Amount
		profitLoss := currentValue - initialCost
		profitLossPercent := (profitLoss / initialCost) * 100

		// Add to totals
		totalValue += currentValue
		totalCost += initialCost

		// Create field with profit/loss info
		fields = append(fields, &discordgo.MessageEmbedField{
			Name: strings.ToUpper(inv.Symbol),
			Value: fmt.Sprintf(
				"Amount: %.4f\n"+
					"Buy Price: $%.2f\n"+
					"Current Price: $%.2f\n"+
					"Value: $%.2f\n"+
					"P/L: $%.2f (%.2f%%)",
				inv.Amount,
				inv.BuyPrice,
				price.Price,
				currentValue,
				profitLoss,
				profitLossPercent,
			),
			Inline: true,
		})
	}
	investMutex.RUnlock()

	// Calculate total profit/loss
	totalProfitLoss := totalValue - totalCost
	totalProfitLossPercent := (totalProfitLoss / totalCost) * 100

	// Add summary field
	fields = append(fields, &discordgo.MessageEmbedField{
		Name: "Portfolio Summary",
		Value: fmt.Sprintf(
			"Total Cost: $%.2f\n"+
				"Total Value: $%.2f\n"+
				"Total P/L: $%.2f (%.2f%%)",
			totalCost,
			totalValue,
			totalProfitLoss,
			totalProfitLossPercent,
		),
		Inline: false,
	})

	// Set embed color based on total profit/loss
	embedColor := 0xFF0000 // Red for loss
	if totalProfitLoss >= 0 {
		embedColor = 0x00FF00 // Green for profit
	}

	embed := &discordgo.MessageEmbed{
		Title:  "Current Assets Value",
		Color:  embedColor,
		Fields: fields,
		Footer: &discordgo.MessageEmbedFooter{
			Text: time.Now().Format("2006-01-02 15:04:05 MST"),
		},
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}

func handleRemoveInvestCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	symbol := strings.ToLower(options[0].StringValue())

	investMutex.Lock()
	exists := false
	if _, exists = investments[symbol]; exists {
		delete(investments, symbol)
	}
	investMutex.Unlock()

	if exists {
		// Save after removing
		if err := saveInvestments(); err != nil {
			log.Printf("Error saving investments: %v", err)
		}

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("Removed investment in %s", strings.ToUpper(symbol)),
			},
		})
	} else {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "No investment found for this cryptocurrency",
			},
		})
	}
}
