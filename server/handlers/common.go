package handlers

import (
	"strings"
	"text/template"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/gin-gonic/gin"
)

type Controller struct {
	Database  *Database
	Templates map[string]*template.Template
}

type Database struct {
	Driver clickhouse.Conn
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
