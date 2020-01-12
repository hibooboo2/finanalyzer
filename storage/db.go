package storage

import (
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
)

type DB struct {
	*gorm.DB
}

func New(models ...interface{}) (*DB, error) {
	db, err := gorm.Open("postgres", "host=192.168.1.29 port=8989 user=postgres dbname=finanalyzer password=root sslmode=disable")
	if err != nil {
		return nil, errors.Wrap(err, "failed to open db")
	}
	err = db.AutoMigrate(models...).Error
	if err != nil {
		return nil, errors.Wrapf(err, "failed to migrate models")
	}
	db = db.Debug()
	return &DB{db}, nil
}

func MustNew(models ...interface{}) *DB {
	db, err := New(models)
	if err != nil {
		panic(fmt.Sprintf("%+v", err))
	}
	return db
}

//docker run -d --name finanalyzer -e POSTGRES_PASSWORD=root -e POSTGRES_DB=finanalyzer -p 8989:5432 postgres:10.7

func (db *DB) MustSave(v interface{}) {
	if err := db.Save(v).Error; err != nil {
		panic(err)
	}
}
