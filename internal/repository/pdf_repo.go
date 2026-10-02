package repository

import (
	"context"
	"errors"

	"github.com/PARSE-DOCUMENT-FAST/pdf-extractor/internal/models"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var ErrEmptyPdfID = errors.New("repository: pdf_id no puede estar vacio")

type PdfRepository interface {
	Upsert(ctx context.Context, doc models.PdfDocument) error
	EnsureIndex(ctx context.Context) error
}

type MongoPdfRepository struct {
	collection *mongo.Collection
}

func NewMongoPdfRepository(db *mongo.Database, collectionName string) *MongoPdfRepository {
	return &MongoPdfRepository{collection: db.Collection(collectionName)}
}

func (r *MongoPdfRepository) Upsert(ctx context.Context, doc models.PdfDocument) error {
	if doc.PdfID == "" {
		return ErrEmptyPdfID
	}

	_, err := r.collection.ReplaceOne(ctx, bson.M{"pdf_id": doc.PdfID}, doc, options.Replace().SetUpsert(true))
	return err
}

func (r *MongoPdfRepository) EnsureIndex(ctx context.Context) error {
	model := mongo.IndexModel{
		Keys:    bson.D{{Key: "pdf_id", Value: 1}},
		Options: options.Index().SetUnique(true),
	}

	_, err := r.collection.Indexes().CreateOne(ctx, model)
	return err
}
