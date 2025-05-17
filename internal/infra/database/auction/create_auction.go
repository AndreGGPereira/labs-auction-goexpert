package auction

import (
	"context"
	"fullcycle-auction_go/configuration/logger"
	"fullcycle-auction_go/internal/entity/auction_entity"
	"fullcycle-auction_go/internal/internal_error"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"

	"go.mongodb.org/mongo-driver/mongo"
)

type AuctionEntityMongo struct {
	Id          string                          `bson:"_id"`
	ProductName string                          `bson:"product_name"`
	Category    string                          `bson:"category"`
	Description string                          `bson:"description"`
	Condition   auction_entity.ProductCondition `bson:"condition"`
	Status      auction_entity.AuctionStatus    `bson:"status"`
	Timestamp   int64                           `bson:"timestamp"`
}
type AuctionRepository struct {
	Collection *mongo.Collection
}

func NewAuctionRepository(database *mongo.Database) *AuctionRepository {
	return &AuctionRepository{
		Collection: database.Collection("auctions"),
	}
}

// type Repository interface {
// 	closeAuctionUpdate(ctx context.Context, auctionID string) *internal_error.InternalError
// }

func (ar *AuctionRepository) CreateAuction(
	ctx context.Context,
	auctionEntity *auction_entity.Auction) *internal_error.InternalError {
	auctionEntityMongo := &AuctionEntityMongo{
		Id:          auctionEntity.Id,
		ProductName: auctionEntity.ProductName,
		Category:    auctionEntity.Category,
		Description: auctionEntity.Description,
		Condition:   auctionEntity.Condition,
		Status:      auctionEntity.Status,
		Timestamp:   auctionEntity.Timestamp.Unix(),
	}
	_, err := ar.Collection.InsertOne(ctx, auctionEntityMongo)
	if err != nil {
		logger.Error("Falied insert auction", err)
		return internal_error.NewInternalServerError("Error trying to insert auction")
	}

	go func() {

		select {
		case <-time.After(getAuctionInterval()):
			ar.closeAuction(ctx, auctionEntityMongo.Id)
		case <-ctx.Done():
			logger.Info("Context cancelled while trying to close auction", zap.String("auctionId", auctionEntityMongo.Id))
		}

	}()
	return nil
}

func (ar *AuctionRepository) closeAuction(ctx context.Context, auctionID string) *internal_error.InternalError {
	_, err := ar.Collection.UpdateOne(ctx, bson.M{"_id": auctionID}, bson.M{"$set": bson.M{"status": auction_entity.Completed}})
	if err != nil {
		logger.Error("Error trying to update auction status", err)
		return internal_error.NewInternalServerError(err.Error())
	}
	logger.Info("Auction closed", zap.String("auction_id", auctionID))
	return nil
}

func getAuctionInterval() time.Duration {
	auctionDuration := os.Getenv("AUCTION_DURATION")
	duration, err := time.ParseDuration(auctionDuration)
	if err != nil {
		return time.Minute * 1
	}
	return duration
}
