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

func GetRelatedIssues(fields map[string]interface{}, direction string, issueType string) []string {
	var linkedKeys []string
	if linksRaw, ok := fields["issuelinks"].([]interface{}); ok {
		for _, linkRaw := range linksRaw {
			link := linkRaw.(map[string]interface{})
			if t, ok := link["type"].(map[string]interface{}); ok {
				if inward, ok := t[direction].(string); ok && inward == issueType {
					if inwardIssue, ok := link[fmt.Sprintf("%sIssue", direction)].(map[string]interface{}); ok {
						if key, ok := inwardIssue["key"].(string); ok {
							linkedKeys = append(linkedKeys, key)
						}
					}
				}
			}
		}
	}
	return linkedKeys
}

func GetIssue(conf *config.Configuration, key string, request_fields string) (map[string]interface{}, error) {
	client := &http.Client{}
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/rest/api/3/issue/%s?fields=issuelinks,%s", conf.JiraBaseURL, key, request_fields), nil)
	req.SetBasicAuth(conf.JiraEmail, conf.JiraAPIKey)
	req.Header.Set("Accept", "application/json")

	var issueMap map[string]interface{}
	resp, err := client.Do(req)
	if err != nil {
		return issueMap, fmt.Errorf("Error getting Jira issue %s: %s", key, err.Error())
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return issueMap, fmt.Errorf("Error reading Jira response %s: %s", key, err.Error())
	}
	if resp.StatusCode != 200 {
		return issueMap, fmt.Errorf("Error getting Jira issue (unsuccessful status code) %s: %s", key, string(body))
	}

	err = json.Unmarshal(body, &issueMap)
	if err != nil {
		return issueMap, fmt.Errorf("Error parsing issue %s: %w", key, err)
	}

	return issueMap["fields"].(map[string]interface{}), nil
}

func UpdateIssues(conf *config.Configuration, fields map[string]interface{}, issues []string, customFields string) error {
	custFields := strings.Split(customFields, ",")
	fieldsToUpdate := make(map[string]interface{})
	for _, cf := range custFields {
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
		putReq, _ := http.NewRequest("PUT", fmt.Sprintf("%s/rest/api/3/issue/%s", conf.JiraBaseURL, targetKey), bytes.NewBuffer(payloadBytes))
		putReq.SetBasicAuth(conf.JiraEmail, conf.JiraAPIKey)
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

func main() {
	slog.Info("Running github.com/mexby/jira-automation-helper...")
	conf := config.Get()

	http.HandleFunc("/v1/issue/{id}/{type}/{typevalue}/{fields}", func(w http.ResponseWriter, r *http.Request) {
		fields, err := GetIssue(conf, r.PathValue("id"), r.PathValue("fields"))
		if err != nil {
			slog.Error("Error getting issue", "error", err.Error())
			w.WriteHeader(500)
			w.Write([]byte(err.Error()))
			return
		}

		switch r.PathValue("type") {
		case "inward":
		case "outward":
			linked := GetRelatedIssues(fields, r.PathValue("type"), r.PathValue("typevalue"))
			if linked == nil {
				slog.Error("No related issues found")
				w.WriteHeader(500)
				w.Write(nil)
				return
			}
			if err = UpdateIssues(conf, fields, linked, r.PathValue("fields")); err != nil {
				slog.Error("Error during issue update", "error", err.Error())
				w.WriteHeader(500)
				w.Write([]byte(err.Error()))
			}
			w.Write(nil)

		case "issue":
			w.WriteHeader(500)
			w.Write([]byte("not implemented"))
			return
		}
	})

	http.ListenAndServe(":3000", nil)
}
