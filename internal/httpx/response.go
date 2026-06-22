package httpx

import (
	"unicode"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
)

type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func capitalize(message string) string {
	if message == "" {
		return message
	}
	r, size := utf8.DecodeRuneInString(message)
	if unicode.IsUpper(r) {
		return message
	}
	return string(unicode.ToUpper(r)) + message[size:]
}

func OK(c *gin.Context, data any) {
	c.JSON(200, data)
}

func Created(c *gin.Context, data any) {
	c.JSON(201, data)
}

func NoContent(c *gin.Context) {
	c.Status(204)
}

func Error(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, gin.H{"error": ErrorBody{Code: code, Message: capitalize(message)}})
}

func NotImplemented(c *gin.Context) {
	c.JSON(501, gin.H{"error": ErrorBody{Code: "not_implemented", Message: "Endpoint not implemented yet"}})
}
