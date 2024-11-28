package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type files struct {
	Id      int    `gorm:"primaryKey" json:"id"`
	Fname   string `json:"fname"`
	Frename string `json:"frename"`
	User    string `json:"user"`
}

type preminum struct {
	Id   int    `gorm:"primaryKey" json:"id"`
	User string `json:"user"`
}

type uriDownload struct {
	User string `uri:"user" binding:"required"`
	File string `uri:"file" binding:"required"`
}

func main() {
	r := gin.Default()

	r.Use(gin.Recovery())

	r.LoadHTMLGlob("HTMLtemplates/*.html")
	r.Static("/static", "static")
	err := godotenv.Load()
	if err != nil {
		fmt.Println(err)
	}
	db, _ := gorm.Open(mysql.Open(os.Getenv("DB")), &gorm.Config{})
	r.GET("/", func(c *gin.Context) {
		usr, err := c.Cookie("user")
		if err != nil || usr == "" {
			c.SetCookie("user", uuid.New().String(), 3600*24*365, "/", "localhost", true, true)
			c.HTML(200, "indexmain.html", gin.H{})
		} else {

			db.Find(&files{}, "user = ?", c.Query(usr))
			c.HTML(200, "indexmain.html", gin.H{})
		}

	})

	r.GET("/history", func(c *gin.Context) {
		usr, err := c.Cookie("user")
		if err != nil || usr == "" {
			c.SetCookie("user", uuid.New().String(), 3600*24*365, "/", "localhost", true, true)
			c.HTML(200, "history.html", gin.H{})
		} else {
			var files []files
			db.Find(&files, "user = ?", usr)
			c.HTML(200, "history.html", gin.H{
				"files": files,
				"user":  usr,
			})
		}
	})

	r.POST("/api/v1/upload", func(c *gin.Context) {
		usr, err := c.Cookie("user")
		if err == nil && usr != "" {
			fileUUID := uuid.New().String()
			file, err := c.FormFile("file")
			if err != nil {
				fmt.Println(err)
				c.String(200, "error")
				return
			}
			var hasPreminum preminum
			dbQP := db.First(&hasPreminum, "user = ?", usr)
			if dbQP.Error != nil {
			}
			if file.Size >= 104857600 {
				if hasPreminum.User != "" {
					if hasPreminum.User == usr {
						if file.Size > 262144000 {
							c.String(200, "error: file too large (≤250 MB, preminum)")
							return
						}
					}
				} else {
					c.String(200, "error: file too large (≤100 MB)")
					return
				}

			}
			fileName := sanitizeFileName(file.Filename)
			filePath := fmt.Sprintf("uploadedfiles/%s", fileUUID)

			c.SaveUploadedFile(file, filePath)
			dbEntry := files{Fname: fileUUID, Frename: fileName, User: usr}
			db.Create(&dbEntry)
			c.String(200, usr+"/"+file.Filename)
		} else {
			c.SetCookie("user", uuid.New().String(), 3600*24*365, "/", "localhost", true, true)
			fileUUID := uuid.New().String()
			file, err := c.FormFile("file")
			if err != nil {
				fmt.Println(err)
				c.String(200, "error")
				return
			}
			if file.Size > 104857600 {
				c.String(200, "error: file too large (≤100 MB)")
				return
			}
			fileName := sanitizeFileName(file.Filename)
			filePath := fmt.Sprintf("uploadedfiles/%s", fileUUID)

			c.SaveUploadedFile(file, filePath)
			dbEntry := files{Fname: fileUUID, Frename: fileName, User: usr}
			db.Create(&dbEntry)
			c.String(200, usr+"/"+file.Filename)
		}
	})

	r.GET("/api/v1/download/:user/:file", func(c *gin.Context) {
		var uriDownload uriDownload
		if err := c.ShouldBindUri(&uriDownload); err != nil {
			fmt.Println(err)
			return
		}
		usr := uriDownload.User
		file := uriDownload.File
		filez := files{}
		db.First(&filez, "Frename = ? AND User = ?", file, usr)
		c.File("uploadedfiles/" + sanitizeFileName(filez.Fname))
	})

	r.Run() // listen and serve on 0.0.0.0:8080
}

func sanitizeFileName(fileName string) string {
	// Get only the base name (removes directory paths)
	cleanName := filepath.Base(fileName)
	// Remove any potentially harmful characters
	cleanName = strings.ReplaceAll(cleanName, "..", "")
	cleanName = strings.ReplaceAll(cleanName, "/", "")
	cleanName = strings.ReplaceAll(cleanName, "\\", "")
	return cleanName
}
