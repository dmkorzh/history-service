package server

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

// @title    Call history REST API
// @version  1.0
// @host     localhost:8081
func (s *HttpServer) makeRouter(router *gin.Engine) {
	router.Use(logger(), gin.CustomRecovery(abortWithError))
	router.GET("/history/:department/", s.ctrl.History)
	router.POST("/call/", s.ctrl.SaveCall)
}

func logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		if len(c.Errors) > 0 {
			for i := range c.Errors {
				log.Warnf("request error: %d %s %s %s: %s",
					c.Writer.Status(), c.Request.Method, c.Request.RequestURI, time.Since(start), c.Errors[i].Err)
			}
		} else {
			log.Infof("%d %s %s %s",
				c.Writer.Status(), c.Request.Method, c.Request.RequestURI, time.Since(start))
		}
	}
}

func abortWithError(c *gin.Context, err interface{}) {
	if e, ok := err.(error); ok {
		_ = c.Error(e)
	} else {
		_ = c.Error(fmt.Errorf("panic: %T %v", err, err))
	}
	c.AbortWithStatus(http.StatusInternalServerError)
}

func parseTemplates(router *gin.Engine) (map[string]*template.Template, error) {
	res := make(map[string]*template.Template)
	for _, route := range router.Routes() {
		endpoint := strings.SplitN(route.Path, ":", 2)[0]
		file := filepath.Join("sql", endpoint, route.Method+".sql")
		data, err := content.ReadFile(file)
		if err != nil {
			return nil, err
		}
		tmpl, err := template.New("").Parse(string(data))
		if err != nil {
			return nil, err
		}
		res[endpoint] = tmpl
	}
	return res, nil
}
