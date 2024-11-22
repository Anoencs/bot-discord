package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

func handlePriceCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})

	options := i.ApplicationCommandData().Options
	cryptoName := options[0].StringValue()
	cryptoInfo, exists := commonCryptos[cryptoName]
	if !exists {
		cryptoInfo = CryptoInfo{
			Symbol:  strings.ToUpper(cryptoName),
			GeckoID: cryptoName,
		}
	}

	price, err := getCryptoPrice(cryptoInfo.GeckoID)
	if err != nil {
		content := fmt.Sprintf("Error: %s", err)
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &content,
		})
		return
	}

	sentiment, err := getSantimentDataForGreedFear("bitcoin")
	if err != nil {
		sentiment = "Data not available"
	}

	embed := &discordgo.MessageEmbed{
		Title: fmt.Sprintf("%s Price Info", cryptoInfo.Symbol),
		Color: 0x00ff00,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "💰 Price",
				Value:  fmt.Sprintf("$%.2f", price.Price),
				Inline: true,
			},
			{
				Name:   "📈 24h Change",
				Value:  fmt.Sprintf("%.2f%%", price.Change24h),
				Inline: true,
			},
			{
				Name:   "📊 Market Cap",
				Value:  fmt.Sprintf("$%.2fM", price.MarketCap/1000000),
				Inline: true,
			},
			{
				Name:   "📉 24h Volume",
				Value:  fmt.Sprintf("$%.2fM", price.Volume24h/1000000),
				Inline: true,
			},
			{
				Name:   "🌟 Market Sentiment",
				Value:  sentiment,
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Data from CoinGecko • " + time.Now().Format("2006-01-02 15:04:05 MST"),
		},
	}

	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
	})
}

func handleHelpCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	embed := &discordgo.MessageEmbed{
		Title: "🤖 Bot Commands",
		Color: 0x00ff00,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:  "/price [crypto]",
				Value: "Get current price information\nExample: `/price bitcoin`",
			},
			{
				Name:  "/add [crypto]",
				Value: "Add a new price display bot\nExample: `/add bitcoin`",
			},
			{
				Name:  "/remove [crypto]",
				Value: "Remove a price display bot\nExample: `/remove bitcoin`",
			},
			{
				Name:  "/help",
				Value: "Show this help message",
			},
			{
				Name: "/setalert [crypto] [upper] [lower]",
				Value: `Set price alerts for a cryptocurrency
Example: 
• /setalert bitcoin upper:50000 lower:40000
• /setalert ethereum upper:3000
• /setalert solana lower:20

How to use:
1. Type /setalert and start typing crypto name
2. Select from autocomplete suggestions
3. Set upper price target (optional)
4. Set lower price target (optional)
Must set at least one target (upper or lower)`,
			},
			{
				Name:  "/removealert [crypto]",
				Value: "Remove price alerts for a cryptocurrency\nExample: /removealert bitcoin",
			},
			{
				Name:  "/listalerts",
				Value: "Show all your active price alerts",
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Type crypto name to see suggestions",
		},
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}
