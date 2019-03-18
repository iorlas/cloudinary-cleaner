package main

import (
	"fmt"
	"github.com/dghubble/sling"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"io/ioutil"
	"net/http"
	"time"
)

const (
	DELETED   = "deleted"
	NOT_FOUND = "not_found"
)

type Resource struct {
	PublicID     string `json:"public_id"`
	Format       string
	Version      int64
	ResourceType string `json:"resource_type"`
	Type         string
}

type UploadResponse struct {
	Bytes        float64
	Version      float64
	Format       string
	ResourceType string `json:"resource_type"`
	Url          string
	SecureUrl    string `json:"secure_url"`
	PublicID     string `json:"public_id"`
	Signature    string
}

type ErrResponse struct {
	Error struct {
		Message string
	}
}

var ErrNotFound = errors.New("file not found")
var ErrRateLimit = errors.New("rate limit reached")

type Cloudinary struct {
	Debug bool

	ApiKey    string
	ApiSecret string
	CloudName string

	log *zap.Logger
}

func NewCloudinary(viper *viper.Viper, log *zap.Logger) (*Cloudinary, error) {
	o := &Cloudinary{
		Debug:     viper.GetBool("debug"),
		ApiKey:    viper.GetString("cloudinary_api_key"),
		ApiSecret: viper.GetString("cloudinary_api_secret"),
		CloudName: viper.GetString("cloudinary_cloud_name"),

		log: log.WithOptions(zap.AddCallerSkip(2)).Named("cloudinary"),
	}
	if o.ApiKey == "" {
		panic("cloudinary_api_key")
	}

	if o.ApiSecret == "" {
		panic("cloudinary_api_secret")
	}

	if o.CloudName == "" {
		panic("cloudinary_cloud_name")
	}
	return o, nil
}

func (o *Cloudinary) delete(action string, data map[string]interface{}, out interface{}) error {
	var URL = fmt.Sprintf("https://api.cloudinary.com/v1_1/%s/%s", o.CloudName, action)
	return o.do(sling.New().Delete(URL).BodyJSON(data), out)
	return nil
}

func (o *Cloudinary) post(action string, data map[string]interface{}, out interface{}) error {
	var URL = fmt.Sprintf("https://api.cloudinary.com/v1_1/%s/%s", o.CloudName, action)
	return o.do(sling.New().Post(URL).BodyJSON(data), out)
	return nil
}

func (o *Cloudinary) do(s *sling.Sling, out interface{}) error {
	var errResponse ErrResponse

	req, err := s.Request()
	if err != nil {
		return errors.WithStack(err)
	}

	var body string

	if req.Body != nil {
		b, err := ioutil.ReadAll(req.Body)
		if err != nil {
			return errors.WithStack(err)
		}
		body = string(b)
	}

	var start = time.Now()
	resp, err := s.SetBasicAuth(o.ApiKey, o.ApiSecret).Receive(out, &errResponse)

	o.log.WithOptions(zap.AddCallerSkip(1)).Info(fmt.Sprintf("%s %s", req.Method, req.URL.Path),
		zap.String("url", req.URL.String()),
		zap.String("method", req.Method),
		zap.String("body", body),
		zap.Duration("duration", time.Now().Sub(start)),
		zap.String("response", resp.Status),
	)

	if err != nil {
		return errors.WithStack(err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return ErrNotFound
	}
	if errResponse.Error.Message != "" {
		return errors.WithStack(errors.New(fmt.Sprintf("Error fetching cloudinady data: %s", errResponse.Error.Message)))
	}

	if resp.StatusCode != http.StatusOK {
		return errors.New(resp.Status)
	}
	return nil
}

type Resources struct {
	TotalCount int         `json:"total_count"`
	NextCursor string      `json:"next_cursor"`
	Resources  []*Resource `json:"resources"`
}

func (o *Cloudinary) Search(expression string, max int) (*Resources, error) {
	var result Resources
	if err := o.post("resources/search", map[string]interface{}{
		"max_results": max,
		"expression":  expression,
	}, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (o *Cloudinary) BatchDelete(ids []string) (*Resources, error) {
	if len(ids) > 100 {
		return nil, errors.New("too much")
	}
	var result Resources
	if err := o.delete("resources/image/upload", map[string]interface{}{
		"public_ids": ids,
	}, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
