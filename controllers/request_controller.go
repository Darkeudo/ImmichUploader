package controllers

import (
    "fmt"
    "immichUploader/database"
    "immichUploader/models"
    "immichUploader/utils"
    "io"
    "os"
    "path/filepath"
    "archive/zip"
    "time"
    "log"

    "net/http"
    "strconv"
    "github.com/gin-gonic/gin"

    "strings"
    "io/ioutil"


)


func RetryFailedUploads(c *gin.Context) {

    db := database.DB
    requestIDStr := c.Param("id")

    requestID, err := strconv.Atoi(requestIDStr)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request_id"})
        return
    }


    if err := db.Model(&models.ExtractedFile{}).
        Where("request_id = ? AND status = 3", requestID).
        Update("status", 0).Error; err != nil { 
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retry failed uploads"})
        return
    }

    log.Println("Restart request ID:", requestIDStr)
    c.JSON(http.StatusOK, gin.H{"message": "Retrying failed uploads"})

}


func ClearLogs(c *gin.Context) {

    logFile := "app.log"

    file, err := os.OpenFile(logFile, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
    if err != nil {
        log.Println("Error opening log file:", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clear log file"})
        return
    }
    defer file.Close() 

    log.Println("Log file cleared successfully")
    c.JSON(http.StatusOK, gin.H{"message": "Log file cleared successfully"})
}


func GetLogs(c *gin.Context) {

    data, err := ioutil.ReadFile("app.log")
    if err != nil {
        c.String(http.StatusInternalServerError, "Error loading logs")
        return
    }

    lines := strings.Split(string(data), "\n")
    if len(lines) > 20 {
        lines = lines[len(lines)-20:] 
    }

    c.String(http.StatusOK, strings.Join(lines, "\n"))
}

func SaveRequest(req models.Request) error {

    if database.DB == nil {
        return fmt.Errorf("database is not initialized")
    }

    return database.DB.Create(&req).Error
}


func GetAllRequests() ([]models.Request, error) {
    var requests []models.Request
    err := database.DB.Find(&requests).Error
    return requests, err
}


func GetFileProcessingStatus(c *gin.Context) {
    
    db := database.DB
    requestIDStr := c.Param("request_id")

    requestID, err := strconv.Atoi(requestIDStr)
    if err != nil {
        log.Println("Error: request_id inválido:", err)
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request_id"})
        return
    }

    statusCounts := map[int]int{
        0: 0, // Pending
        1: 0, // Processing
        2: 0, // Duplicate
        3: 0, // Error
        4: 0, // Completed
    }


    rows, err := db.Raw("SELECT status, COUNT(*) FROM extracted_files WHERE request_id = ? GROUP BY status", requestID).Rows()
    if err != nil {
        log.Println("Error retrieving file status:", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Error retrieving status"})
        return
    }
    defer rows.Close()

    var status, count int
    total := 0

    for rows.Next() {
        if err := rows.Scan(&status, &count); err == nil {
            statusCounts[status] = count
            total += count
        }
    }

    c.JSON(http.StatusOK, gin.H{
        "request_id": requestID,
        "pending":    statusCounts[0],
        "processing": statusCounts[1],
        "duplicate":  statusCounts[2],
        "error":      statusCounts[3],
        "completed":  statusCounts[4],
        "total":      total, 
    })
}


func ProcessPendingRequests() error {

    var requests []models.Request

    
    if err := database.DB.Where("status = ?", 0).Find(&requests).Error; err != nil {
        return fmt.Errorf("error al obtener solicitudes pendientes: %v", err)
    }

    if len(requests) == 0 {

        log.Println("There are no pending requests")
        return nil
    }
    
    log.Println("Assets process started")

    for _, req := range requests {
     
        if err := database.DB.Model(&req).Update("status", 6).Error; err != nil {
            log.Printf("Error locking the request %d: %v\n", req.ID, err)
            continue
        }

      
        if err := processZipFilesInPath(req.Path, req.ID); err != nil {
            database.DB.Model(&req).Update("status", 5) 
            log.Printf("Error processing the request %d: %v\n", req.ID, err)
            continue
        }

       
        database.DB.Model(&req).Update("status", 1)
    }

    
    log.Println("Assets process completed")

    return nil
}


func processZipFilesInPath(path string, requestID uint) error {
   
    files, err := os.ReadDir(path)
    if err != nil {
        return fmt.Errorf("error al leer la ruta: %v", err)
    }


    for _, file := range files {
        if filepath.Ext(file.Name()) == ".zip" {
            zipPath := filepath.Join(path, file.Name())
            
            err := processZipFile(zipPath, requestID)
            if err != nil {
                return fmt.Errorf("Error processing the ZIP file %s: %v", zipPath, err)
            }
        }
    }

    return nil
}


func processZipFile(zipPath string, requestID uint) error {
  
    zipFile, err := zip.OpenReader(zipPath)
    if err != nil {
        return fmt.Errorf("Error opening the ZIP file: %v", err)
    }
    defer zipFile.Close()

    
    extractDir := filepath.Dir(zipPath)

   
    for _, file := range zipFile.File {
        
        if !utils.IsValidExtension(filepath.Ext(file.Name)) {
            continue 
        }


       
        extractedPath := filepath.Join(extractDir, file.Name)

       
        if file.FileInfo().IsDir() {
            err := os.MkdirAll(extractedPath, 0755) 
            if err != nil {
                return fmt.Errorf("Error creating directory: %v", err)
            }
            continue
        }

       
        err = os.MkdirAll(filepath.Dir(extractedPath), 0755)
        if err != nil {
            return fmt.Errorf("Error creating directories for the file: %v", err)
        }

        
        destFile, err := os.Create(extractedPath)
        if err != nil {
            return fmt.Errorf("Error creating extracted file: %v", err)
        }
        defer destFile.Close()

        sourceFile, err := file.Open()
        if err != nil {
            return fmt.Errorf("Error opening file inside the zip: %v", err)
        }
        defer sourceFile.Close()

        
        _, err = io.Copy(destFile, sourceFile)
        if err != nil {
            return fmt.Errorf("Error copying data from the zip file: %v", err)
        }

        
        err = saveExtractedFileInfo(extractedPath, file.Name, requestID)
        if err != nil {
            return fmt.Errorf("Error saving the extracted file information: %v", err)
        }
    }

    return nil
}



func saveExtractedFileInfo(path, fileName string, requestID uint) error {

    checksum, err := utils.CalculateChecksum(path) 
    if err != nil {
        return fmt.Errorf("Error calculating checksum: %v", err)
    }


    extractedFile := models.ExtractedFile{
        RequestID:requestID,    
        Path:      path,
        Status:    0,   
        Date:      time.Now(),
        CheckSum:  checksum,  
    }
   
    return database.DB.Create(&extractedFile).Error
}
