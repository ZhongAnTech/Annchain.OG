// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package ogdb

import (
	"context"
	"errors"
	"fmt"
	"github.com/annchain/OG/common"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"sync"
	"time"
)

/*
 * MongoDB operator
 */
type MongoDBDatabase struct {
	config MongoDBConfig
	lock   sync.RWMutex
	client *mongo.Client
	coll   *mongo.Collection
}
type MongoDBConfig struct {
	Uri        string
	Database   string
	Collection string
	UserName   string
	Password   string
}

func NewMongoDBDatabase(config MongoDBConfig) (*MongoDBDatabase, error) {
	database := &MongoDBDatabase{
		config: config,
	}
	err := database.ConnectMongoDB()
	return database, err
}

func (db *MongoDBDatabase) ConnectMongoDB() (err error) {
	ctx, _ := context.WithTimeout(context.Background(), 3*time.Second)

	// user Connection database

	// Set client options
	clientOptions := options.Client().ApplyURI(fmt.Sprintf("%s", db.config.Uri)).SetAuth(options.Credential{
		//AuthMechanism:           "",
		//AuthMechanismProperties: nil,
		AuthSource: db.config.Database,
		Username:   db.config.UserName,
		Password:   db.config.Password,
		//PasswordSet:             false,
	})

	// Connect to MongoDB
	client, err := mongo.Connect(ctx, clientOptions)

	if err != nil {
		logrus.Warn(err)
	}

	db.coll = client.Database(db.config.Database).Collection(db.config.Collection)

	// Check the connection
	err = client.Ping(ctx, nil)

	if err != nil {
		logrus.Warn(err)
	}
	return nil
}

func (db *MongoDBDatabase) Put(key []byte, value []byte) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	ctx, _ := context.WithTimeout(context.Background(), 3*time.Second)
	filter := bson.M{
		"_id": string(key),
	}
	updated := bson.M{
		"value": string(value),
	}
	opt := &options.ReplaceOptions{}
	opt.SetUpsert(true)

	_, err := db.coll.ReplaceOne(ctx, filter, updated, opt)
	return err
}

func (db *MongoDBDatabase) Has(key []byte) (bool, error) {
	v, err := db.Get(key)
	if err != nil {
		return false, err
	}
	return v != nil, nil
}

func (db *MongoDBDatabase) Get(key []byte) ([]byte, error) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	ctx, _ := context.WithTimeout(context.Background(), 3*time.Second)
	filter := bson.M{
		"_id": string(key),
	}
	v := db.coll.FindOne(ctx, filter)
	if v.Err() != nil {
		return nil, v.Err()
	}
	doc := bson.M{}

	err := v.Decode(&doc)
	if err == mongo.ErrNoDocuments {
		return nil, errors.New("not found")
	}
	if err != nil {
		return nil, err
	}
	return []byte(doc["value"].(string)), nil
}

func (db *MongoDBDatabase) Delete(key []byte) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	ctx, _ := context.WithTimeout(context.Background(), 3*time.Second)
	filter := bson.M{
		"_id": string(key),
	}
	_, err := db.coll.DeleteOne(ctx, filter)
	return err
}

func (db *MongoDBDatabase) Close() {
	ctx, _ := context.WithTimeout(context.Background(), 3*time.Second)
	_ = db.client.Disconnect(ctx)
}

func (db *MongoDBDatabase) NewBatch() Batch {
	return &mongoBatch{db: db}
}

type mongoBatch struct {
	db     *MongoDBDatabase
	writes []kv
	size   int
}

func (b *mongoBatch) Put(key, value []byte) error {
	b.writes = append(b.writes, kv{common.CopyBytes(key), common.CopyBytes(value)})
	b.size += len(value)
	return nil
}

func (b *mongoBatch) Write() error {
	for _, kv := range b.writes {
		err := b.db.Put(kv.k, kv.v)
		if err != nil {
			logrus.WithError(err).Warn("mongo batch write error")
		}
	}
	return nil
}

func (b *mongoBatch) ValueSize() int {
	return b.size
}

func (b *mongoBatch) Reset() {
	b.writes = b.writes[:0]
	b.size = 0
}
