package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

type CryptoPrice struct {
	Price     float64 `json:"usd"`
	Change24h float64 `json:"usd_24h_change"`
	MarketCap float64 `json:"usd_market_cap"`
	Volume24h float64 `json:"usd_24h_vol"`
}

type PriceBot struct {
	Token      string
	Symbol     string
	GuildID    string
	Session    *discordgo.Session
	LastPrice  float64
	LastUpdate time.Time
}

var (
	priceBots = make(map[string]*PriceBot)
	botsMutex sync.RWMutex
	tokenPool []string
)

func init() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}
	tokenPool = strings.Split(os.Getenv("BOT_TOKENS"), ",")
}

func main() {
	discord, err := discordgo.New("Bot " + os.Getenv("DISCORD_TOKEN"))
	if err != nil {
		log.Fatal("Error creating Discord session:", err)
		return
	}
	alertBot.Session = discord

	discord.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("Bot is ready: %v#%v", s.State.User.Username, s.State.User.Discriminator)
		registerCommands(s)
	})

	discord.AddHandler(interactionHandler)
	discord.Identify.Intents = discordgo.IntentsGuildMembers | discordgo.IntentsGuilds

	err = discord.Open()
	if err != nil {
		log.Fatal("Error opening connection:", err)
		return
	}

	// Start price update routine
	go updatePrices()

	fmt.Println("Bot is running. Press Ctrl+C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM)
	<-sc

	cleanupBots()
	discord.Close()
}

func registerCommands(s *discordgo.Session) {
	commands := []*discordgo.ApplicationCommand{
		{
			Name:        "price",
			Description: "Get cryptocurrency price information",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:         discordgo.ApplicationCommandOptionString,
					Name:         "crypto",
					Description:  "Cryptocurrency name (e.g., bitcoin, ethereum)",
					Required:     true,
					Autocomplete: true,
				},
			},
		},
		{
			Name:        "add",
			Description: "Add a new price display bot",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:         discordgo.ApplicationCommandOptionString,
					Name:         "crypto",
					Description:  "Cryptocurrency to track (e.g., bitcoin)",
					Required:     true,
					Autocomplete: true,
				},
			},
		},
		{
			Name:        "remove",
			Description: "Remove a price display bot",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "crypto",
					Description: "Cryptocurrency to stop tracking",
					Required:    true,
				},
			},
		},
		{
			Name:        "help",
			Description: "Show available commands",
		},
		{
			Name:        "invite",
			Description: "Get invite links for available price bots",
		},
		{
			Name:        "setalert",
			Description: "Set price alert for a cryptocurrency",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:         discordgo.ApplicationCommandOptionString,
					Name:         "crypto",
					Description:  "Cryptocurrency name (e.g., bitcoin, ethereum)",
					Required:     true,
					Autocomplete: true, // Enable autocomplete
				},
				{
					Type:        discordgo.ApplicationCommandOptionNumber,
					Name:        "upper",
					Description: "Alert when price goes above this value (e.g., 50000)",
					Required:    false,
				},
				{
					Type:        discordgo.ApplicationCommandOptionNumber,
					Name:        "lower",
					Description: "Alert when price goes below this value (e.g., 40000)",
					Required:    false,
				},
			},
		},
		{
			Name:        "removealert",
			Description: "Remove price alert for a cryptocurrency",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:         discordgo.ApplicationCommandOptionString,
					Name:         "crypto",
					Description:  "Cryptocurrency to remove alert for",
					Required:     true,
					Autocomplete: true,
				},
			},
		},
		{
			Name:        "listalerts",
			Description: "List all active price alerts",
		},
	}

	for _, cmd := range commands {
		_, err := s.ApplicationCommandCreate(s.State.User.ID, "", cmd)
		if err != nil {
			log.Printf("Error creating command %v: %v", cmd.Name, err)
		}
	}
}

func interactionHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		handleSlashCommand(s, i)
	case discordgo.InteractionApplicationCommandAutocomplete:
		handleAutocomplete(s, i)
	}
}

func handleSlashCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		switch i.ApplicationCommandData().Name {
		case "price":
			handlePriceCommand(s, i)
		case "add":
			handleAddCommand(s, i)
		case "remove":
			handleRemoveCommand(s, i)
		case "help":
			handleHelpCommand(s, i)
		case "invite":
			handleInviteCommand(s, i)
		case "setalert": // Add this case
			handleSetAlert(s, i)
		case "removealert": // Add this case
			handleRemoveAlert(s, i)
		case "listalerts": // Add this case
			handleListAlerts(s, i)
		}

	case discordgo.InteractionApplicationCommandAutocomplete:
		handleAutocomplete(s, i)
	}
}

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
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Data from CoinGecko • " + time.Now().Format("2006-01-02 15:04:05 MST"),
		},
	}

	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
	})
}

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

