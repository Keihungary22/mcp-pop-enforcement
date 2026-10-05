package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var ErrInactiveToken = errors.New("access token is inactive")

// IntrospectionValidator augments local token validation with
// authorization-server token activity checking.
//
// IntrospectionValidator はローカルのトークン検証に加えて、
// Authorization Server によるトークン有効状態を確認する。
type IntrospectionValidator struct {
	base         TokenValidator
	endpoint     string
	clientID     string
	clientSecret string
	httpClient   *http.Client
}

func NewIntrospectionValidator(
	base TokenValidator,
	endpoint string,
	clientID string,
	clientSecret string,
) *IntrospectionValidator {
	return &IntrospectionValidator{
		base:         base,
		endpoint:     endpoint,
		clientID:     clientID,
		clientSecret: clientSecret,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (v *IntrospectionValidator) Validate(
	tokenString string,
) (*Claims, error) {
	claims, err := v.base.Validate(tokenString)
	if err != nil {
		return nil, err
	}

	form := url.Values{}
	form.Set("token", tokenString)

	request, err := http.NewRequest(
		http.MethodPost,
		v.endpoint,
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"create token introspection request: %w",
			err,
		)
	}

	request.Header.Set(
		"Content-Type",
		"application/x-www-form-urlencoded",
	)

	request.SetBasicAuth(
		v.clientID,
		v.clientSecret,
	)

	response, err := v.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf(
			"token introspection request failed: %w",
			err,
		)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"token introspection returned HTTP %d",
			response.StatusCode,
		)
	}

	var result struct {
		Active bool `json:"active"`
	}

	if err := json.NewDecoder(response.Body).Decode(
		&result,
	); err != nil {
		return nil, fmt.Errorf(
			"decode token introspection response: %w",
			err,
		)
	}

	if !result.Active {
		return nil, ErrInactiveToken
	}

	return claims, nil
}
