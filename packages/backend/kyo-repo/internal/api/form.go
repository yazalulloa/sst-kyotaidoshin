package api

import (
	"net/http"
	"strconv"
)

type FormRequest struct {
	req *http.Request
}

func NewFormRequest(r *http.Request) (*FormRequest, error) {
	err := r.ParseForm()
	if err != nil {
		return nil, err
	}
	return &FormRequest{req: r}, nil
}

func (r *FormRequest) StringArray(paramName string) []string {
	return r.req.Form[paramName]
}

func (r *FormRequest) String(paramName string) string {
	return r.req.Form.Get(paramName)
}

func (r *FormRequest) Int(paramName string) int {
	str := r.req.Form.Get(paramName)
	if str == "" {
		return 0
	}
	value, err := strconv.Atoi(str)
	if err != nil {
		return 0
	}
	return value
}

func (r *FormRequest) Int64(paramName string) int64 {
	str := r.req.Form.Get(paramName)
	if str == "" {
		return 0
	}
	value, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return 0
	}
	return value
}

func (r *FormRequest) Float64(paramName string) float64 {
	str := r.req.Form.Get(paramName)
	if str == "" {
		return 0
	}
	value, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return 0
	}
	return value
}

func (r *FormRequest) Bool(paramName string) bool {
	str := r.req.Form.Get(paramName)
	if str == "" {
		return false
	}
	value, err := strconv.ParseBool(str)
	if err != nil {
		return false
	}
	return value
}
