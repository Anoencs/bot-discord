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

var (
	portfolios = make(map[string]*Portfolio)
	portMutex  sync.RWMutex
	saveFile   = "portfolios.json"
)

type Investment struct {
	Amount       float64   `json:"amount"`
	Symbol       string    `json:"symbol"`
	BuyPrice     float64   `json:"buy_price"`
	Type         string    `json:"type"`                   // "personal" or "collective"
	Participants []string  `json:"participants,omitempty"` // List of user IDs for collective investments
	CreatedBy    string    `json:"created_by"`
	CreatedAt    time.Time `json:"created_at"`
}

type Portfolio struct {
	UserID      string                 `json:"user_id"`
	Investments map[string]*Investment `json:"investments"`
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
			// Split the mentions string by spaces
			mentionsStr := opt.StringValue()
			mentions := strings.Fields(mentionsStr)

			// Process each mention
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
	var participantNames []string
	for _, userID := range participants {
		if user, err := s.User(userID); err == nil {
			participantNames = append(participantNames, user.Username)
		}
	}

	// For collective investments, require participants
	if investType == "collective" && len(participants) == 0 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Collective investments require at least one participant. Please mention the participants.",
			},
		})
		return
	}

	userID := i.Member.User.ID
	geckoID := symbol
	if strings.Contains(symbol, "(") {
		parts := strings.Split(symbol, "(")
		geckoID = strings.TrimSpace(strings.TrimRight(parts[1], ")"))
	}

	// Create investment entry
	investment := &Investment{
		Amount:    amount,
		Symbol:    geckoID,
		BuyPrice:  buyPrice,
		Type:      investType,
		CreatedBy: userID,
		CreatedAt: time.Now(),
	}

	if investType == "collective" {
		investment.Participants = participants
	}

	// Save to creator's portfolio
	portfolio := getOrCreatePortfolio(userID)
	portfolio.Investments[geckoID] = investment

	// For collective investments, create reference in participants' portfolios
	if investType == "collective" {
		for _, participantID := range participants {
			if participantID != userID { // Skip creator as they already have it
				partPortfolio := getOrCreatePortfolio(participantID)
				partPortfolio.Investments[geckoID] = investment
			}
		}
	}

	if err := savePortfolios(); err != nil {
		log.Printf("Error saving portfolios: %v", err)
	}

	// Create response message
	var response string
	if investType == "collective" {
		participantsStr := "No participants"
		if len(participantNames) > 0 {
			participantsStr = strings.Join(participantNames, ", ")
		}
		response = fmt.Sprintf("Collective investment set: %.4f %s at $%.2f per coin\nParticipants: %s",
			amount, strings.ToUpper(geckoID), buyPrice, participantsStr)
	} else {
		response = fmt.Sprintf("Personal investment set: %.4f %s at $%.2f per coin",
			amount, strings.ToUpper(geckoID), buyPrice)
	}

	interactResp := &discordgo.InteractionResponse{
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
	}
	s.InteractionRespond(i.Interaction, interactResp)
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
		// Send public collective investments message first (visible to everyone except command caller)
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
		}

		// Send private complete portfolio only to the command caller
		// Regardless of whether they have collective investments or not
		completePortfolio := make(map[string]*Investment)
		// Add all personal investments
		for k, v := range personalInvestments {
			completePortfolio[k] = v
		}
		// Add all collective investments
		for k, v := range visibleCollectiveInvestments {
			completePortfolio[k] = v
		}

		// Only send the follow-up message if there are any investments to show
		if len(completePortfolio) > 0 {
			_, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
				Embeds: []*discordgo.MessageEmbed{createPortfolioEmbed(s, completePortfolio, "Complete", i.Member.User.Username)},
				Flags:  discordgo.MessageFlagsEphemeral,
			})
			if err != nil {
				log.Printf("Error sending complete portfolio: %v", err)
			}
		}
	}
}

func createPortfolioEmbed(s *discordgo.Session, investments map[string]*Investment, portfolioType string, username string) *discordgo.MessageEmbed {
	var fields []*discordgo.MessageEmbedField
	var totalValue, totalCost float64
	// Collect all unique participants
	uniqueParticipants := make(map[string]bool)
	if portfolioType == "Collective" {
		for _, inv := range investments {
			if inv.Type == "collective" {
				// Add creator
				uniqueParticipants[inv.CreatedBy] = true
				// Add participants
				for _, participantID := range inv.Participants {
					uniqueParticipants[participantID] = true
				}
			}
		}
	}
	// Create mentions string
	var mentions []string
	for userID := range uniqueParticipants {
		mentions = append(mentions, fmt.Sprintf("<@%s>", userID))
	}

	for _, inv := range investments {
		price, err := getCryptoPrice(inv.Symbol)
		if err != nil {
			log.Printf("Error getting price for %s: %v", inv.Symbol, err)
			continue
		}

		currentValue := price.Price * inv.Amount
		initialCost := inv.BuyPrice * inv.Amount
		profitLoss := currentValue - initialCost
		profitLossPercent := (profitLoss / initialCost) * 100

		totalValue += currentValue
		totalCost += initialCost

		var description string
		if inv.Type == "collective" {
			description = fmt.Sprintf(
				"Amount: %.4f\n"+
					"Buy Price: $%.2f\n"+
					"Current Price: $%.2f\n"+
					"Value: $%.2f\n"+
					"P/L: $%.2f (%.2f%%)\n"+
					"Participants: %s\n"+
					"Created by: <@%s>\n"+
					"Created at: %s",
				inv.Amount,
				inv.BuyPrice,
				price.Price,
				currentValue,
				profitLoss,
				profitLossPercent,
				formatParticipants(s, inv.Participants),
				inv.CreatedBy,
				inv.CreatedAt.Format("2006-01-02 15:04:05"),
			)
		} else {
			description = fmt.Sprintf(
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
			)
		}

		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   fmt.Sprintf("%s (%s)", strings.ToUpper(inv.Symbol), inv.Type),
			Value:  description,
			Inline: true,
		})
	}

	// Add summary field
	totalPL := totalValue - totalCost
	totalPLPercent := 0.0
	if totalCost > 0 {
		totalPLPercent = (totalPL / totalCost) * 100
	}

	fields = append(fields, &discordgo.MessageEmbedField{
		Name: fmt.Sprintf("%s Portfolio Summary", portfolioType),
		Value: fmt.Sprintf(
			"Total Cost: $%.2f\n"+
				"Total Value: $%.2f\n"+
				"Total P/L: $%.2f (%.2f%%)",
			totalCost,
			totalValue,
			totalPL,
			totalPLPercent,
		),
		Inline: false,
	})

	embedColor := 0xFF0000 // Red for loss
	if totalPL >= 0 {
		embedColor = 0x00FF00 // Green for profit
	}
	var description string
	if portfolioType == "Collective" && len(mentions) > 0 {
		description = fmt.Sprintf("Hey %s! Check out our collective investments!",
			strings.Join(mentions, " "))
	}

	return &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s Portfolio for %s", portfolioType, username),
		Description: description,
		Color:       embedColor,
		Fields:      fields,
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
