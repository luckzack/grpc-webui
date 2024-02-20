package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gogoods/grpc-webui/loadtest"

	core "github.com/gogoods/grpcli"
	"github.com/gorilla/mux"
)

// Handler hold all handler methods
type Handler struct {
	g *core.Client
}

// InitHandler Constructor
func InitHandler() *Handler {
	return &Handler{
		g: core.NewClient(),
	}
}

func (h *Handler) index(w http.ResponseWriter, r *http.Request) {
	body := new(bytes.Buffer)
	err := indexHTML.Execute(body, make(map[string]string))
	if err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(body.Bytes())
}

func (h *Handler) getActiveConns(w http.ResponseWriter, r *http.Request) {
	response(w, h.g.GetActiveConns(context.TODO()))
}

func (h *Handler) closeActiveConns(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	host := vars["host"]
	if host == "" {
		writeError(w, fmt.Errorf("Invalid Host"))
		return
	}

	err := h.g.CloseActiveConns(strings.Trim(host, " "))
	if err != nil {
		writeError(w, err)
		return
	}
	response(w, map[string]bool{"success": true})
}

func (h *Handler) getLists(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	host := vars["host"]
	if host == "" {
		writeError(w, fmt.Errorf("Invalid Host"))
		return
	}

	md, err := parseMetadataFromQuery(r.URL.Query())
	if err != nil {
		writeError(w, fmt.Errorf("parseMetadataFromQuery fail: %s", err.Error()))
		return
	}
	h.g.SetHeaders(md.Slice)

	service := vars["serv_name"]

	useTLS, _ := strconv.ParseBool(r.Header.Get("use_tls"))
	restart, _ := strconv.ParseBool(r.FormValue("restart"))

	res, err := h.g.GetResource(context.Background(), host, !useTLS, restart)
	if err != nil {
		writeError(w, err)
		return
	}

	result, err := res.List(service)
	if err != nil {
		writeError(w, err)
		return
	}

	h.g.Extend(host)
	response(w, result)
}

// getListsWithProto handling client request for service list with proto
func (h *Handler) getListsWithProto(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	host := vars["host"]
	if host == "" {
		writeError(w, fmt.Errorf("Invalid Host"))
		return
	}

	md, err := parseMetadataFromQuery(r.URL.Query())
	if err != nil {
		writeError(w, fmt.Errorf("parseMetadataFromQuery fail: %s", err.Error()))
		return
	}
	h.g.SetHeaders(md.Slice)

	service := vars["serv_name"]

	useTLS, _ := strconv.ParseBool(r.Header.Get("use_tls"))
	restart, _ := strconv.ParseBool(r.FormValue("restart"))

	// limit upload file to 5mb
	err = r.ParseMultipartForm(5 << 20)
	if err != nil {
		writeError(w, err)
		return
	}

	// convert uploaded files to list of Proto struct
	files := r.MultipartForm.File["protos"]
	protos := make([]core.Proto, 0, len(files))
	for _, file := range files {
		fileData, err := file.Open()
		if err != nil {
			writeError(w, err)
			return
		}
		defer fileData.Close()

		content, err := ioutil.ReadAll(fileData)
		if err != nil {
			writeError(w, err)
		}

		protos = append(protos, core.Proto{
			Name:    file.Filename,
			Content: content,
		})
	}

	res, err := h.g.GetResourceWithProto(context.Background(), host, !useTLS, restart, protos)
	if err != nil {
		writeError(w, err)
		return
	}

	result, err := res.List(service)
	if err != nil {
		writeError(w, err)
		return
	}

	h.g.Extend(host)
	response(w, result)
}

