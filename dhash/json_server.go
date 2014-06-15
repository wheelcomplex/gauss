package dhash

// TODO: This code is a piece of shit. Remove gorilla/mux dependency.

import (
	"code.google.com/p/go.net/websocket"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/purak/gauss/common"
	"io"
	"net"
	"net/http"
	"net/rpc"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	updateInterval = time.Second
)

type socketMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

type baseData struct {
	Timestamp int64
}

var (
	prefPattern = regexp.MustCompile("^([^\\s;]+)(;q=([\\d.]+))?$")
	apiMethods  string
)

func getFormat(t reflect.Type) interface{} {
	if t.Kind() == reflect.Struct {
		result := make(map[string]interface{})
		var field reflect.StructField
		for i := 0; i < t.NumField(); i++ {
			field = t.Field(i)
			if field.Type.Kind() == reflect.Slice && field.Type.Elem().Kind() == reflect.Uint8 {
				result[field.Name] = "[]byte"
			} else if field.Type.Kind() == reflect.Ptr && field.Type.Elem().Kind() == reflect.Int {
				result[field.Name] = "int"
			} else {
				result[field.Name] = field.Type.Name()
			}
		}
		return result
	}
	return t.Name()
}

func SetApi(t reflect.Type) {
	m := make(map[string]map[string]interface{})
	var meth reflect.Method
	var in reflect.Type
	for i := 0; i < t.NumMethod(); i++ {
		meth = t.Method(i)
		if strings.ToUpper(string(meth.Name[0])) == string(meth.Name[0]) && meth.Type.NumIn() == 3 {
			in = meth.Type.In(1)
			m[meth.Name] = map[string]interface{}{
				"name":      meth.Name,
				"parameter": getFormat(in),
			}
		}
	}
	var bts []byte
	var err error
	if bts, err = json.Marshal(m); err != nil {
		panic(err)
	}
	apiMethods = string(bts)
}

func (self baseData) ApiMethods() string {
	return apiMethods
}

func mostAccepted(r *http.Request, def, name string) string {
	bestValue := def
	var bestScore float64 = -1
	var score float64
	for _, pref := range strings.Split(r.Header.Get(name), ",") {
		if match := prefPattern.FindStringSubmatch(pref); match != nil {
			score = 1
			if match[3] != "" {
				score = common.MustParseFloat64(match[3])
			}
			if score > bestScore {
				bestScore = score
				bestValue = match[1]
			}
		}
	}
	return bestValue
}

func wantsJSON(r *http.Request, m *mux.RouteMatch) bool {
	return mostAccepted(r, "text/html", "Accept") == "application/json"
}

func wantsHTML(r *http.Request, m *mux.RouteMatch) bool {
	return mostAccepted(r, "text/html", "Accept") == "text/html"
}

type requestContext struct {
	method   string
	request  *http.Request
	response http.ResponseWriter
}

func (self *requestContext) ReadRequestHeader(r *rpc.Request) error {
	*r = rpc.Request{
		ServiceMethod: self.method,
	}
	return nil
}

func (self *requestContext) getBodyString() string {
	b := make([]byte, self.request.ContentLength)
	if _, err := io.ReadFull(self.request.Body, b); err != nil {
		panic(err)
	}
	return string(b)
}

func (self *requestContext) ReadRequestBody(b interface{}) (err error) {
	if b != nil {
		if self.request.ContentLength > 0 {
			if _, ok := b.(*int); ok {
				var i int64
				if i, err = strconv.ParseInt(self.getBodyString(), 10, 64); err != nil {
					return
				}
				reflect.ValueOf(b).Elem().SetInt(i)
			} else {
				err = json.NewDecoder(self.request.Body).Decode(b)
			}
		}
	}
	return
}

func (self *requestContext) WriteResponse(resp *rpc.Response, b interface{}) (err error) {
	self.response.Header().Set("Content-Type", "application/json; charset=UTF-8")
	var bts []byte
	if resp.Error != "" {
		self.response.WriteHeader(500)
		if bts, err = json.Marshal(resp.Error); err != nil {
			return
		}
	} else {
		if bts, err = json.Marshal(b); err != nil {
			return
		}
	}
	self.response.Header().Set("Content-Length", fmt.Sprint(len(bts)))
	_, err = self.response.Write(bts)
	return
}

