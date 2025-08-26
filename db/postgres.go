package db

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"time"

	eve "eve.evalgo.org/common"
)

type RabbitLog struct {
	gorm.Model
	DocumentID string
	State      string
	Version    string
	// Log        []byte `gorm:"type:bytea"`
	Log []byte `gorm:"type:text"`
}

func PGInfo(pgUrl string) {
	db, err := gorm.Open(postgres.Open(pgUrl), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	sqlDB, err := db.DB()
	// SetMaxIdleConns sets the maximum number of connections in the idle connection pool.
	sqlDB.SetMaxIdleConns(10)
	// SetMaxOpenConns sets the maximum number of open connections to the database.
	sqlDB.SetMaxOpenConns(100)
	// SetConnMaxLifetime sets the maximum amount of time a connection may be reused.
	sqlDB.SetConnMaxLifetime(time.Hour)
	fmt.Println(sqlDB)
	var tables []string
	if err := db.Table("information_schema.tables").Where("table_schema = ?", "public").Pluck("table_name", &tables).Error; err != nil {
		panic(err)
	}
	fmt.Println("Successfully connected!", tables)
}

func PGMigrations(pgUrl string) {
	db, err := gorm.Open(postgres.Open(pgUrl), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	db.AutoMigrate(&RabbitLog{})
}

func PGRabbitLogNew(pgUrl, documentId, state, version string) {
	db, err := gorm.Open(postgres.Open(pgUrl), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	db.Create(&RabbitLog{DocumentID: documentId, State: state, Version: version})
}

func PGRabbitLogList(pgUrl string) {
	db, err := gorm.Open(postgres.Open(pgUrl), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	var logs []RabbitLog
	logsRes := db.Find(&logs)
	if logsRes.Error != nil {
		eve.Logger.Error(err)
	}
	for _, logEntry := range logs {
		eve.Logger.Info(logEntry, " => ", string(logEntry.Log))
	}
}

func PGRabbitLogFormatList(pgUrl string, format string) interface{} {
	db, err := gorm.Open(postgres.Open(pgUrl), &gorm.Config{})
	if err != nil {
		eve.Logger.Error(err)
		return nil
	}
	var logs []RabbitLog
	logsRes := db.Find(&logs)
	if logsRes.Error != nil {
		eve.Logger.Error(err)
		return nil
	}
	if format == "application/json" {
		logsJSON, err := json.Marshal(logs)
		if err != nil {
			eve.Logger.Error(err)
			return nil
		}
		return logsJSON
	}
	if format == "struct" {
		return logs
	}
	eve.Logger.Error("unsupported format ", format)
	return nil
}

func PGRabbitLogUpdate(pgUrl, documentId, state string, logText []byte) {
	db, err := gorm.Open(postgres.Open(pgUrl), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	db.Model(&RabbitLog{}).Where("document_id = ?", documentId).Updates(map[string]interface{}{"state": state, "log": base64.StdEncoding.EncodeToString(logText)})
}
