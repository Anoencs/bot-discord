package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

var (
	portfolios = make(map[string]*Portfolio)
	portMutex  sync.RWMutex
	saveFile   = "portfolios.json"
)

type Investment struct {
	Symbol       string     `json:"symbol"`
	Type         string     `json:"type"`                   // "personal" or "collective"
	Participants []string   `json:"participants,omitempty"` // List of user IDs for collective investments
	Entries      []DCAEntry `json:"entries"`
	CreatedBy    string     `json:"created_by"`
	CreatedAt    time.Time  `json:"created_at"`
}

type DCAEntry struct {
	Amount    float64   `json:"amount"`
	BuyPrice  float64   `json:"buy_price"`
	EntryDate time.Time `json:"entry_date"`
}

type Portfolio struct {
	UserID      string                 `json:"user_id"`
	Investments map[string]*Investment `json:"investments"`
}

// Load portfolios from file
func loadPortfolios() error {
	portMutex.Lock()
	defer portMutex.Unlock()

	data, err := os.ReadFile(saveFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	return json.Unmarshal(data, &portfolios)
}

// Save portfolios to file
func savePortfolios() error {
	portMutex.RLock()
	defer portMutex.RUnlock()

	data, err := json.MarshalIndent(portfolios, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(saveFile, data, 0644)
}

// Hàm helper để lấy hoặc tạo portfolio cho user
func getOrCreatePortfolio(userID string) *Portfolio {
	portMutex.Lock()
	defer portMutex.Unlock()

	if portfolio, exists := portfolios[userID]; exists {
		return portfolio
	}

	// Tạo portfolio mới nếu chưa tồn tại
	portfolios[userID] = &Portfolio{
		UserID:      userID,
		Investments: make(map[string]*Investment),
	}
	return portfolios[userID]
}
func handleSetInvestCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	symbol := strings.ToLower(options[0].StringValue())
	amount := options[1].FloatValue()
	buyPrice := options[2].FloatValue()

	// Default values
	investType := "personal"
	var participants []string

	// Process options
	for _, opt := range options[3:] {
		switch opt.Name {
		case "type":
			investType = strings.ToLower(opt.StringValue())
		case "participants":
			mentionsStr := opt.StringValue()
			mentions := strings.Fields(mentionsStr)
			for _, mention := range mentions {
				userID := strings.Trim(mention, "<@!> ")
				if userID != "" {
					_, err := s.User(userID)
					if err == nil {
						participants = append(participants, userID)
					}
				}
			}
		}
	}

	userID := i.Member.User.ID
	geckoID := symbol
	if strings.Contains(symbol, "(") {
		parts := strings.Split(symbol, "(")
		geckoID = strings.TrimSpace(strings.TrimRight(parts[1], ")"))
	}

	portfolio := getOrCreatePortfolio(userID)

	// Create new DCA entry
	newEntry := DCAEntry{
		Amount:    amount,
		BuyPrice:  buyPrice,
		EntryDate: time.Now(),
	}

	// Check if investment already exists
	if existing, exists := portfolio.Investments[geckoID]; exists {
		// Add new entry to existing investment
		existing.Entries = append(existing.Entries, newEntry)
	} else {
		// Create new investment with first entry
		investment := &Investment{
			Symbol:    geckoID,
			Type:      investType,
			CreatedBy: userID,
			CreatedAt: time.Now(),
			Entries:   []DCAEntry{newEntry},
		}

		if investType == "collective" {
			investment.Participants = participants
		}

		portfolio.Investments[geckoID] = investment
	}

	// Handle collective investments
	if investType == "collective" {
		for _, participantID := range participants {
			if participantID != userID {
				partPortfolio := getOrCreatePortfolio(participantID)
				partPortfolio.Investments[geckoID] = portfolio.Investments[geckoID]
			}
		}
	}

	if err := savePortfolios(); err != nil {
		log.Printf("Error saving portfolios: %v", err)
	}

	// Create response message with DCA information
	investment := portfolio.Investments[geckoID]
	avgBuyPrice := calculateAverageBuyPrice(investment.Entries)
	totalAmount := calculateTotalAmount(investment.Entries)

	var response string
	if investType == "collective" {
		participantsStr := formatParticipants(s, participants)
		response = fmt.Sprintf("Added to collective investment: %.4f %s at $%.2f per coin\n"+
			"Total Amount: %.4f\nAverage Buy Price: $%.2f\nParticipants: %s",
			amount, strings.ToUpper(geckoID), buyPrice, totalAmount, avgBuyPrice, participantsStr)
	} else {
		response = fmt.Sprintf("Added to personal investment: %.4f %s at $%.2f per coin\n"+
			"Total Amount: %.4f\nAverage Buy Price: $%.2f",
			amount, strings.ToUpper(geckoID), buyPrice, totalAmount, avgBuyPrice)
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: response,
			Flags: func() discordgo.MessageFlags {
				if investType == "personal" {
					return discordgo.MessageFlagsEphemeral
				}
				return 0
			}(),
		},
	})
}
func handleAssetsCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	requestingUserID := i.Member.User.ID

	var filterType string
	if len(i.ApplicationCommandData().Options) > 0 {
		filterType = strings.ToLower(i.ApplicationCommandData().Options[0].StringValue())
	}

	portMutex.RLock()
	defer portMutex.RUnlock()

	// Get all visible collective investments
	visibleCollectiveInvestments := make(map[string]*Investment)
	for _, p := range portfolios {
		for symbol, inv := range p.Investments {
			if inv.Type == "collective" {
				isParticipant := false
				for _, participantID := range inv.Participants {
					if participantID == requestingUserID {
						isParticipant = true
						break
					}
				}
				if isParticipant || inv.CreatedBy == requestingUserID {
					visibleCollectiveInvestments[symbol] = inv
				}
			}
		}
	}

	// Get personal investments of the requesting user
	personalInvestments := make(map[string]*Investment)
	if portfolio, exists := portfolios[requestingUserID]; exists {
		for symbol, inv := range portfolio.Investments {
			if inv.Type == "personal" && inv.CreatedBy == requestingUserID {
				personalInvestments[symbol] = inv
			}
		}
	}

	switch filterType {
	case "personal":
		// Handle personal filter...
		if len(personalInvestments) > 0 {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Flags:  discordgo.MessageFlagsEphemeral,
					Embeds: []*discordgo.MessageEmbed{createPortfolioEmbed(s, personalInvestments, "Personal", i.Member.User.Username)},
				},
			})
		} else {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "You have no personal investments.",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
		}

	case "collective":
		// Handle collective filter...
		if len(visibleCollectiveInvestments) > 0 {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Embeds: []*discordgo.MessageEmbed{createPortfolioEmbed(s, visibleCollectiveInvestments, "Collective", i.Member.User.Username)},
				},
			})
		} else {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "You have no collective investments.",
				},
			})
		}

	default:
		completePortfolio := make(map[string]*Investment)
		// Add all personal investments
		for k, v := range personalInvestments {
			completePortfolio[k] = v
		}
		// Add all collective investments
		for k, v := range visibleCollectiveInvestments {
			completePortfolio[k] = v
		}

		if len(completePortfolio) == 0 {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "You don't have any investments yet.",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}

		if len(visibleCollectiveInvestments) > 0 {
			err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Embeds: []*discordgo.MessageEmbed{createPortfolioEmbed(s, visibleCollectiveInvestments, "Collective", i.Member.User.Username)},
				},
			})
			if err != nil {
				log.Printf("Error sending collective investments: %v", err)
				return
			}

			// If there are collective investments, send complete portfolio as followup
			if len(completePortfolio) > 0 {
				_, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
					Embeds: []*discordgo.MessageEmbed{createPortfolioEmbed(s, completePortfolio, "Complete", i.Member.User.Username)},
					Flags:  discordgo.MessageFlagsEphemeral,
				})
				if err != nil {
					log.Printf("Error sending complete portfolio: %v", err)
				}
			}
		} else {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Embeds: []*discordgo.MessageEmbed{createPortfolioEmbed(s, completePortfolio, "Personal", i.Member.User.Username)},
					Flags:  discordgo.MessageFlagsEphemeral,
				},
			})
		}
	}
}

