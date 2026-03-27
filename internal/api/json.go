package api

import (
	"encoding/json"
	"net/http"
	"reflect"
)

const apiCodeOK = 0

func writeObjOK(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(struct {
		Success bool   `json:"success"`
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    any    `json:"data"`
	}{
		Success: true,
		Code:    apiCodeOK,
		Message: "",
		Data:    data,
	})
}

func writeObjErr(w http.ResponseWriter, status int, code string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(struct {
		Success bool   `json:"success"`
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    any    `json:"data"`
	}{
		Success: false,
		Code:    APICodeNum(code),
		Message: msgForCode(code),
		Data:    nil,
	})
}

// listDataForJSON 保证列表 `data` 序列化为 JSON 数组：nil 切片在 Go 里赋给 any 后 `== nil` 为 false，
// 直接 Encode 会得到 "data":null，须换成空切片才得到 []。
func listDataForJSON(data any) any {
	if data == nil {
		return []any{}
	}
	rv := reflect.ValueOf(data)
	if rv.Kind() == reflect.Slice && rv.IsNil() {
		return reflect.MakeSlice(rv.Type(), 0, 0).Interface()
	}
	return data
}

func writeListOK(w http.ResponseWriter, status int, data any, page int, total int64) {
	data = listDataForJSON(data)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(struct {
		Success bool   `json:"success"`
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    any    `json:"data"`
		Page    int    `json:"page"`
		Total   int64  `json:"total"`
	}{
		Success: true,
		Code:    apiCodeOK,
		Message: "",
		Data:    data,
		Page:    page,
		Total:   total,
	})
}

func writeListErr(w http.ResponseWriter, status int, code string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(struct {
		Success bool   `json:"success"`
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    any    `json:"data"`
		Page    int    `json:"page"`
		Total   int64  `json:"total"`
	}{
		Success: false,
		Code:    APICodeNum(code),
		Message: msgForCode(code),
		Data:    []any{},
		Page:    0,
		Total:   0,
	})
}

func readJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}
