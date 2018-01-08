package httpjsonrpc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/rpc"
	"sync"
)

type Codec struct {
	URL string
	http.Client

	once       sync.Once
	abort      chan struct{}
	respBuffer chan *receivedResponse
	resp       *receivedResponse
}

func (c *Codec) init() {
	c.once.Do(func() {
		c.respBuffer = make(chan *receivedResponse)
		c.abort = make(chan struct{})
	})
}

type clientRequest struct {
	Method string         `json:"method"`
	Params [1]interface{} `json:"params"`
	Id     uint64         `json:"id"`
}

func (c *Codec) WriteRequest(r *rpc.Request, param interface{}) error {
	req := clientRequest{
		Method: r.ServiceMethod,
		Id:     r.Seq,
	}
	body, err := json.Marshal(req)
	if err != nil {
		return err
	}
	resp, err := c.Client.Post(c.URL, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || 300 <= resp.StatusCode {
		return fmt.Errorf("request failed: %v", resp.Status)
	}
	var cr clientResponse
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(&cr); err != nil {
		return err
	}

	c.init()
	go func() {
		resp := &receivedResponse{ServiceMethod: r.ServiceMethod, clientResponse: cr}
		select {
		case c.respBuffer <- resp:
			//noop
		case <-c.abort:
			//noop
		}
	}()
	return nil
}

type receivedResponse struct {
	ServiceMethod string
	clientResponse
}

type clientResponse struct {
	Id     uint64           `json:"id"`
	Result *json.RawMessage `json:"result"`
	Error  interface{}      `json:"error"`
}

func (c *Codec) ReadResponseHeader(r *rpc.Response) error {
	c.init()
	var ok bool
	c.resp, ok = <-c.respBuffer
	if !ok {
		return fmt.Errorf("codec is closed")
	}

	r.ServiceMethod = c.resp.ServiceMethod
	r.Error = ""
	r.Seq = c.resp.Id
	if c.resp.Error != nil || c.resp.Result == nil {
		x, ok := c.resp.Error.(string)
		if !ok {
			return fmt.Errorf("invalid error %v", c.resp.Error)
		}
		if x == "" {
			x = "unspecified error"
		}
		r.Error = x
	}
	return nil
}

func (c *Codec) ReadResponseBody(x interface{}) error {
	if x == nil {
		return nil
	}
	return json.Unmarshal(*c.resp.Result, x)
}

func (c *Codec) Close() error {
	c.init()
	select {
	case <-c.abort:
		// noop, already closed
	default:
		close(c.abort)
	}
	return nil
}