func (self *requestContext) Close() error {
	return self.request.Body.Close()
}

type jsonRpcServer struct {
	server *rpc.Server
}

func (self jsonRpcServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	context := &requestContext{
		method:   mux.Vars(r)["method"],
		request:  r,
		response: w,
	}
	self.server.ServeRequest(context)
}

func (self *Node) jsonDescription() string {
	b, err := json.Marshal(socketMessage{
		Type: "RingChange",
		Data: map[string]interface{}{
			"description": self.Description(),
			"routes":      self.node.Nodes(),
		},
	})
	if err != nil {
		panic(err)
	}
	return string(b)
}

func Route(handler websocket.Handler, router *mux.Router) {
	router.Path("/ws").Handler(handler)
}

func (self *Node) startJson() {
	var nodeAddr *net.TCPAddr
	var err error

	if nodeAddr, err = net.ResolveTCPAddr("tcp", self.node.GetListenAddr()); err != nil {
		return
	}

	rpcServer := rpc.NewServer()
	jsonApi := (*JSONApi)(self)
	SetApi(reflect.TypeOf(jsonApi))
	rpcServer.RegisterName("DHash", jsonApi)
	jsonServer := jsonRpcServer{server: rpcServer}

	router := mux.NewRouter()
	router.Methods("POST").Path("/rpc/{method}").MatcherFunc(wantsJSON).Handler(jsonServer)

	Route(func(ws *websocket.Conn) {
		if websocket.Message.Send(ws, self.jsonDescription()) == nil {
			go func() {
				for {
					time.Sleep(updateInterval)
					if websocket.Message.Send(ws, self.jsonDescription()) != nil {
						break
					}
				}
			}()
			self.AddCommListener(func(comm Comm) bool {
				b, err := json.Marshal(socketMessage{
					Type: "Comm",
					Data: map[string]interface{}{
						"source":      comm.Source,
						"destination": comm.Destination,
						"key":         comm.Key,
						"sub_key":     comm.SubKey,
						"type":        comm.Type,
					},
				})
				if err != nil {
					panic(err)
				}
				return websocket.Message.Send(ws, string(b)) == nil
			})
			self.AddChangeListener(func(ring *common.Ring) bool {
				b, err := json.Marshal(socketMessage{
					Type: "RingChange",
					Data: map[string]interface{}{
						"description": self.Description(),
						"routes":      self.node.Nodes(),
					},
				})
				if err != nil {
					panic(err)
				}
				return websocket.Message.Send(ws, string(b)) == nil
			})
			self.AddSyncListener(func(source, dest common.Remote, pulled, pushed int) bool {
				b, err := json.Marshal(socketMessage{
					Type: "Sync",
					Data: map[string]interface{}{
						"source":      source,
						"destination": dest,
						"pulled":      pulled,
						"pushed":      pushed,
					},
				})
				if err != nil {
					panic(err)
				}
				return websocket.Message.Send(ws, string(b)) == nil
			})
			self.AddCleanListener(func(source, dest common.Remote, cleaned, pushed int) bool {
				b, err := json.Marshal(socketMessage{
					Type: "Clean",
					Data: map[string]interface{}{
						"source":      source,
						"destination": dest,
						"cleaned":     cleaned,
						"pushed":      pushed,
					},
				})
				if err != nil {
					panic(err)
				}
				return websocket.Message.Send(ws, string(b)) == nil
			})
			var mess string
			for {
				if err = websocket.Message.Receive(ws, &mess); err != nil {
					break
				}
			}
		}
	}, router)

	mux := http.NewServeMux()
	mux.Handle("/", router)

	listener, err := net.Listen("tcp", fmt.Sprintf("%v:%v", nodeAddr.IP, nodeAddr.Port+1))
	if err != nil {
		panic(err)
	}

	go (&http.Server{
		Handler: mux,
	}).Serve(listener)
}
