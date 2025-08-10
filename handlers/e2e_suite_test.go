package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/suite"
)

type E2ETestSuite struct {
	suite.Suite
	baseURL        string
	ownerToken     string
	guestToken     string
	createdSpaceID int
	createdTopicID int
	createdNoteID  int
}

func (s *E2ETestSuite) SetupSuite() {
	// Use test API container name when running in Docker, localhost otherwise
	if os.Getenv("CI") != "" || os.Getenv("DOCKER") != "" {
		s.baseURL = "http://test-api:8080"
	} else {
		s.baseURL = "http://localhost:8080"
	}
}

func (s *E2ETestSuite) createTopic(name string, typeID int) int {
	reqBody := map[string]interface{}{
		"spaceId": s.createdSpaceID,
		"name":    name,
		"typeId":  typeID,
	}
	jsonBody, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", s.baseURL+"/topics", bytes.NewBuffer(jsonBody))
	req.Header.Set("Authorization", "Bearer "+s.ownerToken)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	s.NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusCreated, resp.StatusCode)
	var topicResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&topicResp)
	if topicResp["success"] != nil && topicResp["success"].(bool) {
		topicData := topicResp["data"].(map[string]interface{})
		return int(topicData["id"].(float64))
	}
	return 0
}

func TestE2ETestSuite(t *testing.T) {
	suite.Run(t, new(E2ETestSuite))
}
