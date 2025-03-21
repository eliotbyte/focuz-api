package main

import (
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	r.GET("/notes/get", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "Placeholder for notes"})
	})
	r.Run()
}
