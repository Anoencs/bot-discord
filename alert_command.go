package main

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

// First, let's add the alert structure
type PriceAlert struct {
	Symbol        string
	GeckoID       string
	UpperTarget   float64
	LowerTarget   float64
	ChannelID     string
	GuildID       string
	CreatedAt     time.Time
	LastAlert     time.Time
	AlertCooldown time.Duration // Prevent spam
}

type AlertBot struct {
	Alerts     map[string][]PriceAlert // Map crypto symbol to its alerts
	AlertMutex sync.RWMutex
	Session    *discordgo.Session
}

var (
	alertBot = &AlertBot{
		Alerts: make(map[string][]PriceAlert),
	}
)

// Add function to check alerts
func checkAlerts(price *CryptoPrice, geckoID string) {
	alertBot.AlertMutex.Lock()
	defer alertBot.AlertMutex.Unlock()

	alerts, exists := alertBot.Alerts[geckoID]
	if !exists {
		return
	}

	currentTime := time.Now()
	var remainingAlerts []PriceAlert

	for _, alert := range alerts {
		// Check if cooldown period has passed
		if currentTime.Sub(alert.LastAlert) < alert.AlertCooldown {
			remainingAlerts = append(remainingAlerts, alert)
			continue
		}

		shouldAlert := false
		alertMessage := ""

		if alert.UpperTarget > 0 && price.Price >= alert.UpperTarget {
			shouldAlert = true
			alertMessage = fmt.Sprintf("🚨 %s has reached your upper target of $%.2f (Current: $%.2f)",
				alert.Symbol, alert.UpperTarget, price.Price)
		} else if alert.LowerTarget > 0 && price.Price <= alert.LowerTarget {
			shouldAlert = true
			alertMessage = fmt.Sprintf("🚨 %s has reached your lower target of $%.2f (Current: $%.2f)",
				alert.Symbol, alert.LowerTarget, price.Price)
		}

		if shouldAlert {
			// Send alert using the session from AlertBot
			embed := &discordgo.MessageEmbed{
				Title:       "Price Alert Triggered!",
				Description: alertMessage,
				Color:       0xff0000,
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:   "24h Change",
						Value:  fmt.Sprintf("%.2f%%", price.Change24h),
						Inline: true,
					},
					{
						Name:   "Volume (24h)",
						Value:  fmt.Sprintf("$%.2fM", price.Volume24h/1000000),
						Inline: true,
					},
				},
				Timestamp: time.Now().Format(time.RFC3339),
			}

			// Use the session from AlertBot to send the message
			_, err := alertBot.Session.ChannelMessageSendEmbed(alert.ChannelID, embed)
			if err != nil {
				log.Printf("Error sending alert for %s: %v", alert.Symbol, err)
			} else {
				alert.LastAlert = currentTime
			}
		}

		remainingAlerts = append(remainingAlerts, alert)
	}

	alertBot.Alerts[geckoID] = remainingAlerts
}

// Add this function to handle listing alerts
func handleListAlerts(s *discordgo.Session, i *discordgo.InteractionCreate) {
	alertBot.AlertMutex.RLock()
	defer alertBot.AlertMutex.RUnlock()

	embed := &discordgo.MessageEmbed{
		Title:  "Active Price Alerts",
		Color:  0x00ff00,
		Fields: []*discordgo.MessageEmbedField{},
	}

	if len(alertBot.Alerts) == 0 {
		embed.Description = "No active alerts"
	} else {
		for _, alerts := range alertBot.Alerts {
			for _, alert := range alerts {
				fieldValue := ""
				if alert.UpperTarget > 0 {
					fieldValue += fmt.Sprintf("Upper: $%.2f\n", alert.UpperTarget)
				}
				if alert.LowerTarget > 0 {
					fieldValue += fmt.Sprintf("Lower: $%.2f", alert.LowerTarget)
				}

				embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
					Name:   alert.Symbol,
					Value:  fieldValue,
					Inline: true,
				})
			}
		}
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}

// Add this function to handle removing alerts
func handleRemoveAlert(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	symbol := strings.ToLower(options[0].StringValue())

	alertBot.AlertMutex.Lock()
	defer alertBot.AlertMutex.Unlock()

	cryptoInfo, exists := commonCryptos[symbol]
	if !exists {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "No alerts found for this cryptocurrency",
			},
		})
		return
	}

	delete(alertBot.Alerts, cryptoInfo.GeckoID)

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Removed all alerts for %s", cryptoInfo.Symbol),
		},
	})
}

func handleSetAlert(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	symbol := strings.ToLower(options[0].StringValue())

	var upperTarget, lowerTarget float64

	// Parse options
	for _, opt := range options {
		switch opt.Name {
		case "upper":
			upperTarget = opt.FloatValue()
		case "lower":
			lowerTarget = opt.FloatValue()
		}
	}

	// Validate targets
	if upperTarget == 0 && lowerTarget == 0 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "⚠️ Please set at least one target price (upper or lower)\n" +
					"Example: `/setalert bitcoin upper:50000 lower:40000`",
			},
		})
		return
	}

	// Get crypto info
	cryptoInfo, exists := commonCryptos[symbol]
	if !exists {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("❌ Cryptocurrency '%s' not found.\n"+
					"Please use autocomplete to select a valid cryptocurrency.", symbol),
			},
		})
		return
	}

	// Get current price for reference
	price, err := getCryptoPrice(cryptoInfo.GeckoID)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("❌ Error fetching price for %s: %v", cryptoInfo.Symbol, err),
			},
		})
		return
	}

	// Create and store alert
	alert := PriceAlert{
		Symbol:        cryptoInfo.Symbol,
		GeckoID:       cryptoInfo.GeckoID,
		UpperTarget:   upperTarget,
		LowerTarget:   lowerTarget,
		ChannelID:     i.ChannelID,
		CreatedAt:     time.Now(),
		AlertCooldown: time.Minute * 5,
	}

	alertBot.AlertMutex.Lock()
	alertBot.Alerts[cryptoInfo.GeckoID] = append(alertBot.Alerts[cryptoInfo.GeckoID], alert)
	alertBot.AlertMutex.Unlock()

	// Create success response
	embed := &discordgo.MessageEmbed{
		Title: fmt.Sprintf("⚡ Price Alert Set for %s", cryptoInfo.Symbol),
		Color: 0x00ff00,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Current Price",
				Value:  fmt.Sprintf("$%.2f", price.Price),
				Inline: true,
			},
		},
	}

	if upperTarget > 0 {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Upper Target",
			Value:  fmt.Sprintf("$%.2f", upperTarget),
			Inline: true,
		})
	}

	if lowerTarget > 0 {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Lower Target",
			Value:  fmt.Sprintf("$%.2f", lowerTarget),
			Inline: true,
		})
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}
