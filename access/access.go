// Copyright 2016 Qiang Xue. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Package access provides an access logging handler for the ozzo makross package.
package access

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	makross "github.com/insionng/makross"
)

// LogFunc logs a message using the given format and optional arguments.
// The usage of format and arguments is similar to that for fmt.Printf().
// LogFunc should be thread safe.
type LogFunc func(format string, a ...interface{})

// LogWriterFunc takes in the request and responseWriter objects as well
// as a float64 containing the elapsed time since the request first passed
// through this middleware and does whatever log writing it wants with that
// information.
// LogWriterFunc should be thread safe.
type LogWriterFunc func(req *http.Request, res *LogResponseWriter, elapsed float64)

// CustomLogger returns a handler that calls the LogWriterFunc passed to it for every request.
// The LogWriterFunc is provided with the http.Request and LogResponseWriter objects for the
// request, as well as the elapsed time since the request first came through the middleware.
// LogWriterFunc can then do whatever logging it needs to do.
//
//     import (
//         "log"
//         "github.com/insionng/makross"
//         "github.com/insionng/makross/access"
//         "net/http"
//     )
//
//     func myCustomLogger(req http.Context, res access.LogResponseWriter, elapsed int64) {
//         // Do something with the request, response, and elapsed time data here
//     }
//     r := makross.New()
//     r.Use(access.CustomLogger(myCustomLogger))
func CustomLogger(loggerFunc LogWriterFunc) makross.Handler {
	return func(c *makross.Context) error {
		startTime := time.Now()

		req := c.Request

		err := c.Next()
		rw := &LogResponseWriter{c.Response, c.Response.Status, 0}

		elapsed := float64(time.Now().Sub(startTime).Nanoseconds()) / 1e6
		loggerFunc(req, rw, elapsed)

		return err
	}

}

// Logger returns a handler that logs a message for every request.
// The access log messages contain information including client IPs, time used to serve each request, request line,
// response status and size.
//
//     import (
//         "log"
//         "github.com/insionng/makross"
//         "github.com/insionng/makross/access"
//     )
//
//     r := makross.New()
//     r.Use(access.Logger(log.Printf))
func Logger(log LogFunc) makross.Handler {
	var logger = func(req *http.Request, rw *LogResponseWriter, elapsed float64) {
		clientIP := GetClientIP(req)
		requestLine := fmt.Sprintf("%s %s %s", req.Method, req.URL.String(), req.Proto)
		log(`[%s] [%.3fms] %s %d %d`, clientIP, elapsed, requestLine, rw.Status, rw.BytesWritten)

	}
	return CustomLogger(logger)
}

// LogResponseWriter wraps http.ResponseWriter in order to capture HTTP status and response length information.
type LogResponseWriter struct {
	*makross.Response
	Status       int
	BytesWritten int64
}

func GetClientIP(req *http.Request) string {
	ip := req.Header.Get("X-Real-IP")
	if ip == "" {
		ip = req.Header.Get("X-Forwarded-For")
		if ip == "" {
			ip = req.RemoteAddr
		}
	}
	if colon := strings.LastIndex(ip, ":"); colon != -1 {
		ip = ip[:colon]
	}
	return ip
}
