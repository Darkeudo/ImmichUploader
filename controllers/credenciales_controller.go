package controllers

import (
    "immichUploader/database"
    "immichUploader/models"
    "errors"
    
)

func SaveCredentials(cred models.Credenciales) error {

    if database.DB == nil {
        return errors.New("database is not initialized")
    }
   
    //database.DB.Where("correo = ?", cred.Correo).Delete(&models.Credenciales{})
    
    return database.DB.Create(&cred).Error
}



func GetCredentials() (models.Credenciales, error) {
    
    var cred models.Credenciales
    if database.DB == nil {
        return cred, errors.New("database is not initialized")
    }

    result := database.DB.Order("id DESC").First(&cred)
    if result.Error != nil {
        return cred, result.Error
    }

    return cred, nil
}
