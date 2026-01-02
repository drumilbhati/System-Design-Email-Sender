package store

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoStore struct {
	client     *mongo.Client
	collection *mongo.Collection
}

type Subscriber struct {
	Email     string    `bson:"email"`
	CreatedAt time.Time `bson:"created_at"`
}

func NewMongoStore(uri string) (*MongoStore, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOptions := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, err
	}

	// Verify connection
	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}

	collection := client.Database("system_design_mailer").Collection("subscribers")
	
	// Create unique index on email
	mod := mongo.IndexModel{
		Keys:    bson.M{"email": 1},
		Options: options.Index().SetUnique(true),
	}
	_, err = collection.Indexes().CreateOne(ctx, mod)
	if err != nil {
		return nil, err
	}

	return &MongoStore{
		client:     client,
		collection: collection,
	}, nil
}

func (s *MongoStore) Add(email string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sub := Subscriber{
		Email:     email,
		CreatedAt: time.Now(),
	}

	// Use UpdateOne with Upsert to be safe, or InsertOne and ignore duplicate error.
	// UpdateOne is cleaner for idempotency.
	filter := bson.M{"email": email}
	update := bson.M{"$setOnInsert": sub}
	opts := options.Update().SetUpsert(true)

	_, err := s.collection.UpdateOne(ctx, filter, update, opts)
	return err
}

func (s *MongoStore) GetAll() ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var results []string
	
	// Projection to return only email
	cursor, err := s.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var sub Subscriber
		if err := cursor.Decode(&sub); err != nil {
			continue
		}
		results = append(results, sub.Email)
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return results, nil
}
