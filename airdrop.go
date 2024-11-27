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

type AirdropMonitor struct {
	Sources     map[string]bool      // Các channel nguồn
	Keywords    []string             // Từ khóa theo dõi
	TargetID    string               // Channel đích để gửi thông báo
	LastChecked map[string]time.Time // Track thời gian check cuối
	mutex       sync.RWMutex
}

var airdropMonitor *AirdropMonitor

func InitAirdropMonitor(s *discordgo.Session) {
	airdropMonitor = &AirdropMonitor{
		Sources:     make(map[string]bool),
		Keywords:    []string{"airdrop", "giveaway", "claim", "whitelist", "free mint", "early access", "OG role", "mint pass"},
		LastChecked: make(map[string]time.Time),
		TargetID:    os.Getenv("AIRDROP_TARGET_CHANNEL"),
	}

	// Hàm helper để verify và thêm channel
	addChannelIfAccessible := func(s *discordgo.Session, channelID string, name string) {
		channel, err := s.Channel(channelID)
		if err == nil && channel != nil {
			airdropMonitor.mutex.Lock()
			airdropMonitor.Sources[channelID] = true
			airdropMonitor.mutex.Unlock()
			log.Printf("Successfully added channel: %s (%s)", name, channelID)
		} else {
			log.Printf("Could not access channel %s (%s): %v", name, channelID, err)
		}
	}

	// Thử lấy danh sách channels từ config file nếu có
	if configFile, err := os.Open("airdrop_channels.json"); err == nil {
		var config struct {
			Channels map[string]string `json:"channels"`
		}
		if err := json.NewDecoder(configFile).Decode(&config); err == nil {
			for name, id := range config.Channels {
				addChannelIfAccessible(s, id, name)
			}
		}
		configFile.Close()
	}

	// Nếu không có channels nào được thêm từ config, thử thêm một số channels mặc định
	airdropMonitor.mutex.RLock()
	channelCount := len(airdropMonitor.Sources)
	airdropMonitor.mutex.RUnlock()

	if channelCount == 0 {
		log.Println("No channels loaded from config, waiting for manual channel addition")
	}

	log.Printf("Airdrop Monitor initialized with %d channels", channelCount)
}

// SaveChannelsConfig lưu danh sách channels hiện tại vào file
func SaveChannelsConfig() error {
	airdropMonitor.mutex.RLock()
	defer airdropMonitor.mutex.RUnlock()

	config := struct {
		Channels map[string]string `json:"channels"`
	}{
		Channels: make(map[string]string),
	}

	// Convert Sources map to config format
	for channelID := range airdropMonitor.Sources {
		config.Channels[fmt.Sprintf("Channel-%s", channelID)] = channelID
	}

	// Save to file
	file, err := os.Create("airdrop_channels.json")
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewEncoder(file).Encode(config)
}

// Thêm hàm để lưu channel khi được thêm mới
func AddChannelToMonitor(s *discordgo.Session, channelID string) error {
	// Verify channel existence and accessibility
	channel, err := s.Channel(channelID)
	if err != nil {
		return fmt.Errorf("could not access channel: %v", err)
	}

	// Add to monitor
	airdropMonitor.mutex.Lock()
	airdropMonitor.Sources[channelID] = true
	airdropMonitor.mutex.Unlock()

	// Save updated config
	if err := SaveChannelsConfig(); err != nil {
		log.Printf("Warning: Could not save channel config: %v", err)
	}

	log.Printf("Added channel to monitor: %s (%s)", channel.Name, channelID)
	return nil
}

func GetAirdropCommands() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "airdrop",
		Description: "Airdrop monitor commands",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "addsource",
				Description: "Add source channel to monitor",
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        "channel_id",
						Description: "Channel ID to monitor",
						Required:    true,
					},
				},
			},
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "removesource",
				Description: "Remove source channel",
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        "channel_id",
						Description: "Channel ID to remove",
						Required:    true,
					},
				},
			},
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "listsources",
				Description: "List all monitored channels",
			},
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "addkeyword",
				Description: "Add keyword to monitor",
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        "keyword",
						Description: "Keyword to monitor",
						Required:    true,
					},
				},
			},
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "listkeywords",
				Description: "List all monitored keywords",
			},
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "removekeyword",
				Description: "Remove a keyword",
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        "keyword",
						Description: "Keyword to remove",
						Required:    true,
					},
				},
			},
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "settarget",
				Description: "Set target channel for notifications",
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        "channel_id",
						Description: "Channel ID for notifications",
						Required:    true,
					},
				},
			},
		},
	}
}

