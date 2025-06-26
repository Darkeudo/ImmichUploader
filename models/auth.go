package models

import (
	"time"
)


// Structure
type FormData struct {
	Correo     string `form:"correo" binding:"required"`
	Contrasena string `form:"contrasena" binding:"required"`
	Host       string `form:"host" binding:"required"`
}

// Structure form Request

type RequestForm struct {
	Ruta string `form:"Ruta" binding:"required"`
}


// StructureAPI
type AuthResponse struct {
	AccessToken string `json:"accessToken"`
	UserID      string `json:"userId"`
	UserEmail   string `json:"userEmail"`
	Name        string `json:"name"`
	IsAdmin     bool   `json:"isAdmin"`
}


// Database

type Credenciales struct {
    ID        uint   `gorm:"primaryKey"`
    Correo    string `gorm:"not null"`
    Host      string `gorm:"not null"`
    Contrasena string `gorm:"not null"`
    Token      string `gorm:"not null"`
}


type Request struct {
	ID     uint      `gorm:"primaryKey"`
	Path   string    `gorm:"not null"`
	Status int       `gorm:"default:0"` // 0 Pending, 1 In Progress, 2 Completed, 3 Upload, 4 Finish, 5 Restart, 6 Lock
	Date   time.Time `gorm:"not null"`
}


type ExtractedFile struct {
    ID        uint      `gorm:"primaryKey"`
    RequestID uint      `gorm:"not null"`
    Path      string    `gorm:"not null"`
    Status    int       `gorm:"default:0"`  // 0 In Progress, 1 Completed, 2 Duplicate, 3 Error, 4 Done
    Date      time.Time `gorm:"not null"`
    CheckSum  string    `gorm:"not null"`  
}