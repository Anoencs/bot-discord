package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	mongoClient     *mongo.Client
	database        *mongo.Database
	portfoliosMutex sync.Mutex
	userPortfolios  map[string]Portfolio
)

func connectDB() error {
	// Get MongoDB URI from environment variable
	mongoURI := os.Getenv("MONGODB_URI")
	if mongoURI == "" {
		return fmt.Errorf("MONGODB_URI environment variable not set")
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create client
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		return fmt.Errorf("failed to connect to MongoDB: %v", err)
	}

	// Ping the database
	err = client.Ping(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to ping MongoDB: %v", err)
	}

	// Set the global client
	mongoClient = client

	// Set the database
	dbName := os.Getenv("MONGODB_DATABASE")
	if dbName == "" {
		dbName = "cryptobot" // default database name
	}
	database = client.Database(dbName)

	log.Println("Successfully connected to MongoDB")
	return nil
}

func getCollection(name string) *mongo.Collection {
	if database == nil {
		log.Fatal("Database not initialized")
	}
	return database.Collection(name)
}

func disconnectDB() {
	if mongoClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := mongoClient.Disconnect(ctx); err != nil {
			log.Printf("Error disconnecting from MongoDB: %v", err)
		}
	}
}

// Portfolio operations
func savePortfolioToDB(portfolio *Portfolio) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := getCollection("portfolios")

	opts := options.Update().SetUpsert(true)
	filter := map[string]interface{}{"user_id": portfolio.UserID}
	update := map[string]interface{}{"$set": portfolio}

	_, err := collection.UpdateOne(ctx, filter, update, opts)
	return err
}

func loadPortfoliosFromDB() error {
	log.Println("Starting to load portfolios from database...")

	collection := getCollection("portfolios")
	log.Printf("Got collection: %v", collection.Name())

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Find all documents
	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		return fmt.Errorf("error finding portfolios: %v", err)
	}
	defer cursor.Close(ctx)

	// Count documents
	count, err := collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		log.Printf("Error counting documents: %v", err)
	} else {
		log.Printf("Found %d portfolios in database", count)
	}

	var portfolios []Portfolio
	if err = cursor.All(ctx, &portfolios); err != nil {
		return fmt.Errorf("error decoding portfolios: %v", err)
	}

	// Debug: Print each portfolio
	for _, p := range portfolios {
		log.Printf("Loaded portfolio: UserID: %s, Investments: %+v",
			p.UserID, p.Investments)
	}

	// Initialize the map if it hasn't been initialized yet
	portfoliosMutex.Lock()
	if userPortfolios == nil {
		userPortfolios = make(map[string]Portfolio)
	}

	// Clear existing portfolios before loading new ones
	userPortfolios = make(map[string]Portfolio)

	for _, p := range portfolios {
		userPortfolios[p.UserID] = p
	}
	portfoliosMutex.Unlock()

	log.Printf("Successfully loaded %d portfolios into memory", len(portfolios))
	return nil
}
