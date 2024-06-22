package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var client *mongo.Client

type Document struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Symbol   string             `bson:"Symbol" json:"Symbol"`
	Year     int                `bson:"Year" json:"Year"`
	Quarter  string             `bson:"Quarter" json:"Quarter"`
	Datetime primitive.DateTime `bson:"Datetime" json:"Datetime"`
	Url      string             `bson:"Url" json:"Url"`
	EPS      float64            `bson:"EPS" json:"EPS"`
}

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		log.Fatal("MONGO_URI environment variable not set")
	}

	clientOptions := options.Client().ApplyURI(mongoURI)

	// Connect to MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err = mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	// Check the connection
	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatalf("Failed to ping MongoDB: %v", err)
	}

	fmt.Println("Connected to MongoDB!")

	r := gin.Default()
	r.Use(cors.Default())
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	api := r.Group("/api/v1")
	{
		api.GET("/hello", helloHandler)
		api.GET("/data", getDataHandler)
		api.GET("/symbols", getUniqueSymbolsHandler)
	}

	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}

func helloHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "Hello, World!",
	})
}

func getDataHandler(c *gin.Context) {
	if client == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "MongoDB client is not initialized"})
		return
	}
	collection := client.Database("StockThaiAnalysis").Collection("predict")

	limit, err := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if err != nil || limit <= 0 {
		limit = 10
	}
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page <= 0 {
		page = 1
	}
	skip := (page - 1) * limit

	sort := c.DefaultQuery("sort", "_id")
	order := c.DefaultQuery("order", "asc")
	sortOrder := 1
	if order == "desc" {
		sortOrder = -1
	}

	filter := bson.D{}
	if symbol := c.Query("Symbol"); symbol != "" {
		filter = append(filter, bson.E{Key: "Symbol", Value: symbol})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var results []Document

	opts := options.Find().SetLimit(int64(limit)).SetSkip(int64(skip)).SetSort(bson.D{{Key: sort, Value: sortOrder}})
	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		log.Printf("Failed to find documents: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find documents"})
		return
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var doc Document
		err := cursor.Decode(&doc)
		if err != nil {
			log.Printf("Failed to decode document: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode document"})
			return
		}
		results = append(results, doc)
	}

	if err := cursor.Err(); err != nil {
		log.Printf("Cursor error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Cursor error"})
		return
	}

	total, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		log.Printf("Failed to count documents: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count documents"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  results,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

func getUniqueSymbolsHandler(c *gin.Context) {
	if client == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "MongoDB client is not initialized"})
		return
	}
	collection := client.Database("StockThaiAnalysis").Collection("predict")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := c.Query("query")

	filter := bson.M{}
	if query != "" {
		filter = bson.M{"Symbol": bson.M{"$regex": query, "$options": "i"}}
	}

	symbols, err := collection.Distinct(ctx, "Symbol", filter)
	if err != nil {
		log.Printf("Failed to get distinct symbols: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get distinct symbols"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"symbols": symbols,
	})
}
