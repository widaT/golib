package server

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/pprof"
	"path"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
	"github.com/widaT/golib/logger"
	"compress/gzip"
)

const gzipMinLength = 20

// ServerConfig is configuration for server objects.
type ServerConfig struct {
	StaticDir    string
	Addr         string
	Port         int
	CookieSecret string
	RecoverPanic bool
	Profiler     bool
	GZIP         bool
}

// Server represents a web.go server.
type Server struct {
	Config *ServerConfig
	tree   *Tree
	filters []filterRoute
	Logger *logger.GxLogger
	Env    map[string]interface{}
	//save the listener so it can be closed
	l net.Listener
}

func NewServer() *Server {
	return &Server{
		Config: Config,
		Logger: defaultLogger,
		tree:	NewTree(),
		Env:    map[string]interface{}{},
	}
}

func (s *Server) initServer() {
	if s.Config == nil {
		s.Config = &ServerConfig{}
	}

	if s.Logger == nil {
		s.Logger = defaultLogger
	}
}

type route struct {
	method      string
	handler     reflect.Value
	httpHandler http.Handler
}


type FilerFun func(* Context) bool

type filterRoute struct {
	r           string
	cr          *regexp.Regexp
	handler     FilerFun
}

func (s *Server) addRoute(r string, handler interface{}) {
	switch handler.(type) {
	case http.Handler:
		s.tree.AddRouter(r,route{httpHandler: handler.(http.Handler)})
	case reflect.Value:
		fv := handler.(reflect.Value)
		s.tree.AddRouter(r,route{handler: fv})
	default:
		fv := reflect.ValueOf(handler)
		s.tree.AddRouter(r,route{handler: fv})
	}
}


func (s *Server) addFilter(r string,  fn FilerFun) {
	cr, err := regexp.Compile(r)
	if err != nil {
		s.Logger.Printf("Error in filter regex %q\n", r)
		return
	}
	s.filters = append(s.filters, filterRoute{r: r, cr: cr,handler: fn})
}


// ServeHTTP is the interface method for Go's http server package
func (s *Server) ServeHTTP(c http.ResponseWriter, req *http.Request) {
	s.Process(c, req)
}

// Process invokes the routing system for server s
func (s *Server) Process(c http.ResponseWriter, req *http.Request) {
	//println(req.PostFormValue("gid"))
	route := s.routeHandler(req, c)
	if route != nil {
		route.httpHandler.ServeHTTP(c, req)
	}
}

//Adds a custom handler. Only for webserver mode. Will have no effect when running as FCGI or SCGI.
func (s *Server) Handler(route string, httpHandler http.Handler) {
	s.addRoute(route, httpHandler)
}

// Run starts the web application and serves HTTP requests for s
func (s *Server) Run(addr string) {
	s.initServer()

	mux := http.NewServeMux()
	if s.Config.Profiler {
		mux.Handle("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
		mux.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
		mux.Handle("/debug/pprof/heap", pprof.Handler("heap"))
		mux.Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
	}
	mux.Handle("/", s)

	s.Logger.Printf("web.go serving %s", addr)

	l, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal("ListenAndServe:", err)
	}
	s.l = l
	err = http.Serve(s.l, mux)
	s.l.Close()
}

// RunTLS starts the web application and serves HTTPS requests for s.
func (s *Server) RunTLS(addr string, config *tls.Config) error {
	s.initServer()
	mux := http.NewServeMux()
	mux.Handle("/", s)
	l, err := tls.Listen("tcp", addr, config)
	if err != nil {
		log.Fatal("Listen:", err)
		return err
	}

	s.l = l
	return http.Serve(s.l, mux)
}

// Close stops server s.
func (s *Server) Close() {
	if s.l != nil {
		s.l.Close()
	}
}

// safelyCall invokes `function` in recover block
func (s *Server) safelyCall(function reflect.Value, args []reflect.Value) (resp []reflect.Value, e interface{}) {
	defer func() {
		if err := recover(); err != nil {
			if !s.Config.RecoverPanic {
				// go back to panic
				panic(err)
			} else {
				e = err
				resp = nil
				s.Logger.Error("Handler crashed with error", err)
				for i := 1; ; i += 1 {
					_, file, line, ok := runtime.Caller(i)
					if !ok {
						break
					}
					s.Logger.Error(file, line)
				}
			}
		}
	}()
	return function.Call(args), nil
}

// requiresContext determines whether 'handlerType' contains
// an argument to 'web.Ctx' as its first argument
func requiresContext(handlerType reflect.Type) bool {
	//if the method doesn't take arguments, no
	if handlerType.NumIn() == 0 {
		return false
	}

	//if the first argument is not a pointer, no
	a0 := handlerType.In(0)
	if a0.Kind() != reflect.Ptr {
		return false
	}
	//if the first argument is a context, yes
	if a0.Elem() == contextType {
		return true
	}

	return false
}

// tryServingFile attempts to serve a static file, and returns
// whether or not the operation is successful.
// It checks the following directories for the file, in order:
// 1) Config.StaticDir
// 2) The 'static' directory in the parent directory of the executable.
// 3) The 'static' directory in the current working directory
func (s *Server) tryServingFile(name string, req *http.Request, w http.ResponseWriter) bool {
	//try to serve a static file
	if s.Config.StaticDir != "" {
		staticFile := path.Join(s.Config.StaticDir, name)
		if fileExists(staticFile) {
			http.ServeFile(w, req, staticFile)
			return true
		}
	} else {
		for _, staticDir := range defaultStaticDirs {
			staticFile := path.Join(staticDir, name)
			if fileExists(staticFile) {
				http.ServeFile(w, req, staticFile)
				return true
			}
		}
	}
	return false
}

