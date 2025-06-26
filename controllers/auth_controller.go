package controllers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"immichUploader/models"
	"immichUploader/database"
	"net/http"
	"time"

)


func GetToken(formData models.FormData) (string, error) {
	
	payload, _ := json.Marshal(map[string]string{
		"email":    formData.Correo,
		"password": formData.Contrasena,
	})

	apiURL := formData.Host + "/api/auth/login"
	fmt.Println("Sending request to:", apiURL)

	req, _ := http.NewRequest("POST", apiURL, bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("HTTP request error:", err)
		return "", fmt.Errorf("HTTP request error: %v", err)
	}
	defer resp.Body.Close()

	
	var authResp models.AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		fmt.Println("Error decoding JSON response:", err)
		return "", fmt.Errorf("error decoding JSON response: %v", err)
	}

	// Check if the token was received
	if authResp.AccessToken == "" {
		fmt.Println("Error: Empty token in response")
		return "", errors.New("authentication error: empty token")
	}

	fmt.Println("Token successfully obtained:", authResp.AccessToken)
	return authResp.AccessToken, nil
}

func ValidateToken(host, token string) (bool, error) {

	url := fmt.Sprintf("%s/api/users/me", host)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		
		return false, nil
	} else if resp.StatusCode != http.StatusOK {
	
		return false, fmt.Errorf("Error HTTP: %d", resp.StatusCode)
	}
	
	return true, nil
}

func DeleteSessionToken() {

	var cred models.Credenciales	
	result := database.DB.Order("id DESC").First(&cred)
	if result.Error != nil {		
		return
	}

	cred.Token = ""
	database.DB.Save(&cred)	

}