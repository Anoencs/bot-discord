package main

import (
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
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

// Add the generate invite URL function
func generateBotInviteURL(clientID string) string {
	permissions := 67584 // Nickname + View Channels + Send Messages
	return fmt.Sprintf("https://discord.com/api/oauth2/authorize?client_id=%s&permissions=%d&scope=bot%%20applications.commands",
		clientID,
		permissions)
}

func retryNicknameUpdate(s *discordgo.Session, guildID, nickname string) error {
	maxRetries := 5
	var lastErr error

	for retry := 0; retry < maxRetries; retry++ {
		if err := s.GuildMemberNickname(guildID, "@me", nickname); err != nil {
			lastErr = err
			log.Printf("Retry %d/%d: Error setting nickname: %v", retry+1, maxRetries, err)
			time.Sleep(time.Second * 2)
			continue
		}
		return nil
	}
	return lastErr
}

func updateBotNickname(bot *PriceBot, price float64) {
	nickname := formatNickname(bot.Symbol, price)
	err := bot.Session.GuildMemberNickname(bot.GuildID, "@me", nickname)
	if err != nil {
		log.Printf("Error updating nickname for %s: %v", bot.Symbol, err)
	} else {
		bot.LastPrice = price
		bot.LastUpdate = time.Now()
	}
}

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

func cleanupBots() {
	botsMutex.Lock()
	defer botsMutex.Unlock()

	for _, bot := range priceBots {
		bot.Session.Close()
		tokenPool = append(tokenPool, bot.Token)
	}
}