func (s *Server) logRequest(ctx Context, sTime time.Time) {
	//log the request
	var logEntry bytes.Buffer
	req := ctx.Request
	requestPath := req.URL.Path

	duration := time.Now().Sub(sTime)
	var client string

	// We suppose RemoteAddr is of the form Ip:Port as specified in the Request
	// documentation at http://golang.org/pkg/net/http/#Request
	pos := strings.LastIndex(req.RemoteAddr, ":")
	if pos > 0 {
		client = req.RemoteAddr[0:pos]
	} else {
		client = req.RemoteAddr
	}
	//考虑代理lbs代理ip
	if ctx.Request.Header.Get("X-Forwarded-For") != "" {
		client = ctx.Request.Header.Get("X-Forwarded-For")
	}
	fmt.Fprintf(&logEntry, "%s - %s %s - %v", client, req.Method, requestPath, duration)

	//处理参数超过500的情况，防止日志过大
	if len(ctx.Params) > 0 {
		paramsCopy := make(map[string]string)
		for key ,param := range ctx.Params {
			if len(param) > 1000 {
				paramsCopy[key] = "len longger than 1000"
			}else{
				paramsCopy[key] = param
			}
		}
		fmt.Fprintf(&logEntry, " - Params: %v", paramsCopy)
	}
	ctx.Server.Logger.Print(logEntry.String())
}

// the main route handler in web.go
// Tries to handle the given request.
// Finds the route matching the request, and execute the callback associated
// with it.  In case of custom http handlers, this function returns an "unused"
// route. The caller is then responsible for calling the httpHandler associated
// with the returned route.
func (s *Server) routeHandler(req *http.Request, w http.ResponseWriter) (unused *route) {
	requestPath := req.URL.Path

	ctx := Context{req, map[string]string{}, s, w}

	//set some default headers
	ctx.SetHeader("Server", "gxrsgo", true)
	tm := time.Now().UTC()

	//ignore errors from ParseForm because it's usually harmless.
	req.ParseForm()
	if len(req.Form) > 0 {
		for k, v := range req.Form {
			ctx.Params[k] = v[0]
		}
	}

	req.ParseMultipartForm(32 << 20)
	if len(req.PostForm) > 0 {
		for k, v := range req.PostForm {
			ctx.Params[k] = v[0]
		}
	}

	defer s.logRequest(ctx, tm)

	ctx.SetHeader("Date", webTime(tm), true)

	if req.Method == "GET" || req.Method == "HEAD" {
		if s.tryServingFile(requestPath, req, w) {
			return
		}
	}

	//Set the default content-type
	ctx.SetHeader("Content-Type", "text/html; charset=utf-8", true)

	//do the filters
	for i := 0; i < len(s.filters); i++ {
		filter_route := s.filters[i]
		cr := filter_route.cr
		if !cr.MatchString(requestPath) {
			continue
		}
		fn := filter_route.handler
		if !fn(&ctx) {
			return
		}
	}

	if ret := s.tree.Match(requestPath);ret != nil {
		route := ret.(route)
		if route.httpHandler != nil {
			unused = &route
			// We can not handle custom http handlers here, give back to the caller.
			return
		}
		var args []reflect.Value
		handlerType := route.handler.Type()
		if requiresContext(handlerType) {
			args = append(args, reflect.ValueOf(&ctx))
		}

		ret, err := s.safelyCall(route.handler, args)
		if err != nil {
			ctx.Abort(500, "Server Error")
		}

		if len(ret) == 0 {
			return
		}

		sval := ret[0]
		var content []byte

		if sval.Kind() == reflect.String {
			content = []byte(sval.String())
		} else if sval.Kind() == reflect.Slice && sval.Type().Elem().Kind() == reflect.Uint8 {
			content = sval.Interface().([]byte)
		}

		if s.Config.GZIP && strings.Contains(ctx.Request.Header.Get("Accept-Encoding"), "gzip") && len(content) > gzipMinLength {
			ctx.SetHeader("Content-Encoding", "gzip", true)
			ctx.SetHeader("Vary", "Accept-Encoding", true)
			ctx.Header().Del("Content-Length")
			ctx.SetHeader("Transfer-Encoding", "chunked", true)
			gz := gzip.NewWriter(ctx.ResponseWriter)
			defer gz.Close()
			_, err = gz.Write(content)
		} else {
			ctx.SetHeader("Content-Length", strconv.Itoa(len(content)), true)
			_, err = ctx.ResponseWriter.Write(content)
		}
		if err != nil {
			ctx.Server.Logger.Error("Error during write: ", err)
		}
		return
	}
	// try serving index.html or index.htm
	if req.Method == "GET" || req.Method == "HEAD" {
		if s.tryServingFile(path.Join(requestPath, "index.html"), req, w) {
			return
		} else if s.tryServingFile(path.Join(requestPath, "index.htm"), req, w) {
			return
		}
	}
	ctx.Abort(404, "Page not found")
	return
}

// SetLogger sets the logger for server s
func (s *Server) SetLogger(logger *logger.GxLogger) {
	s.Logger = logger
}
