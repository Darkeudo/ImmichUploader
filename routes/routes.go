package routes

import (
	"immichUploader/config"
	"immichUploader/controllers"
	"immichUploader/models"
	"github.com/gin-gonic/gin"
	"net/http"
	"fmt"
	"log"
	"time"	
)

func SetupRouter() *gin.Engine {
	
	r := gin.Default()

	r.LoadHTMLGlob("templates/*") 

	r.GET("/", func(c *gin.Context) {
		
		cred, err := controllers.GetCredentials()		
		if err != nil {

			c.HTML(http.StatusOK, "form.html", gin.H{"error": "", "correo": "", "host": "", "password": ""})
			return
		}

		valido, err := controllers.ValidateToken(cred.Host, cred.Token)		
		if err != nil {
			fmt.Println("Error validating the token:", err)
		}

		if valido {

			config.GlobalHost = cred.Host
			config.GlobalToken = cred.Token	
			config.GlobalEmail = cred.Correo

			c.HTML(http.StatusOK, "dashboard.html", gin.H{"token": config.GlobalToken})
			return
		}

		
		c.HTML(http.StatusOK, "form.html", gin.H{
			"error":    "Expired token. Please log in again.",
			"correo":   cred.Correo,
			"host":     cred.Host,
			"password": cred.Contrasena,
		})

	})

	r.POST("/search", func(c *gin.Context) {

		var formData models.FormData
		if err := c.ShouldBind(&formData); err != nil {
			c.HTML(http.StatusBadRequest, "form.html", gin.H{"error": "Invalid data"})
			return
		}

		token, err := controllers.GetToken(formData)
		if err != nil {
			c.HTML(http.StatusUnauthorized, "form.html", gin.H{"error": "Could not retrieve the token. Please verify your credentials."})
			return
		}

		
		err = controllers.SaveCredentials(models.Credenciales{
			Correo:     formData.Correo,
			Host:       formData.Host,
			Contrasena: formData.Contrasena,
			Token:      token, 
		})

		
		config.GlobalHost = formData.Host
		config.GlobalToken = token
		config.GlobalEmail = formData.Correo

		c.HTML(http.StatusOK, "dashboard.html", gin.H{"token": config.GlobalToken})

	})


	r.GET("/search", func(c *gin.Context) {

		cred, err := controllers.GetCredentials()
		if err == nil && cred.Token != "" {		
			c.HTML(http.StatusOK, "dashboard.html", gin.H{"token": cred.Token})
			return			
		}

	
		c.HTML(http.StatusOK, "form.html", gin.H{
			"error":    "Expired token. Please log in again.",
			"correo":   cred.Correo,
			"host":     cred.Host,
			"password": cred.Contrasena, 
		})


	})

	r.POST("/process", func(c *gin.Context) {
		
		cred, err := controllers.GetCredentials()
		if err != nil {			
			c.HTML(http.StatusInternalServerError, "dashboard.html", gin.H{"error": "Error retrieving credentials.", "token": ""})
			return
		}


		var requestData models.RequestForm
		if err := c.ShouldBind(&requestData); err != nil {
			c.HTML(http.StatusBadRequest, "dashboard.html", gin.H{"error": "Error processing the form.", "token": cred.Token})
			return
		}

	
		req := models.Request{
			Path:   requestData.Ruta,
			Status: 0, 
			Date:   time.Now(),
		}

		if err := controllers.SaveRequest(req); err != nil {
			c.HTML(http.StatusInternalServerError, "dashboard.html", gin.H{"error": "Error saving the request.", "token": cred.Token})
			return
		}
		
		c.Redirect(http.StatusSeeOther, "/panel")

	})


	r.GET("/panel", func(c *gin.Context) {

		
		go func() {
		    err := controllers.ProcessPendingRequests()
		    if err != nil {
		        fmt.Printf("Error processing pending requests : %v\n", err)
		    }
		}()

	
		requests, err := controllers.GetAllRequests()
		if err != nil {
		
			c.HTML(http.StatusInternalServerError, "panel.html", gin.H{"error": "Error retrieving requests"})
			return
		}

		
		c.HTML(http.StatusOK, "panel.html", gin.H{
			"requests": requests, "user":config.GlobalEmail,
		})

	})




	r.GET("/upload", func(c *gin.Context) {
	    
	    cred, err := controllers.GetCredentials()
	    if err != nil || cred.Token == "" {
	        c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid or missing credentials"})
	        return
	    }

	  
	    config.GlobalHost = cred.Host
	    config.GlobalToken = cred.Token
	    config.GlobalEmail = cred.Correo

	    log.Println("Credentials obtained, starting file processing...")
	   

	   
	    go func() {
	        if err := controllers.ProcessFilesController(); err != nil {
	            log.Println("Error in files processing: ", err)
	        } else {
	            log.Println("Files processing completed successfully.")
	        }
	    }()

	    c.Redirect(http.StatusSeeOther, "/panel")
	})



	r.GET("/requests/:request_id/status", controllers.GetFileProcessingStatus)

	r.GET("/logs", controllers.GetLogs)

	r.DELETE("/logs", controllers.ClearLogs)

	r.POST("/requests/:id/retry-errors", controllers.RetryFailedUploads)

	r.GET("/delete-token", func(c *gin.Context) {
		controllers.DeleteSessionToken()
		c.Redirect(http.StatusSeeOther, "/")
	})

	return r
}
