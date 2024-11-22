package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

func handleAddCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	symbol := strings.ToLower(options[0].StringValue())

	cryptoInfo, exists := commonCryptos[symbol]
	if !exists {
		cryptoInfo = CryptoInfo{
			Symbol:  strings.ToUpper(symbol),
			GeckoID: symbol,
		}
	}

	// Acknowledge interaction first
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})

	// Verify the cryptocurrency exists and get initial price
	price, err := getCryptoPrice(cryptoInfo.GeckoID)
	if err != nil {
		content := fmt.Sprintf("Error: Could not fetch price for %s . Please verify the cryptocurrency name.", symbol)
		fmt.Println(cryptoInfo.GeckoID)
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &content,
		})
		return
	}

	botsMutex.Lock()
	defer botsMutex.Unlock()

	// Check if already monitoring
	if _, exists := priceBots[symbol]; exists {
		content := fmt.Sprintf("❌ Already monitoring %s", strings.ToUpper(symbol))
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &content,
		})
		return
	}

	// Check available tokens
	if len(tokenPool) == 0 {
		content := "❌ No available bot tokens. Please remove some existing price bots first."
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &content,
		})
		return
	}

	// Get token and create session
	token := tokenPool[0]
	tokenPool = tokenPool[1:]

	session, err := discordgo.New("Bot " + token)
	if err != nil {
		tokenPool = append(tokenPool, token)
		content := "❌ Error creating bot connection"
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &content,
		})
		return
	}

	// Enable required intents
	session.Identify.Intents = discordgo.IntentsGuildMembers | discordgo.IntentsGuilds

	// Create bot instance
	bot := &PriceBot{
		Token:      token,
		Symbol:     symbol,
		GuildID:    i.GuildID,
		Session:    session,
		LastPrice:  price.Price,
		LastUpdate: time.Now(),
	}

	// Add ready handler for immediate price update
	session.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("Price bot for %s is ready", strings.ToUpper(symbol))
		updateBotNickname(bot, price.Price)
	})

	// Open connection
	if err := session.Open(); err != nil {
		tokenPool = append(tokenPool, token)
		content := fmt.Sprintf("❌ Error connecting bot: %v", err)
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &content,
		})
		return
	}

	// Set initial nickname
	nickname := formatNickname(symbol, price.Price)

	// Retry logic for setting nickname
	err = retryNicknameUpdate(session, i.GuildID, nickname)
	if err != nil {
		log.Printf("Warning: Initial nickname set failed for %s: %v", symbol, err)
	}

	// Store bot in map
	priceBots[symbol] = bot

	// Create success response
	embed := &discordgo.MessageEmbed{
		Title: "✅ Price Bot Created",
		Color: 0x00ff00,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Cryptocurrency",
				Value:  cryptoInfo.Symbol,
				Inline: true,
			},
			{
				Name:   "Initial Price",
				Value:  formatPrice(symbol, price.Price),
				Inline: true,
			},
			{
				Name:   "Status",
				Value:  "Bot is now active and will update every 30 seconds",
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Started at %s", time.Now().Format("15:04:05 MST")),
		},
	}

	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
	})

	// Trigger immediate price update
	go func() {
		time.Sleep(5 * time.Second) // Small delay to ensure bot is ready
		updateBotNickname(bot, price.Price)
	}()
}

func handleRemoveCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	symbol := strings.ToLower(options[0].StringValue())

	botsMutex.Lock()
	if bot, exists := priceBots[symbol]; exists {
		bot.Session.Close()
		tokenPool = append(tokenPool, bot.Token)
		delete(priceBots, symbol)
		botsMutex.Unlock()

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("Removed price bot for %s", strings.ToUpper(symbol)),
			},
		})
	} else {
		botsMutex.Unlock()
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "No price bot found for this cryptocurrency",
			},
		})
	}
}
