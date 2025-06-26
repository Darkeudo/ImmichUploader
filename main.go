package main

import (

	"log"
	"immichUploader/routes"
	"immichUploader/database"

	"os/exec"
	"runtime"
	"time"

	"strings"
	"net/http"
	"github.com/gin-gonic/gin"
	"bytes"
	"os"
	"io"


)


func selectFolder(c *gin.Context) {

	var ruta string
	var err error

	switch runtime.GOOS {
	case "windows":
		ruta, err = selectFolderWindows()
	case "darwin": // macOS
		ruta, err = selectFolderMac()
	default: 
		ruta, err = selectFolderLinux()
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not select the folder."})
		return
	}

	ruta = strings.TrimSpace(ruta)

	c.JSON(http.StatusOK, gin.H{"ruta": ruta})
}


func selectFolderWindows() (string, error) {

	out, err := exec.Command("powershell", "-Command", "(New-Object -ComObject Shell.Application).BrowseForFolder(0, 'Select Folder', 0).Self.Path").Output()
	return string(out), err
}


func selectFolderLinux() (string, error) {

	cmd := exec.Command("zenity", "--file-selection", "--directory", "--title=Select Folder")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	return out.String(), err
}


func selectFolderMac() (string, error) {

	cmd := exec.Command("osascript", "-e", `tell application "Finder" to set folderPath to choose folder as string`)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	return out.String(), err
}




func main() {

	database.InitDB()

	logFile, err := os.OpenFile("app.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
    if err != nil {
        log.Fatal(err)
    }
    defer logFile.Close()

    log.SetOutput(logFile) 
    log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)  

    // Gin "release"
    gin.SetMode(gin.ReleaseMode)
    gin.DefaultWriter = io.Discard
	gin.DefaultErrorWriter = io.Discard
	
	r := routes.SetupRouter()

	r.GET("/select-folder", selectFolder)

	url := "http://localhost:8081"
	
	go func() {
		time.Sleep(1 * time.Second) 
		openBrowser(url)
	}()

	log.Println("Server running at", url)

	r.Run(":8081")

}


func openBrowser(url string) {

	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start", url}
	case "darwin":
		cmd = "open"
		args = []string{url}
	default: 
		cmd = "xdg-open"
		args = []string{url}
	}

	if err := exec.Command(cmd, args...).Start(); err != nil {
		log.Println("Could not open the browser:", err)
	}

}
