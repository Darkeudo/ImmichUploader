package controllers


import (
    "bytes"
    "crypto/sha1"
    "fmt"
    "io"
    "os"
    "path/filepath"
    "time"
    "encoding/json"
    "net/http"

    "immichUploader/config"   
    "immichUploader/models"
    "immichUploader/database"

    "log"
    "io/ioutil"
    "strconv"

    "mime/multipart"

    "github.com/rwcarlsen/goexif/exif"

)

type Metadata struct {
    FileCreatedAt  string  `json:"fileCreatedAt"`
    FileModifiedAt string  `json:"fileModifiedAt"`
    Latitude       float64 `json:"latitude"`
    Longitude      float64 `json:"longitude"`
    Altitude       float64 `json:"altitude"`
    DeviceId       string  `json:"deviceId"`
}

func calculateChecksum(filePath string) (string, error) {
    file, err := os.Open(filePath)
    if err != nil {
        return "", err
    }
    defer file.Close()

    sha1Hash := sha1.New()
    if _, err := io.Copy(sha1Hash, file); err != nil {
        return "", err
    }

    return fmt.Sprintf("%x", sha1Hash.Sum(nil)), nil
}

func checkFileInImmich(checksum string) (bool, error) {
    url := config.GlobalHost + "/api/assets/bulk-upload-check"
    requestBody := fmt.Sprintf(`{"assets": [{"id": "%s", "checksum": "%s"}]}`, checksum, checksum)
    req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(requestBody)))
    if err != nil {
        return false, err
    }

    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+config.GlobalToken)

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return false, err
    }
    defer resp.Body.Close()

    respBody, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        return false, fmt.Errorf("Error reading response body: %v", err)
    }

    log.Println("Immich Response:", string(respBody))

    var data map[string]interface{}
    if err := json.Unmarshal(respBody, &data); err != nil {
        return false, fmt.Errorf("Error parsing the response JSON: %v", err)
    }

    if results, ok := data["results"].([]interface{}); ok && len(results) > 0 {
        if firstItem, ok := results[0].(map[string]interface{}); ok {
            if reason, ok := firstItem["reason"].(string); ok && reason == "duplicate" {
                return true, nil
            }
            if action, ok := firstItem["action"].(string); ok && action == "accept" {
                return false, nil
            }
        }
    }

    return false, fmt.Errorf("Unexpected response from Immich: %s", string(respBody))
}

func findJsonForImage(filePath string) (string, error) {
    baseName := filepath.Base(filePath)
    fileNameWithoutExt := baseName[:len(baseName)-len(filepath.Ext(baseName))]
    dir := filepath.Dir(filePath)

    pattern := filepath.Join(dir, fileNameWithoutExt+"*.json")
    jsonFiles, err := filepath.Glob(pattern)
    if err != nil {
        return "", fmt.Errorf("Error searching for JSON files: %v", err)
    }

    if len(jsonFiles) == 0 {
        return "", fmt.Errorf("JSON file not found")
    }

    return jsonFiles[0], nil
}

func extractMetadataFromJson(jsonPath string) (*Metadata, error) {
    file, err := os.Open(jsonPath)
    if err != nil {
        return nil, err
    }
    defer file.Close()

    var data map[string]interface{}
    decoder := json.NewDecoder(file)
    if err := decoder.Decode(&data); err != nil {
        return nil, err
    }

    metadata := &Metadata{}

    if creationTime, ok := data["creationTime"].(map[string]interface{}); ok {
        if timestampStr, ok := creationTime["timestamp"].(string); ok {
            timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
            if err == nil {
                metadata.FileCreatedAt = timestampToDateString(timestamp)
                metadata.FileModifiedAt = metadata.FileCreatedAt
            }
        }
    }

    if geoData, ok := data["geoData"].(map[string]interface{}); ok {
        metadata.Latitude = geoData["latitude"].(float64)
        metadata.Longitude = geoData["longitude"].(float64)
        metadata.Altitude = geoData["altitude"].(float64)
    }

    if origin, ok := data["googlePhotosOrigin"].(map[string]interface{}); ok {
        if mobileUpload, ok := origin["mobileUpload"].(map[string]interface{}); ok {
            metadata.DeviceId = mobileUpload["deviceType"].(string)
        }
    }

    return metadata, nil
}

