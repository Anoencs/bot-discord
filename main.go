package main

import (
	"fmt"
	"log"
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

var (
	COINMARKETCAP_API_KEY string
	SANTIMENT_API_KEY     string
)

func init() {
	// In production (Railway), env vars are already set in the environment
	// Only try to load .env file during local development
	if os.Getenv("RAILWAY_ENVIRONMENT") == "" {
		if err := godotenv.Load(); err != nil {
			log.Printf("Warning: Error loading .env file (this is normal in production): %v", err)
		}
	}

	if err := loadInvestments(); err != nil {
		log.Printf("Error loading investments: %v", err)
	}

	// Get environment variables directly
	tokens := strings.Split(os.Getenv("BOT_TOKENS"), ",")
	clientIDs := strings.Split(os.Getenv("BOT_CLIENT_IDS"), ",")
	COINMARKETCAP_API_KEY = os.Getenv("COINMARKETCAP_API_KEY")
	SANTIMENT_API_KEY = os.Getenv("SANTIMENT_API_KEY")

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
			Name:        "restart-bot",
			Description: "Restart all price bots",
		},
		{
			Name:        "clear-bot",
			Description: "Remove all price bots",
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
		{
			Name:        "setinvest",
			Description: "Set investment amount and buy price",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:         discordgo.ApplicationCommandOptionString,
					Name:         "crypto",
					Description:  "Cryptocurrency (e.g., avalanche-2)",
					Required:     true,
					Autocomplete: true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionNumber,
					Name:        "amount",
					Description: "Amount of coins",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionNumber,
					Name:        "buy_price",
					Description: "Price per coin when bought in USD",
					Required:    true,
				},
			},
		},
		{
			Name:        "assets",
			Description: "Check all assets value",
		},
		{
			Name:        "removeinvest",
			Description: "Remove an investment",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:         discordgo.ApplicationCommandOptionString,
					Name:         "crypto",
					Description:  "Cryptocurrency to remove",
					Required:     true,
					Autocomplete: true,
				},
			},
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
		case "setinvest":
			handleSetInvestCommand(s, i)
		case "assets":
			handleAssetsCommand(s, i)
		case "removeinvest":
			handleRemoveInvestCommand(s, i)
		case "restart-bot":
			handleRestartCommand(s, i)
		case "clear-bot":
			handleClearCommand(s, i)
		}

	case discordgo.InteractionApplicationCommandAutocomplete:
		handleAutocomplete(s, i)
	}
}

func handleAutocomplete(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.ApplicationCommandData().Name != "price" &&
		i.ApplicationCommandData().Name != "add" &&
		i.ApplicationCommandData().Name != "setinvest" &&
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
