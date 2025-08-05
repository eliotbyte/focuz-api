package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"
)

func (s *E2ETestSuite) Test10_CreateTopicAsOwner() {
	reqBody := map[string]interface{}{
		"spaceId": s.createdSpaceID,
		"name":    "My Notebook",
		"typeId":  1,
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
		s.createdTopicID = int(topicData["id"].(float64))
		s.True(s.createdTopicID > 0)
	} else {
		s.Fail("Topic creation failed")
	}
}

func (s *E2ETestSuite) Test11_GuestCannotCreateTopic() {
	reqBody := map[string]interface{}{
		"spaceId": s.createdSpaceID,
		"name":    "Guest Topic",
		"typeId":  1,
	}
	jsonBody, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", s.baseURL+"/topics", bytes.NewBuffer(jsonBody))
	req.Header.Set("Authorization", "Bearer "+s.guestToken)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	s.NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusForbidden, resp.StatusCode)
}

// New or modified tests start here:

func (s *E2ETestSuite) Test12_UpdateTopicTypeIdShouldFail() {
	reqBody := map[string]interface{}{
		"name":   "Renamed Topic",
		"typeId": 2,
	}
	jsonBody, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("PATCH", s.baseURL+"/topics/"+strconv.Itoa(s.createdTopicID), bytes.NewBuffer(jsonBody))
	req.Header.Set("Authorization", "Bearer "+s.ownerToken)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	s.NoError(err)
	defer resp.Body.Close()
	// Depending on your implementation, this might return 400 or 403
	// We'll assume 400 Bad Request if the code explicitly forbids changing typeId
	s.Equal(http.StatusBadRequest, resp.StatusCode)
}

func (s *E2ETestSuite) Test13_GetTopicsBySpaceIncludesDates() {
	req, _ := http.NewRequest("GET", s.baseURL+"/spaces/"+strconv.Itoa(s.createdSpaceID)+"/topics", nil)
	req.Header.Set("Authorization", "Bearer "+s.ownerToken)
	client := &http.Client{}
	resp, err := client.Do(req)
	s.NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusOK, resp.StatusCode)

	var response map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&response)
	s.True(response["success"].(bool))
	topics := response["data"].([]interface{})
	s.True(len(topics) > 0)

	for _, t := range topics {
		topic := t.(map[string]interface{})
		s.Contains(topic, "id")
		s.Contains(topic, "name")
		s.Contains(topic, "typeId")
		s.Contains(topic, "createdAt")
		s.Contains(topic, "modifiedAt")
	}
}
