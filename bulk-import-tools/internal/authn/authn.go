// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package authn

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
)

func Authenticate(ctx context.Context, u *url.URL) (string, error) {
	username, password, err := getUserPass()
	if err != nil {
		return "", err
	}
	u.Path = path.Join(u.Path, "/realms/master/protocol/openid-connect/token")

	// Prepare the form data
	formData := url.Values{
		"username":   {username},
		"password":   {password},
		"grant_type": {"password"},
		"client_id":  {"system-client"},
		"scope":      {"openid"},
	}

	// Create the HTTP client and make request
	client := &http.Client{}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), strings.NewReader(formData.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	var resp *http.Response
	if resp, err = client.Do(req); err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", errors.New("authentication failed with status code: " + resp.Status)
	}

	// Parse JSON response to extract the access token
	var tokenResponse struct {
		AccessToken string `json:"access_token"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResponse); err != nil {
		return "", err
	}

	return tokenResponse.AccessToken, nil
}

func getUserPass() (string, string, error) {
	username := os.Getenv("EDGEORCH_USER")
	password := os.Getenv("EDGEORCH_PASSWORD")
	var err error

	if username == "" || password == "" {
		reader := bufio.NewReader(os.Stdin)

		if username == "" {
			fmt.Print("Enter Username: ")
			username, err = reader.ReadString('\n')
			if err != nil {
				return "", "", err
			}
			username = strings.TrimSpace(username)
		}

		if password == "" {
			fmt.Print("Enter Password: ")
			password, err = reader.ReadString('\n')
			if err != nil {
				return "", "", err
			}
			password = strings.TrimSpace(password)
		}
	}

	return username, password, nil
}
