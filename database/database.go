package database

import (
    "immichUploader/models"
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
    "log"
)


var DB *gorm.DB


func InitDB() {

    var err error
    DB, err = gorm.Open(sqlite.Open("immich.db"), &gorm.Config{})
    if err != nil {
        log.Fatal("Error connecting to the database: ", err)
    }
 
    err = DB.AutoMigrate(&models.Credenciales{},&models.Request{},&models.ExtractedFile{})
    if err != nil {
        log.Fatal("Error in database migration: ", err)
    }

    log.Println("Database initialized successfully")
}

