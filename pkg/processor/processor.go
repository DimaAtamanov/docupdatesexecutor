package processor

import (
	"context"
	"docupdatesexecutor/pkg/connector"
	"docupdatesexecutor/pkg/structures"
	"errors"
)

// Определение интерфейса Processor, который включает метод Process
type Processor interface {
	Process(doc *structures.TDocument) (*structures.TDocument, error)
}

// Структура DocumentProcessor, реализующая интерфейс Processor
type DocumentProcessor struct {
	dbConnector       connector.DBConnector
	cacheConnector    connector.CacheConnector
	eventBusConnector connector.EventBusConnector
}

// Конструктор NewDocumentProcessor для создания нового экземпляра DocumentProcessor
func NewDocumentProcessor(db connector.DBConnector, cache connector.CacheConnector, eventBus connector.EventBusConnector) Processor {
	return &DocumentProcessor{
		dbConnector:       db,
		cacheConnector:    cache,
		eventBusConnector: eventBus,
	}
}

// Метод Process для обработки документа
func (p *DocumentProcessor) Process(doc *structures.TDocument) (*structures.TDocument, error) {
	ctx := context.Background()

	// Сначала проверяем кэш
	cachedDoc, err := p.cacheConnector.Get(ctx, doc.Url)
	if err != nil && err != connector.ErrCacheMiss {
		return nil, err
	}

	if cachedDoc != nil {
		// Обновляем информацию о документе в кэше
		updatedDoc := p.updateDocumentFields(cachedDoc, doc)
		if err := p.cacheConnector.Set(ctx, doc.Url, updatedDoc, 0); err != nil {
			return nil, err
		}
		return updatedDoc, nil
	}

	// Если в кэше нет, проверяем базу данных
	dbDoc, err := p.dbConnector.GetDocument(doc.Url)
	if err != nil {
		if errors.Is(err, connector.ErrDocumentNotFound) {
			// Добавляем новый документ в БД и кэш
			if err := p.dbConnector.AddDocument(doc); err != nil {
				return nil, err
			}
			if err := p.cacheConnector.Set(ctx, doc.Url, doc, 0); err != nil {
				return nil, err
			}
			return doc, nil
		}
		return nil, err
	}

	// Обновляем существующий документ
	updatedDoc := p.updateDocumentFields(dbDoc, doc)
	if err := p.dbConnector.UpdateDocument(updatedDoc); err != nil {
		return nil, err
	}
	if err := p.cacheConnector.Set(ctx, doc.Url, updatedDoc, 0); err != nil {
		return nil, err
	}

	return updatedDoc, nil
}

// Метод updateDocumentFields для обновления полей документа по заданным правилам
func (p *DocumentProcessor) updateDocumentFields(existingDoc, newDoc *structures.TDocument) *structures.TDocument {
	if newDoc.FetchTime > existingDoc.FetchTime {
		existingDoc.Text = newDoc.Text
		existingDoc.FetchTime = newDoc.FetchTime
	}
	if newDoc.FetchTime < existingDoc.FetchTime {
		existingDoc.PubDate = newDoc.PubDate
		existingDoc.FirstFetchTime = newDoc.FetchTime
	}
	return existingDoc
}
