package database

import (
	"github.com/RedHatInsights/quickstarts/pkg/models"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CreateQuickstart is a function that creates new quickstart entry in the DB
func CreateQuickstart(quickstart *models.Quickstart) (primitive.ObjectID, error) {
	client, ctx, cancel := GetConnection()
	defer cancel()
	defer client.Disconnect(ctx)

	quickstart.ID = primitive.NewObjectID()
	result, err := client.Database("quickstarts").Collection("quickstarts").InsertOne(ctx, quickstart)

	if err != nil {
		logrus.Error("Could not create quickstart %v", err)
		return primitive.NilObjectID, err
	}

	oid := result.InsertedID.(primitive.ObjectID)
	return oid, nil
}

// GetQuickstarts list all avaiable quickstarts
func GetQuickstarts() ([]models.Quickstart, error) {
	client, ctx, cancel := GetConnection()
	defer cancel()
	defer client.Disconnect(ctx)
	quickstarts := make([]models.Quickstart, 0)

	result, err := client.Database("quickstarts").Collection("quickstarts").Find(ctx, bson.D{})
	if err != nil {
		logrus.Error("Could not find quickstarts %v", err)
		return quickstarts, err
	}

	defer result.Close(ctx)
	for result.Next(ctx) {
		var entry models.Quickstart
		err := result.Decode(&entry)
		if err != nil {
			logrus.Error("Could not process quickstart", err)
		}
		quickstarts = append(quickstarts, entry)

	}
	return quickstarts, nil
}