func (h *Handler) describeFunction(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	host := vars["host"]
	if host == "" {
		writeError(w, fmt.Errorf("Invalid Host"))
		return
	}

	md, err := parseMetadataFromQuery(r.URL.Query())
	if err != nil {
		writeError(w, fmt.Errorf("parseMetadataFromQuery fail: %s", err.Error()))
		return
	}
	h.g.SetHeaders(md.Slice)

	funcName := vars["func_name"]
	if funcName == "" {
		writeError(w, fmt.Errorf("Invalid Func Name"))
		return
	}

	useTLS, _ := strconv.ParseBool(r.Header.Get("use_tls"))

	res, err := h.g.GetResource(context.Background(), host, !useTLS, false)
	if err != nil {
		writeError(w, err)
		return
	}

	// get param
	result, _, err := res.Describe(funcName)
	if err != nil {
		writeError(w, err)
		return
	}
	match := reGetFuncArg.FindStringSubmatch(result)
	if len(match) < 2 {
		writeError(w, fmt.Errorf("Invalid Func Type"))
		return
	}

	// describe func
	result, template, err := res.Describe(match[1])
	if err != nil {
		writeError(w, err)
		return
	}

	type desc struct {
		Schema   string `json:"schema"`
		Template string `json:"template"`
	}

	h.g.Extend(host)
	response(w, desc{
		Schema:   result,
		Template: template,
	})

}

func (h *Handler) invokeFunction(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	host := vars["host"]
	if host == "" {
		writeError(w, fmt.Errorf("Invalid Host"))
		return
	}

	funcName := vars["func_name"]
	if funcName == "" {
		writeError(w, fmt.Errorf("Invalid Func Name"))
		return
	}

	md, err := parseMetadataFromQuery(r.URL.Query())
	if err != nil {
		writeError(w, fmt.Errorf("parseMetadataFromQuery fail: %s", err.Error()))
		return
	}
	h.g.SetHeaders(md.Slice)

	useTLS, _ := strconv.ParseBool(r.Header.Get("use_tls"))

	res, err := h.g.GetResource(context.Background(), host, !useTLS, false)
	if err != nil {
		writeError(w, err)
		return
	}

	result, timer, err := res.Invoke(context.Background(), funcName, r.Body)
	if err != nil {
		writeError(w, err)
		return
	}

	type invRes struct {
		Time   string `json:"timer"`
		Result string `json:"result"`
	}

	h.g.Extend(host)
	response(w, invRes{
		Time:   timer.String(),
		Result: result,
	})
}

func (h *Handler) loadTestFunction(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	host := vars["host"]
	if host == "" {
		writeError(w, fmt.Errorf("Invalid Host"))
		return
	}

	funcName := vars["func_name"]
	if funcName == "" {
		writeError(w, fmt.Errorf("Invalid Func Name"))
		return
	}

	md, err := parseMetadataFromQuery(r.URL.Query())
	if err != nil {
		writeError(w, fmt.Errorf("parseMetadataFromQuery fail: %s", err.Error()))
		return
	}
	//h.g.SetHeaders(md.Slice)

	useTLS, _ := strconv.ParseBool(r.Header.Get("use_tls"))

	result, cost, err := loadtest.Run(funcName, host, !useTLS, md.Map, r.Body)

	if err != nil {
		writeError(w, err)
		return
	}
	//
	type invRes struct {
		Time   string `json:"timer"`
		Result string `json:"result"`
	}
	//
	//h.g.Extend(host)
	response(w, invRes{
		Time:   cost.String(),
		Result: result,
	})
}

type Metadata struct {
	Map   map[string]string //{"k1":"v1", "k2":"v2"}
	Slice []string          //["k1: v1", "k2: v2"]
}

func parseMetadataFromQuery(query url.Values) (md Metadata, err error) {

	mdValue := ""
	for key, vals := range query {
		if key == "md" {
			mdValue = vals[0]
		}
	}

	if mdValue == "" {
		return
	}

	md = Metadata{map[string]string{}, []string{}}
	err = jsonDecode([]byte(mdValue), &md.Map)
	if err != nil {
		return
	}

	for k, v := range md.Map {
		md.Slice = append(md.Slice, fmt.Sprintf("%s: %s", k, v))
	}

	return
}

func jsonDecode(bs []byte, target interface{}) error {
	dec := json.NewDecoder(bytes.NewBuffer(bs))
	dec.UseNumber()
	return dec.Decode(target)
}