func createPortfolioEmbed(s *discordgo.Session, investments map[string]*Investment, portfolioType string, username string) *discordgo.MessageEmbed {
	var fields []*discordgo.MessageEmbedField
	var totalValue, totalCost float64

	type performanceData struct {
		symbol       string
		amount       float64
		profitLoss   float64
		percentage   float64
		currentValue float64
		buyValue     float64
	}

	var performances []performanceData

	// Process each investment with high precision
	for _, inv := range investments {
		price, err := getCryptoPrice(inv.Symbol)
		if err != nil {
			log.Printf("Error getting price for %s: %v", inv.Symbol, err)
			continue
		}

		totalAmount := calculateTotalAmount(inv.Entries)
		avgBuyPrice := calculateAverageBuyPrice(inv.Entries)

		// Calculate values with high precision
		currentValue := math.Round(price.Price*totalAmount*100000000) / 100000000
		totalInvestment := 0.0

		for _, entry := range inv.Entries {
			entryInvestment := math.Round(entry.Amount*entry.BuyPrice*100000000) / 100000000
			totalInvestment += entryInvestment
		}

		profitLoss := math.Round((currentValue-totalInvestment)*100000000) / 100000000
		profitLossPercent := 0.0
		if totalInvestment > 0 {
			profitLossPercent = math.Round((profitLoss/totalInvestment)*10000) / 10000 * 100
		}

		// Accumulate totals with high precision
		totalValue = math.Round((totalValue+currentValue)*100000000) / 100000000
		totalCost = math.Round((totalCost+totalInvestment)*100000000) / 100000000

		// Build entries detail with enhanced precision
		var entriesDetail strings.Builder
		if len(inv.Entries) > 0 {
			entriesDetail.WriteString("\nEntries:")
			for i, entry := range inv.Entries {
				entryValue := math.Round(price.Price*entry.Amount*100000000) / 100000000
				entryInvestment := math.Round(entry.Amount*entry.BuyPrice*100000000) / 100000000
				entryPL := math.Round((entryValue-entryInvestment)*100000000) / 100000000
				entryPLPercent := 0.0
				if entryInvestment > 0 {
					entryPLPercent = math.Round((entryPL/entryInvestment)*10000) / 10000 * 100
				}

				entriesDetail.WriteString(fmt.Sprintf("\n%d. %s @ %s (%s) | P/L: %s (%s)",
					i+1,
					formatCryptoAmount(entry.Amount),
					formatFiatPrice(entry.BuyPrice),
					entry.EntryDate.Format("2006-01-02"),
					formatFiatAmount(entryPL),
					formatPercentage(entryPLPercent)))
			}
		}

		// Store performance data for summary
		performances = append(performances, performanceData{
			symbol:       inv.Symbol,
			amount:       totalAmount,
			profitLoss:   profitLoss,
			percentage:   profitLossPercent,
			currentValue: currentValue,
			buyValue:     totalInvestment,
		})

		description := fmt.Sprintf(
			"Total Amount: %s\n"+
				"Avg Buy Price: %s\n"+
				"Current Price: %s\n"+
				"Total Investment: %s\n"+
				"Current Value: %s\n"+
				"Total P/L: %s (%s)%s",
			formatCryptoAmount(totalAmount),
			formatFiatPrice(avgBuyPrice),
			formatFiatPrice(price.Price),
			formatFiatAmount(totalInvestment),
			formatFiatAmount(currentValue),
			formatFiatAmount(profitLoss),
			formatPercentage(profitLossPercent),
			entriesDetail.String())

		if inv.Type == "collective" {
			description += fmt.Sprintf("\n\nParticipants: %s\nCreated by: <@%s>",
				formatParticipants(s, inv.Participants),
				inv.CreatedBy)
		}

		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   fmt.Sprintf("%s (%s)", strings.ToUpper(inv.Symbol), inv.Type),
			Value:  description,
			Inline: true,
		})
	}

	// Calculate portfolio summary with high precision
	totalPL := math.Round((totalValue-totalCost)*100000000) / 100000000
	totalPLPercent := 0.0
	if totalCost > 0 {
		totalPLPercent = math.Round((totalPL/totalCost)*10000) / 10000 * 100
	}

	// Sort performances by percentage
	sort.Slice(performances, func(i, j int) bool {
		return performances[i].percentage > performances[j].percentage
	})

	var summaryBuilder strings.Builder
	summaryBuilder.WriteString(fmt.Sprintf("Total Investment: %s\n", formatFiatAmount(totalCost)))
	summaryBuilder.WriteString(fmt.Sprintf("Current Value: %s\n", formatFiatAmount(totalValue)))
	summaryBuilder.WriteString(fmt.Sprintf("Total P/L: %s (%s)\n",
		formatFiatAmount(totalPL),
		formatPercentage(totalPLPercent)))

	// Only show performers when we have more than 2 coins
	if len(performances) > 2 {
		summaryBuilder.WriteString("\nBest Performers:\n")
		for i := 0; i < min(3, len(performances)); i++ {
			p := performances[i]
			summaryBuilder.WriteString(fmt.Sprintf("%s: %s (%s)\n",
				strings.ToUpper(p.symbol),
				formatPercentage(p.percentage),
				formatFiatAmount(p.profitLoss)))
		}

		summaryBuilder.WriteString("\nWorst Performers:\n")
		for i := len(performances) - 1; i >= max(0, len(performances)-3); i-- {
			p := performances[i]
			summaryBuilder.WriteString(fmt.Sprintf("%s: %s (%s)\n",
				strings.ToUpper(p.symbol),
				formatPercentage(p.percentage),
				formatFiatAmount(p.profitLoss)))
		}
	}

	// Always show portfolio allocation
	summaryBuilder.WriteString("\nPortfolio Allocation:\n")
	for _, p := range performances {
		allocation := 0.0
		if totalValue > 0 {
			allocation = math.Round((p.currentValue/totalValue)*10000) / 10000 * 100
		}
		summaryBuilder.WriteString(fmt.Sprintf("%s: %s (%s)\n",
			strings.ToUpper(p.symbol),
			formatPercentage(allocation),
			formatCryptoAmount(p.amount)))
	}

	fields = append(fields, &discordgo.MessageEmbedField{
		Name:   fmt.Sprintf("%s Portfolio Summary", portfolioType),
		Value:  summaryBuilder.String(),
		Inline: false,
	})

	embedColor := 0xFF0000 // Red for loss
	if totalPL >= 0 {
		embedColor = 0x00FF00 // Green for profit
	}

	return &discordgo.MessageEmbed{
		Title:  fmt.Sprintf("%s Portfolio for %s", portfolioType, username),
		Color:  embedColor,
		Fields: fields,
		Footer: &discordgo.MessageEmbedFooter{
			Text: time.Now().Format("2006-01-02 15:04:05 MST"),
		},
	}
}

func handleRemoveInvestCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	symbol := strings.ToLower(options[0].StringValue())
	userID := i.Member.User.ID

	portMutex.Lock()
	portfolio, exists := portfolios[userID]
	removed := false
	if exists {
		if _, hasInvestment := portfolio.Investments[symbol]; hasInvestment {
			delete(portfolio.Investments, symbol)
			removed = true
		}
	}
	portMutex.Unlock()

	if removed {
		if err := savePortfolios(); err != nil {
			log.Printf("Error saving portfolios: %v", err)
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
