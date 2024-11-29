package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/didip/tollbooth"
	"github.com/didip/tollbooth/limiter"
	"github.com/didip/tollbooth_gin"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"os"
	"path/filepath"
	"strings"
	"time"
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
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	r.Use(gin.Recovery())

	lmt := tollbooth.NewLimiter(2, nil)
	lmt = tollbooth.NewLimiter(2, &limiter.ExpirableOptions{DefaultExpirationTTL: time.Hour})

	r.Use(tollbooth_gin.LimitHandler(lmt))

	r.LoadHTMLGlob("HTMLtemplates/*.html")
	r.Static("/static", "static")
	r.StaticFile("/favicon.ico", "static/favicon.ico")

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
			ShaUSR := sha256.New()
			ShaUSR.Write([]byte(usr))
			var files []files
			db.Find(&files, "user = ?", hex.EncodeToString(ShaUSR.Sum(nil)))
			c.HTML(200, "history.html", gin.H{
				"files": files,
				"user":  strings.Split(usr, "-")[1] + strings.Split(usr, "-")[0],
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
			ShaUSR := sha256.New()
			ShaUSR.Write([]byte(usr))
			c.SaveUploadedFile(file, filePath)
			dbEntry := files{Fname: fileUUID, Frename: fileName, User: hex.EncodeToString(ShaUSR.Sum(nil))}
			db.Create(&dbEntry)
			c.String(200, hex.EncodeToString(ShaUSR.Sum(nil))+"/"+file.Filename)
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
			ShaUSR := sha256.New()
			ShaUSR.Write([]byte(usr))
			c.SaveUploadedFile(file, filePath)
			dbEntry := files{Fname: fileUUID, Frename: fileName, User: hex.EncodeToString(ShaUSR.Sum(nil))}
			db.Create(&dbEntry)
			c.String(200, hex.EncodeToString(ShaUSR.Sum(nil))+"/"+file.Filename)
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

	r.GET("/api/v1/delete/:user/:file", func(c *gin.Context) {
		user, err := c.Cookie("user")
		if err != nil || user == "" {
			return
		}
		var uriDownload uriDownload
		if err := c.ShouldBindUri(&uriDownload); err != nil {
			fmt.Println(err)
			return
		}
		usr := uriDownload.User
		file := uriDownload.File
		ShaUSR := sha256.New()
		ShaUSR.Write([]byte(user))
		if usr != hex.EncodeToString(ShaUSR.Sum(nil)) {
			c.String(200, "error: unauthorized")
			return
		}
		filez := files{}
		db.First(&filez, "Frename = ? AND user = ?", file, usr)
		db.Delete(&filez)
		os.Remove("uploadedfiles/" + sanitizeFileName(filez.Fname))
		c.String(200, "deleted")
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
