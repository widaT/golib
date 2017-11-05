// Package web is a lightweight web framework for Go. It's ideal for
// writing simple, performant backend web services.
package server

import (
    "bytes"
    "crypto/hmac"
    "crypto/sha1"
    "crypto/tls"
    "encoding/base64"
    "fmt"
    "io/ioutil"
    "mime"
    "net/http"
    "os"
    "path"
    "reflect"
    "strconv"
    "strings"
    "time"
    "github.com/widaT/golib/logger"
)

// A Context object is created for every incoming HTTP request, and is
// passed to handlers as an optional first argument. It provides information
// about the request, including the http.Request object, the GET and POST params,
// and acts as a Writer for the response.
type Context struct {
    Request *http.Request
    Params  map[string]string
    Server  *Server
    http.ResponseWriter
}

// WriteString writes string data into the response object.
func (ctx *Context) WriteString(content string) {
    ctx.ResponseWriter.Write([]byte(content))
}

// Abort is a helper method that sends an HTTP header and an optional
// body. It is useful for returning 4xx or 5xx errors.
// Once it has been called, any return value from the handler will
// not be written to the response.
func (ctx *Context) Abort(status int, body string) {
    ctx.ResponseWriter.WriteHeader(status)
    ctx.ResponseWriter.Write([]byte(body))
}

// Redirect is a helper method for 3xx redirects.
func (ctx *Context) Redirect(status int, url_ string) {
    ctx.ResponseWriter.Header().Set("Location", url_)
    ctx.ResponseWriter.WriteHeader(status)
    ctx.ResponseWriter.Write([]byte("Redirecting to: " + url_))
}

// Notmodified writes a 304 HTTP response
func (ctx *Context) NotModified() {
    ctx.ResponseWriter.WriteHeader(304)
}

// NotFound writes a 404 HTTP response
func (ctx *Context) NotFound(message string) {
    ctx.ResponseWriter.WriteHeader(404)
    ctx.ResponseWriter.Write([]byte(message))
}

//Unauthorized writes a 401 HTTP response
func (ctx *Context) Unauthorized() {
    ctx.ResponseWriter.WriteHeader(401)
}

//Forbidden writes a 403 HTTP response
func (ctx *Context) Forbidden() {
    ctx.ResponseWriter.WriteHeader(403)
}

// ContentType sets the Content-Type header for an HTTP response.
// For example, ctx.ContentType("json") sets the content-type to "application/json"
// If the supplied value contains a slash (/) it is set as the Content-Type
// verbatim. The return value is the content type as it was
// set, or an empty string if none was found.
func (ctx *Context) ContentType(val string) string {
    var ctype string
    if strings.ContainsRune(val, '/') {
        ctype = val
    } else {
        if !strings.HasPrefix(val, ".") {
            val = "." + val
        }
        ctype = mime.TypeByExtension(val)
    }
    if ctype != "" {
        ctx.Header().Set("Content-Type", ctype)
    }
    return ctype
}

// SetHeader sets a response header. If `unique` is true, the current value
// of that header will be overwritten . If false, it will be appended.
func (ctx *Context) SetHeader(hdr string, val string, unique bool) {
    if unique {
        ctx.Header().Set(hdr, val)
    } else {
        ctx.Header().Add(hdr, val)
    }
}

// SetCookie adds a cookie header to the response.
func (ctx *Context) SetCookie(cookie *http.Cookie) {
    ctx.SetHeader("Set-Cookie", cookie.String(), false)
}

func getCookieSig(key string, val []byte, timestamp string) string {
    hm := hmac.New(sha1.New, []byte(key))

    hm.Write(val)
    hm.Write([]byte(timestamp))

    hex := fmt.Sprintf("%02x", hm.Sum(nil))
    return hex
}