func handleAirdropCommands(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	if len(options) == 0 {
		return
	}

	subCmd := options[0]
	switch subCmd.Name {
	case "addsource":
		channelID := subCmd.Options[0].StringValue()
		airdropMonitor.mutex.Lock()
		airdropMonitor.Sources[channelID] = true
		airdropMonitor.mutex.Unlock()

		respondToInteraction(s, i, fmt.Sprintf("Added channel `%s` to monitoring list", channelID))

	case "removesource":
		channelID := subCmd.Options[0].StringValue()
		airdropMonitor.mutex.Lock()
		delete(airdropMonitor.Sources, channelID)
		airdropMonitor.mutex.Unlock()

		respondToInteraction(s, i, fmt.Sprintf("Removed channel `%s` from monitoring list", channelID))

	case "listsources":
		airdropMonitor.mutex.RLock()
		sources := "Monitored channels:\n"
		for channelID := range airdropMonitor.Sources {
			sources += fmt.Sprintf("- <#%s>\n", channelID)
		}
		airdropMonitor.mutex.RUnlock()

		respondToInteraction(s, i, sources)

	case "addkeyword":
		keyword := strings.ToLower(subCmd.Options[0].StringValue())
		airdropMonitor.mutex.Lock()
		airdropMonitor.Keywords = append(airdropMonitor.Keywords, keyword)
		airdropMonitor.mutex.Unlock()

		respondToInteraction(s, i, fmt.Sprintf("Added keyword `%s` to monitoring list", keyword))

	case "listkeywords":
		airdropMonitor.mutex.RLock()
		keywords := "Monitored keywords:\n"
		for _, keyword := range airdropMonitor.Keywords {
			keywords += fmt.Sprintf("- `%s`\n", keyword)
		}
		airdropMonitor.mutex.RUnlock()

		respondToInteraction(s, i, keywords)

	case "removekeyword":
		keyword := strings.ToLower(subCmd.Options[0].StringValue())
		airdropMonitor.mutex.Lock()
		for i, k := range airdropMonitor.Keywords {
			if k == keyword {
				airdropMonitor.Keywords = append(airdropMonitor.Keywords[:i], airdropMonitor.Keywords[i+1:]...)
				break
			}
		}
		airdropMonitor.mutex.Unlock()

		respondToInteraction(s, i, fmt.Sprintf("Removed keyword `%s` from monitoring list", keyword))

	case "settarget":
		channelID := subCmd.Options[0].StringValue()
		airdropMonitor.mutex.Lock()
		airdropMonitor.TargetID = channelID
		airdropMonitor.mutex.Unlock()

		respondToInteraction(s, i, fmt.Sprintf("Set target channel to <#%s>", channelID))
	}
}

func handleAirdropMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore bot messages
	if m.Author.Bot {
		return
	}

	airdropMonitor.mutex.RLock()
	defer airdropMonitor.mutex.RUnlock()

	// Check if channel is monitored
	if !airdropMonitor.Sources[m.ChannelID] {
		return
	}

	// Check keywords
	content := strings.ToLower(m.Content)
	hasKeyword := false
	for _, keyword := range airdropMonitor.Keywords {
		if strings.Contains(content, keyword) {
			hasKeyword = true
			break
		}
	}

	if !hasKeyword {
		return
	}

	// Create embed
	embed := &discordgo.MessageEmbed{
		Title:       "🎁 New Airdrop Detected!",
		Description: m.Content,
		Color:       0x00ff00,
		Timestamp:   time.Now().Format(time.RFC3339),
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Source",
				Value:  fmt.Sprintf("Server: %s\nChannel: <#%s>", m.GuildID, m.ChannelID),
				Inline: false,
			},
			{
				Name:   "Original Message",
				Value:  fmt.Sprintf("[Click here](https://discord.com/channels/%s/%s/%s)", m.GuildID, m.ChannelID, m.ID),
				Inline: false,
			},
		},
	}

	// Add author info if available
	if m.Author != nil {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Posted by",
			Value:  fmt.Sprintf("%s#%s", m.Author.Username, m.Author.Discriminator),
			Inline: false,
		})
	}

	// Add any attachments
	if len(m.Attachments) > 0 {
		embed.Image = &discordgo.MessageEmbedImage{
			URL: m.Attachments[0].URL,
		}
	}

	// Send notification
	_, err := s.ChannelMessageSendEmbed(airdropMonitor.TargetID, embed)
	if err != nil {
		log.Printf("Error sending airdrop notification: %v", err)
	}
}

func respondToInteraction(s *discordgo.Session, i *discordgo.InteractionCreate, content string) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
		},
	})
	if err != nil {
		log.Printf("Error responding to interaction: %v", err)
	}
}
