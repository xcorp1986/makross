// Package makross is a high productive and modular web framework in Golang.

package makross

import (
	"bytes"
	ktx "context"
	"net"
	"net/http"
)

// Context represents the contextual data and environment while processing an incoming HTTP request.
type Context struct {
	Request  *http.Request       // the current request
	Response http.ResponseWriter // the response writer
	ktx      ktx.Context         // standard context
	makross  *Makross
	pnames   []string               // list of route parameter names
	pvalues  []string               // list of parameter values corresponding to pnames
	data     map[string]interface{} // data items managed by Get and Set
	index    int                    // the index of the currently executing handler in handlers
	handlers []Handler              // the handlers associated with the current route
	writer   DataWriter
}

// NewContext creates a new Context object with the given response, request, and the handlers.
// This method is primarily provided for writing unit tests for handlers.
func NewContext(res http.ResponseWriter, req *http.Request, handlers ...Handler) *Context {
	c := &Context{handlers: handlers}
	c.Reset(res, req)
	return c
}

// Makross returns the Makross that is handling the incoming HTTP request.
func (c *Context) Makross() *Makross {
	return c.makross
}

func (c *Context) Kontext() ktx.Context {
	return c.ktx
}

func (c *Context) SetKontext(ktx ktx.Context) {
	c.ktx = ktx
}

func (c *Context) Handler() Handler {
	return c.handlers[c.index]
}

func (c *Context) SetHandler(h Handler) {
	c.handlers[c.index] = h
}

func (c *Context) Error(status int, message ...interface{}) {
	herr := NewHTTPError(status, message...)
	c.HandleError(herr)
}

func (c *Context) HandleError(err error) {
	c.makross.HandleError(c, err)
}

// RealIP implements `Context#RealIP` function.
func (c *Context) RealIP() string {
	ra := c.Request.RemoteAddr
	if ip := c.Request.Header.Get(HeaderXForwardedFor); len(ip) > 0 {
		ra = ip
	} else if ip := c.Request.Header.Get(HeaderXRealIP); len(ip) > 0 {
		ra = ip
	} else {
		ra, _, _ = net.SplitHostPort(ra)
	}
	return ra
}

// Param returns the named parameter value that is found in the URL path matching the current route.
// If the named parameter cannot be found, an empty string will be returned.
func (c *Context) Param(name string) string {
	for i, n := range c.pnames {
		if n == name {
			return c.pvalues[i]
		}
	}
	return ""
}

// Get returns the named data item previously registered with the context by calling Set.
// If the named data item cannot be found, nil will be returned.
func (c *Context) Get(name string) interface{} {
	return c.data[name]
}

// Set stores the named data item in the context so that it can be retrieved later.
func (c *Context) Set(name string, value interface{}) {
	if c.data == nil {
		c.data = make(map[string]interface{})
	}
	c.data[name] = value
}

func (c *Context) SetStore(data map[string]interface{}) {
	if c.data == nil {
		c.data = make(map[string]interface{})
	}
	for k, v := range data {
		c.data[k] = v
	}
}

func (c *Context) GetStore() map[string]interface{} {
	return c.data
}

// Query returns the first value for the named component of the URL query parameters.
// If key is not present, it returns the specified default value or an empty string.
func (c *Context) Query(name string, defaultValue ...string) string {
	if vs, _ := c.Request.URL.Query()[name]; len(vs) > 0 {
		return vs[0]
	}
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return ""
}

// Form returns the first value for the named component of the query.
// Form reads the value from POST and PUT body parameters as well as URL query parameters.
// The form takes precedence over the latter.
// If key is not present, it returns the specified default value or an empty string.
func (c *Context) Form(key string, defaultValue ...string) string {
	r := c.Request
	r.ParseMultipartForm(32 << 20)
	if vs := r.Form[key]; len(vs) > 0 {
		return vs[0]
	}

	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return ""
}

