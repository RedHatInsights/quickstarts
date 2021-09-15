package database

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	// Timeout operations after N seconds
	connectTimeout           = 5
	connectionStringTemplate = "mongodb://%s:%s@%s"
)

func GetConnection() (*mongo.Client, context.Context, context.CancelFunc) {
	username := os.Getenv("MONGO_INITDB_ROOT_USERNAME")
	password := os.Getenv("MONGO_INITDB_ROOT_PASSWORD")
	clusterEndpoint := os.Getenv("MONGO_INITDB_ENDPOINT")

	connectionURI := fmt.Sprintf(connectionStringTemplate, username, password, clusterEndpoint)
	fmt.Printf("connection %v, username %v, password %v, endpoint %v\n", connectionURI, username, password, clusterEndpoint)
	client, err := mongo.NewClient(options.Client().ApplyURI(connectionURI))
	if err != nil {
		logrus.Errorf("Failed to create client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), connectTimeout*time.Second)
	err = client.Connect(ctx)
	if err != nil {
		logrus.Error("Failed to connect to cluster: %v", err)
	}

	// Force a connection to verify our connection string
	err = client.Ping(ctx, nil)
	if err != nil {
		logrus.Error("Failed to ping cluster: %v", err)
	}

	logrus.Info("Connected to MongoDB")
	return client, ctx, cancel

}
