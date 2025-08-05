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
	s.baseURL = "http://localhost:8080"
}

func (s *E2ETestSuite) getGuestUserID() int {
	body := `{"username":"guest","password":"guestpass"}`
	req, _ := http.NewRequest("POST", s.baseURL+"/login", bytes.NewBuffer([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()

	var data map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&data)
	if data["success"] != nil && data["success"].(bool) {
		tokenData := data["data"].(map[string]interface{})
		if tokenData["token"] != nil {
			return 2
		}
	}
	return 0
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
	if os.Getenv("E2E") != "" {
		suite.Run(t, new(E2ETestSuite))
	}
}
