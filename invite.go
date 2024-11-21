package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

// Add these new types and variables at the top of your file
type BotConfig struct {
	Token    string
	ClientID string
}

var (
	// Update your existing tokenPool to use BotConfig
	botConfigs []BotConfig
)

// Update your init() function
func init() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	// Load bot configurations from environment
	tokens := strings.Split(os.Getenv("BOT_TOKENS"), ",")
	clientIDs := strings.Split(os.Getenv("BOT_CLIENT_IDS"), ",")

	if len(tokens) != len(clientIDs) {
		log.Fatal("Number of tokens and client IDs must match")
	}

	for i := range tokens {
		botConfigs = append(botConfigs, BotConfig{
			Token:    strings.TrimSpace(tokens[i]),
			ClientID: strings.TrimSpace(clientIDs[i]),
		})
	}
}

// Add the generate invite URL function
func generateBotInviteURL(clientID string) string {
	permissions := 67584 // Nickname + View Channels + Send Messages
	return fmt.Sprintf("https://discord.com/api/oauth2/authorize?client_id=%s&permissions=%d&scope=bot%%20applications.commands",
		clientID,
		permissions)
}

// Add a command to show invite URLs
func handleInviteCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	botsMutex.RLock()
	defer botsMutex.RUnlock()

	var availableBots []BotConfig
	usedTokens := make(map[string]bool)

	// Mark used tokens
	for _, bot := range priceBots {
		usedTokens[bot.Token] = true
	}

	// Find available bots
	for _, config := range botConfigs {
		if !usedTokens[config.Token] {
			availableBots = append(availableBots, config)
		}
	}

	// Create embed with invite URLs
	fields := []*discordgo.MessageEmbedField{
		{
			Name: "Available Price Bots",
			Value: fmt.Sprintf("There are %d bots available for price display",
				len(availableBots)),
		},
	}

	for i, config := range availableBots {
		inviteURL := generateBotInviteURL(config.ClientID)
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:  fmt.Sprintf("Bot %d", i+1),
			Value: fmt.Sprintf("[Click to invite](%s)", inviteURL),
		})
	}

	embed := &discordgo.MessageEmbed{
		Title:  "Price Bot Invite Links",
		Color:  0x00ff00,
		Fields: fields,
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Use /add after inviting a bot",
		},
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
			Flags:  discordgo.MessageFlagsEphemeral, // Only visible to command user
		},
	})
}