func (ctx *Context) SetSecureCookie(name string, val string, age int64) {
    //base64 encode the val
    if len(ctx.Server.Config.CookieSecret) == 0 {
        ctx.Server.Logger.Print("Secret Key for secure cookies has not been set. Please assign a cookie secret to web.Config.CookieSecret.")
        return
    }
    var buf bytes.Buffer
    encoder := base64.NewEncoder(base64.StdEncoding, &buf)
    encoder.Write([]byte(val))
    encoder.Close()
    vs := buf.String()
    vb := buf.Bytes()
    timestamp := strconv.FormatInt(time.Now().Unix(), 10)
    sig := getCookieSig(ctx.Server.Config.CookieSecret, vb, timestamp)
    cookie := strings.Join([]string{vs, timestamp, sig}, "|")
    ctx.SetCookie(NewCookie(name, cookie, age))
}

func (ctx *Context) GetSecureCookie(name string) (string, bool) {
    for _, cookie := range ctx.Request.Cookies() {
        if cookie.Name != name {
            continue
        }

        parts := strings.SplitN(cookie.Value, "|", 3)

        val := parts[0]
        timestamp := parts[1]
        sig := parts[2]

        if getCookieSig(ctx.Server.Config.CookieSecret, []byte(val), timestamp) != sig {
            return "", false
        }

        ts, _ := strconv.ParseInt(timestamp, 0, 64)

        if time.Now().Unix()-31*86400 > ts {
            return "", false
        }

        buf := bytes.NewBufferString(val)
        encoder := base64.NewDecoder(base64.StdEncoding, buf)

        res, _ := ioutil.ReadAll(encoder)
        return string(res), true
    }
    return "", false
}

// small optimization: cache the context type instead of repeteadly calling reflect.Typeof
var contextType reflect.Type

var defaultStaticDirs []string

var defaultLogger *logger.GxLogger

func init() {
    contextType = reflect.TypeOf(Context{})
    //find the location of the exe file
    wd, _ := os.Getwd()
    wd = strings.Replace(wd,"\\","/",-1);
    arg0 := path.Clean(os.Args[0])
    var exeFile string
    if strings.HasPrefix(arg0, "/") {
        exeFile = arg0
    } else {
        //TODO for robustness, search each directory in $PATH
        exeFile = path.Join(wd, arg0)
    }
    parent, _ := path.Split(exeFile)
    defaultStaticDirs = append(defaultStaticDirs, path.Join(parent, "static"))
    defaultStaticDirs = append(defaultStaticDirs, path.Join(wd, "static"))
    defaultLogger = logger.NewLogger(`{"filename":"`+path.Join(wd, "log")+`/access.log"}`)
    return
}

// Process invokes the main server's routing system.
func Process(c http.ResponseWriter, req *http.Request) {
    mainServer.Process(c, req)
}

// Run starts the web application and serves HTTP requests for the main server.
func Run(addr string) {
    mainServer.Run(addr)
}

// RunTLS starts the web application and serves HTTPS requests for the main server.
func RunTLS(addr string, config *tls.Config) {
    mainServer.RunTLS(addr, config)
}

// Close stops the main server.
func Close() {
    mainServer.Close()
}


func AddFilter(route string, handler FilerFun) {
    mainServer.addFilter(route, handler)
}

// Get adds a handler for the 'GET' http method in the main server.
func Get(route string, handler interface{}) {
    mainServer.Get(route, handler)
}

// Post adds a handler for the 'POST' http method in the main server.
func Post(route string, handler interface{}) {
    mainServer.addRoute(route, "POST", handler)
}

// Put adds a handler for the 'PUT' http method in the main server.
func Put(route string, handler interface{}) {
    mainServer.addRoute(route, "PUT", handler)
}

// Delete adds a handler for the 'DELETE' http method in the main server.
func Delete(route string, handler interface{}) {
    mainServer.addRoute(route, "DELETE", handler)
}

// Match adds a handler for an arbitrary http method in the main server.
func Match(method string, route string, handler interface{}) {
    mainServer.addRoute(route, method, handler)
}

//Adds a custom handler. Only for webserver mode. Will have no effect when running as FCGI or SCGI.
func Handler(route string, method string, httpHandler http.Handler) {
    mainServer.Handler(route, method, httpHandler)
}

// SetLogger sets the logger for the main server.
func SetLogger(logger *logger.GxLogger) {
    mainServer.Logger = logger
}

// Config is the configuration of the main server.
var Config = &ServerConfig{
    RecoverPanic: true,
    Port:808,
}

var mainServer = NewServer()