// PostForm returns the first value for the named component from POST and PUT body parameters.
// If key is not present, it returns the specified default value or an empty string.
func (c *Context) PostForm(key string, defaultValue ...string) string {
	r := c.Request
	r.ParseMultipartForm(32 << 20)
	if vs := r.PostForm[key]; len(vs) > 0 {
		return vs[0]
	}

	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return ""
}

// Next calls the rest of the handlers associated with the current route.
// If any of these handlers returns an error, Next will return the error and skip the following handlers.
// Next is normally used when a handler needs to do some postprocessing after the rest of the handlers
// are executed.
func (c *Context) Next() error {
	c.index++
	for n := len(c.handlers); c.index < n; c.index++ {
		if err := c.handlers[c.index](c); err != nil {
			return err
		}
	}
	return nil
}

// Abort skips the rest of the handlers associated with the current route.
// Abort is normally used when a handler handles the request normally and wants to skip the rest of the handlers.
// If a handler wants to indicate an error condition, it should simply return the error without calling Abort.
func (c *Context) Abort() {
	c.index = len(c.handlers)
}

// URL creates a URL using the named route and the parameter values.
// The parameters should be given in the sequence of name1, value1, name2, value2, and so on.
// If a parameter in the route is not provided a value, the parameter token will remain in the resulting URL.
// Parameter values will be properly URL encoded.
// The method returns an empty string if the URL creation fails.
func (c *Context) URL(route string, pairs ...interface{}) string {
	if r := c.makross.namedRoutes[route]; r != nil {
		return r.URL(pairs...)
	}
	return ""
}

// Read populates the given struct variable with the data from the current request.
// If the request is NOT a GET request, it will check the "Content-Type" header
// and find a matching reader from DataReaders to read the request data.
// If there is no match or if the request is a GET request, it will use DefaultFormDataReader
// to read the request data.
func (c *Context) Read(data interface{}) error {
	if c.Request.Method != "GET" {
		t := getContentType(c.Request)
		if reader, ok := DataReaders[t]; ok {
			return reader.Read(c.Request, data)
		}
	}

	return DefaultFormDataReader.Read(c.Request, data)
}

// Write writes the given data of arbitrary type to the response.
// The method calls the data writer set via SetDataWriter() to do the actual writing.
// By default, the DefaultDataWriter will be used.
func (c *Context) Write(data interface{}) error {
	return c.writer.Write(c.Response, data)
}

func (c *Context) Render(name string, status ...int) (err error) {
	var code int
	if len(status) > 0 {
		code = status[0]
	} else {
		code = StatusOK
	}
	if c.makross.renderer == nil {
		return ErrRendererNotRegistered
	}
	buf := new(bytes.Buffer)
	if err = c.makross.renderer.Render(buf, name, c); err != nil {
		return
	}
	c.Response.Header().Set(HeaderContentType, MIMETextHTMLCharsetUTF8)
	c.Response.WriteHeader(code)
	err = c.Write(buf.Bytes())
	c.Abort()
	return
}

func (c *Context) String(s string, status ...int) (err error) {
	var code int
	if len(status) > 0 {
		code = status[0]
	} else {
		code = StatusOK
	}
	c.Response.Header().Set(HeaderContentType, MIMETextPlainCharsetUTF8)
	c.Response.WriteHeader(code)
	err = c.Write([]byte(s))
	c.Abort()
	return
}

func (c *Context) NoContent(status ...int) error {
	var code int
	if len(status) > 0 {
		code = status[0]
	} else {
		code = StatusOK
	}
	c.Response.WriteHeader(code)
	return nil
}

// SetDataWriter sets the data writer that will be used by Write().
func (c *Context) SetDataWriter(writer DataWriter) {
	c.writer = writer
}

// Reset sets the request and response of the context and resets all other properties.
func (c *Context) Reset(response http.ResponseWriter, request *http.Request) {
	c.Response = response
	c.Request = request
	c.ktx = ktx.Background()
	c.data = nil
	c.index = -1
	c.writer = DefaultDataWriter
}

func getContentType(req *http.Request) string {
	t := req.Header.Get("Content-Type")
	for i, c := range t {
		if c == ' ' || c == ';' {
			return t[:i]
		}
	}
	return t
}