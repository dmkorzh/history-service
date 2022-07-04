package handlers

import (
	"database/sql"
	"net"
	"strings"
	"text/template"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/gin-gonic/gin"
)

type Controller struct {
	AccessList []net.IP
	MsgTopic   string
	FailTopic  string
	MsgC       chan<- *kafka.Message
	Database   *Database
	Templates  map[string]*template.Template
}

type Database struct {
	Driver *sql.DB
	Table  string
}

type HTTPError struct {
	Error string `json:"error" example:"error text"`
}

func endpoint(c *gin.Context) string {
	domain := c.Param("department")
	return strings.SplitN(c.Request.URL.Path, domain, 2)[0]
}

func httpError(c *gin.Context, code int, err error) {
	response := &HTTPError{Error: err.Error()}
	c.JSON(code, response)
	_ = c.Error(err)
}
