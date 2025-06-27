package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/mexby/jira-automation-helper/config"
)

type RequestPayload struct {
	ID        string   `json:"id"`
	Type      string   `json:"type"`
	TypeValue string   `json:"typevalue"`
	Fields    []string `json:"fields"`
	APIKey    string   `json:"api_key"`
	Email     string   `json:"email"`
	BaseURL   string   `json:"base_url"`
}

func getRelatedIssues(fields map[string]interface{}, direction string, issueType string) []string {
	var linkedKeys []string

	linksRaw, ok := fields["issuelinks"].([]interface{})
	if !ok {
		return linkedKeys
	}

	for _, linkRaw := range linksRaw {
		link, ok := linkRaw.(map[string]interface{})
		if !ok {
			continue
		}

		t, ok := link["type"].(map[string]interface{})
		if !ok {
			continue
		}

		inward, ok := t[direction].(string)
		if !ok || inward != issueType {
			continue
		}

		issueField, ok := link[fmt.Sprintf("%sIssue", direction)].(map[string]interface{})
		if !ok {
			continue
		}

		key, ok := issueField["key"].(string)
		if !ok {
			continue
		}

		linkedKeys = append(linkedKeys, key)
	}

	return linkedKeys
}

func getIssue(payload *RequestPayload) (map[string]interface{}, error) {
	request_fields := strings.Join(payload.Fields, ",")
	client := &http.Client{}
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/rest/api/3/issue/%s?fields=issuelinks,%s", payload.BaseURL, payload.ID, request_fields), nil)
	req.SetBasicAuth(payload.Email, payload.APIKey)
	req.Header.Set("Accept", "application/json")

	var issueMap map[string]interface{}
	resp, err := client.Do(req)
	if err != nil {
		return issueMap, fmt.Errorf("Error getting Jira issue %s: %s", payload.ID, err.Error())
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return issueMap, fmt.Errorf("Error reading Jira response %s: %s", payload.ID, err.Error())
	}
	if resp.StatusCode != 200 {
		return issueMap, fmt.Errorf("Error getting Jira issue (unsuccessful status code) %s: %s", payload.ID, string(body))
	}

	err = json.Unmarshal(body, &issueMap)
	if err != nil {
		return issueMap, fmt.Errorf("Error parsing issue %s: %w", payload.ID, err)
	}

	return issueMap["fields"].(map[string]interface{}), nil
}

func updateIssues(fields map[string]interface{}, issues []string, payload *RequestPayload) error {
	fieldsToUpdate := make(map[string]interface{})
	for _, cf := range payload.Fields {
		if val, ok := fields[cf]; ok {
			fieldsToUpdate[cf] = val
		}
	}

	for _, targetKey := range issues {
		updatePayload := map[string]interface{}{
			"fields": fieldsToUpdate,
		}
		payloadBytes, _ := json.Marshal(updatePayload)

		client := &http.Client{}
		putReq, _ := http.NewRequest("PUT", fmt.Sprintf("%s/rest/api/3/issue/%s", payload.BaseURL, targetKey), bytes.NewBuffer(payloadBytes))
		putReq.SetBasicAuth(payload.Email, payload.APIKey)
		putReq.Header.Set("Content-Type", "application/json")

		putResp, err := client.Do(putReq)
		if err != nil {
			return fmt.Errorf("Error during PUT for %s: %v\n", targetKey, err)
		}
		defer putResp.Body.Close()

		respBody, _ := io.ReadAll(putResp.Body)
		if putResp.StatusCode != 204 {
			return fmt.Errorf("Error updating %s (%d): %s\n", targetKey, putResp.StatusCode, string(respBody))
		}
	}
	return nil
}

func grant(fn func(http.ResponseWriter, *http.Request), config *config.Configuration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token != config.APIKey {
			http.Error(w, "Authentication is Invalid", http.StatusInternalServerError)
			return
		}
		fn(w, r)
	}
}

func main() {
	slog.Info("Running github.com/mexby/jira-automation-helper...")
	conf := config.Get()

	http.HandleFunc("/v1/issue/", grant(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var payload RequestPayload
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&payload); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			slog.Error("Invalid request payload", "error", err)
			return
		}

		slog.Info("Received request",
			"id", payload.ID,
			"type", payload.Type,
			"typevalue", payload.TypeValue,
			"fields", payload.Fields,
			"api_key", payload.APIKey,
			"email", payload.Email,
			"base_url", payload.BaseURL,
		)

		fields, err := getIssue(&payload)
		if err != nil {
			slog.Error("Error getting issue", "error", err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		switch payload.Type {
		case "inward", "outward":
			linked := getRelatedIssues(fields, payload.Type, payload.TypeValue)
			if linked == nil {
				slog.Error("No related issues found")
				http.Error(w, "No related issues found", http.StatusBadRequest)
				return
			}
			if err = updateIssues(fields, linked, &payload); err != nil {
				slog.Error("Error during issue update", "error", err.Error())
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			w.Write(nil)

		case "issue":
			http.Error(w, "not impelmented yet", http.StatusInternalServerError)
			return
		}
	}, conf))

	http.ListenAndServe(":3000", nil)
}