func extractMetadataFromImage(path string) (*Metadata, error) {
    f, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    defer f.Close()

    x, err := exif.Decode(f)
    if err != nil {
        return nil, fmt.Errorf("no EXIF metadata: %v", err)
    }

    metadata := &Metadata{}

    if dt, err := x.DateTime(); err == nil {
        metadata.FileCreatedAt = dt.UTC().Format("2006-01-02T15:04:05.000Z")
        metadata.FileModifiedAt = metadata.FileCreatedAt
    }

    if model, err := x.Get(exif.Model); err == nil {
        if val, err := model.StringVal(); err == nil {
            metadata.DeviceId = val
        }
    }

    lat, lon, _ := x.LatLong()
    metadata.Latitude = lat
    metadata.Longitude = lon

    return metadata, nil
}

func timestampToDateString(timestamp int64) string {
    t := time.Unix(timestamp, 0).UTC()
    return t.Format("2006-01-02T15:04:05.000Z")
}

func uploadFileWithMetadata(filePath string, metadatos *Metadata, checksum string) error {
    url := config.GlobalHost + "/api/assets"
    headers := map[string]string{
        "Authorization":     fmt.Sprintf("Bearer %s", config.GlobalToken),
        "x-immich-checksum": checksum,
    }

    file, err := os.Open(filePath)
    if err != nil {
        return fmt.Errorf("Error opening the file: %v", err)
    }
    defer file.Close()

    body := &bytes.Buffer{}
    writer := multipart.NewWriter(body)

    part, _ := writer.CreateFormFile("assetData", filepath.Base(filePath))
    _, _ = io.Copy(part, file)

    writer.WriteField("deviceAssetId", "Dark")

    if metadatos != nil {
        writer.WriteField("deviceId", metadatos.DeviceId)
        writer.WriteField("fileCreatedAt", metadatos.FileCreatedAt)
        writer.WriteField("fileModifiedAt", metadatos.FileModifiedAt)
    } else {
        now := time.Now().Format("2006-01-02T15:04:05.000Z")
        writer.WriteField("deviceId", "default")
        writer.WriteField("fileCreatedAt", now)
        writer.WriteField("fileModifiedAt", now)
    }

    writer.WriteField("isFavorite", "false")
    writer.WriteField("isVisible", "true")

    writer.Close()

    req, err := http.NewRequest("POST", url, body)
    if err != nil {
        return fmt.Errorf("Error creating the request: %v", err)
    }

    for key, value := range headers {
        req.Header.Add(key, value)
    }
    req.Header.Set("Content-Type", writer.FormDataContentType())

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return fmt.Errorf("Error making the request: %v", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode == http.StatusCreated {
        return nil
    } else {
        bodyResp, _ := ioutil.ReadAll(resp.Body)
        return fmt.Errorf("Error uploading the file: %v", string(bodyResp))
    }
}

func ProcessFilesController() error {
    db := database.DB

    var files []models.ExtractedFile
    err := db.Where("status = 0").Find(&files).Error
    if err != nil {
        return fmt.Errorf("Error retrieving pending files")
    }

    for _, file := range files {
        checksum, err := calculateChecksum(file.Path)
        if err != nil {
            file.Status = 3
            log.Printf("Error generating checksum for %s: %v", file.Path, err)
            _ = db.Save(&file)
            continue
        }

        log.Println("File processing:", filepath.Base(file.Path))

        exists, err := checkFileInImmich(checksum)
        if err != nil {
            file.Status = 3
            log.Printf("Error checking Immich for %s: %v", file.Path, err)
            _ = db.Save(&file)
            continue
        }

        if exists {
            file.Status = 2
            log.Println("Duplicate found, skipping upload")
            _ = db.Save(&file)
            continue
        }

        var metadatos *Metadata = nil
        jsonPath, err := findJsonForImage(file.Path)
        if err == nil {
            metadatos, err = extractMetadataFromJson(jsonPath)
            if err != nil {
                log.Println("Error parsing JSON metadata, trying EXIF")
                metadatos, err = extractMetadataFromImage(file.Path)
            }
        } else {
            log.Println("JSON not found, trying EXIF metadata")
            metadatos, err = extractMetadataFromImage(file.Path)
        }

        err = uploadFileWithMetadata(file.Path, metadatos, checksum)
        if err != nil {
            file.Status = 3
            log.Printf("Error uploading file %s: %v", file.Path, err)
            _ = db.Save(&file)
        } else {
            file.Status = 4
            _ = db.Save(&file)
            log.Println("File Uploaded Successfully")
        }
    }

    return nil
}