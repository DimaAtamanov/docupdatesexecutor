package connector

import (
	"docupdatesexecutor/pkg/structures"
	"errors"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var ErrDocumentNotFound = errors.New("document not found")

type DBConnector interface {
	AddDocument(doc *structures.TDocument) error
	UpdateDocument(doc *structures.TDocument) error
	GetDocument(url string) (*structures.TDocument, error)
}

type GormDBConnector struct {
	db *gorm.DB
}

func NewDBConnector(DSN string) (DBConnector, error) {
	db, err := gorm.Open(postgres.Open(DSN), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	db.AutoMigrate(&structures.TDocument{})
	return &GormDBConnector{db: db}, nil
}

func (g *GormDBConnector) AddDocument(doc *structures.TDocument) error {
	return g.db.Create(doc).Error
}

func (g *GormDBConnector) UpdateDocument(doc *structures.TDocument) error {
	return g.db.Save(doc).Error
}

func (g *GormDBConnector) GetDocument(url string) (*structures.TDocument, error) {
	var doc structures.TDocument
	err := g.db.First(&doc, "url = ?", url).Error
	return &doc, err
}
