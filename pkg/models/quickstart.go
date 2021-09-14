package models

import "go.mongodb.org/mongo-driver/bson/primitive"

// Quickstart represents the quickstart json content
type Quickstart struct {
	ID    primitive.ObjectID
	Title string
}
