package parser

import (
	"context"
	"crypto/md5"
	"fmt"
	"net/url"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Document struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"`
	URL         string            `bson:"url"`
	RawHTML     string            `bson:"raw_html"`
	Source      string            `bson:"source"`
	CrawlTime   int64             `bson:"crawl_time"`
	HTMLHash    string            `bson:"html_hash"`
	LastChecked int64             `bson:"last_checked"`
}

type Database struct {
	client     *mongo.Client
	collection *mongo.Collection
	ctx        context.Context
}

func NewDatabase(uri, dbName, collectionName string) (*Database, error) {
	ctx := context.Background()

	clientOptions := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	db := client.Database(dbName)
	collection := db.Collection(collectionName)

	indexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: "url", Value: 1}},
		Options: options.Index().SetUnique(true),
	}
	_, err = collection.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
	}

	return &Database{
		client:     client,
		collection: collection,
		ctx:        ctx,
	}, nil
}

func (db *Database) Close() error {
	return db.client.Disconnect(db.ctx)
}

func NormalizeURL(urlStr string) (string, error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return "", err
	}

	u.Fragment = ""

	u.Scheme = "https"
	if u.Host == "" {
		u.Host = u.Scheme
	}

	if len(u.Path) > 1 && u.Path[len(u.Path)-1] == '/' {
		u.Path = u.Path[:len(u.Path)-1]
	}

	return u.String(), nil
}

func computeHTMLHash(html string) string {
	hash := md5.Sum([]byte(html))
	return fmt.Sprintf("%x", hash)
}

func (db *Database) SaveDocument(normalizedURL, rawHTML, source string) error {
	htmlHash := computeHTMLHash(rawHTML)
	crawlTime := time.Now().Unix()

	filter := bson.M{"url": normalizedURL}
	update := bson.M{
		"$set": bson.M{
			"raw_html":     rawHTML,
			"source":       source,
			"crawl_time":   crawlTime,
			"html_hash":    htmlHash,
			"last_checked": crawlTime,
		},
	}

	opts := options.Update().SetUpsert(true)
	_, err := db.collection.UpdateOne(db.ctx, filter, update, opts)
	return err
}

func (db *Database) DocumentExists(normalizedURL string) (bool, error) {
	filter := bson.M{"url": normalizedURL}
	count, err := db.collection.CountDocuments(db.ctx, filter)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (db *Database) GetDocument(normalizedURL string) (*Document, error) {
	var doc Document
	filter := bson.M{"url": normalizedURL}
	err := db.collection.FindOne(db.ctx, filter).Decode(&doc)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &doc, nil
}

func (db *Database) HasDocumentChanged(normalizedURL, newHTML string) (bool, error) {
	doc, err := db.GetDocument(normalizedURL)
	if err != nil {
		return false, err
	}
	if doc == nil {
		return true, nil
	}

	if newHTML == "" {
		return true, nil
	}

	newHash := computeHTMLHash(newHTML)
	return doc.HTMLHash != newHash, nil
}

func (db *Database) GetDocumentsForReCrawl(reCrawlInterval int) ([]Document, error) {
	if reCrawlInterval <= 0 {
		return nil, nil
	}

	cutoffTime := time.Now().Unix() - int64(reCrawlInterval)
	filter := bson.M{
		"last_checked": bson.M{
			"$lt": cutoffTime,
		},
	}

	cursor, err := db.collection.Find(db.ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(db.ctx)

	var docs []Document
	if err := cursor.All(db.ctx, &docs); err != nil {
		return nil, err
	}

	return docs, nil
}

func (db *Database) UpdateLastChecked(normalizedURL string) error {
	filter := bson.M{"url": normalizedURL}
	update := bson.M{
		"$set": bson.M{
			"last_checked": time.Now().Unix(),
		},
	}
	_, err := db.collection.UpdateOne(db.ctx, filter, update)
	return err
}

func (db *Database) GetLastProcessedURL() (string, error) {
	opts := options.FindOne().SetSort(bson.D{{Key: "crawl_time", Value: -1}})
	var doc Document
	err := db.collection.FindOne(db.ctx, bson.M{}, opts).Decode(&doc)
	if err == mongo.ErrNoDocuments {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return doc.URL, nil
}