// Helper functions
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

func handleAutocomplete(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.ApplicationCommandData().Name != "price" &&
		i.ApplicationCommandData().Name != "add" &&
		i.ApplicationCommandData().Name != "setalert" {
		return
	}

	input := strings.ToLower(i.ApplicationCommandData().Options[0].StringValue())

	// Debug log to see what we're searching with
	log.Printf("Searching for: %s", input)

	var choices []*discordgo.ApplicationCommandOptionChoice
	for geckoID, info := range commonCryptos {
		// Debug log to see what cryptocurrencies are available
		log.Printf("Checking: %s (%s)", info.Symbol, geckoID)

		// Make the search more flexible
		if strings.Contains(strings.ToLower(geckoID), input) ||
			strings.Contains(strings.ToLower(info.Symbol), input) {
			displayName := fmt.Sprintf("%s (%s)", info.Symbol, strings.Title(geckoID))

			// Debug log for matches
			log.Printf("Found match: %s", displayName)

			choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
				Name:  displayName,
				Value: geckoID,
			})
		}
	}

	// Log total number of matches
	log.Printf("Total matches found: %d", len(choices))

	// If we have too many matches, sort them by relevance and take top 25
	if len(choices) > 25 {
		// Sort by exact match first, then by contains
		sort.Slice(choices, func(i, j int) bool {
			iExact := strings.ToLower(choices[i].Name) == input
			jExact := strings.ToLower(choices[j].Name) == input
			if iExact != jExact {
				return iExact
			}
			return len(choices[i].Name) < len(choices[j].Name)
		})
		choices = choices[:25]
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{
			Choices: choices,
		},
	})
}
func updateAllBotPrices() {
	botsMutex.RLock()
	defer botsMutex.RUnlock()

	for symbol, bot := range priceBots {
		// Skip if last update was less than 5 seconds ago
		if time.Since(bot.LastUpdate) < 5*time.Second {
			continue
		}

		// Get the proper GeckoID from commonCryptos
		cryptoInfo, exists := commonCryptos[symbol]
		if !exists {
			// If not found in commonCryptos, use symbol as GeckoID
			cryptoInfo = CryptoInfo{
				Symbol:  strings.ToUpper(symbol),
				GeckoID: symbol,
			}
		}

		// Use GeckoID for price fetching
		price, err := getCryptoPrice(cryptoInfo.GeckoID)
		if err != nil {
			log.Printf("Error fetching price for %s: %v", cryptoInfo.Symbol, err)
			continue
		}

		// Use Symbol for display
		nickname := fmt.Sprintf("%s $%.2f", cryptoInfo.Symbol, price.Price)
		if cryptoInfo.Symbol == "BTC" {
			nickname = fmt.Sprintf("BTC $%.0f", price.Price)
		}

		err = bot.Session.GuildMemberNickname(bot.GuildID, "@me", nickname)
		if err != nil {
			log.Printf("Error updating nickname for %s: %v", cryptoInfo.Symbol, err)
		} else {
			bot.LastPrice = price.Price
			bot.LastUpdate = time.Now()
		}
	}
}

func updatePrices() {
	// Initial fast update (5 seconds)
	initialTicker := time.NewTicker(5 * time.Second)
	go func() {
		for range initialTicker.C {
			updateAllBotPrices()
		}
	}()
	time.Sleep(30 * time.Second) // Run fast updates for 30 seconds
	initialTicker.Stop()

	// Regular updates every 30 seconds
	regularTicker := time.NewTicker(30 * time.Second)
	for range regularTicker.C {
		alertBot.AlertMutex.RLock()
		for geckoID := range alertBot.Alerts {
			price, err := getCryptoPrice(geckoID)
			if err != nil {
				log.Printf("Error fetching price for %s: %v", geckoID, err)
				continue
			}

			go checkAlerts(price, geckoID)
		}
		alertBot.AlertMutex.RUnlock()

		updateAllBotPrices()
	}
}

func cleanupBots() {
	botsMutex.Lock()
	defer botsMutex.Unlock()

	for _, bot := range priceBots {
		bot.Session.Close()
		tokenPool = append(tokenPool, bot.Token)
	}
}

func getCryptoPrice(id string) (*CryptoPrice, error) {
	url := fmt.Sprintf("https://api.coingecko.com/api/v3/simple/price?ids=%s&vs_currencies=usd&include_market_cap=true&include_24hr_vol=true&include_24hr_change=true", id)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		return nil, fmt.Errorf("rate limit exceeded, please try again later")
	}

	var data map[string]CryptoPrice
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	if price, ok := data[id]; ok {
		return &price, nil

	}

	return nil, fmt.Errorf("cryptocurrency '%s' not found", id)
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
