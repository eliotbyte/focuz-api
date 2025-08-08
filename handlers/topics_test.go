package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"
)

func (s *E2ETestSuite) Test19_CreateTopicAsOwner() {
	reqBody := map[string]interface{}{
		"spaceId": s.createdSpaceID,
		"name":    "Test Topic",
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
		id := int(topicData["id"].(float64))
		s.createdTopicID = id
		s.True(s.createdTopicID > 0)
	} else {
		s.Fail("Topic creation failed")
	}
}

func (s *E2ETestSuite) Test20_GuestCannotCreateTopic() {
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

func (s *E2ETestSuite) Test21_UpdateTopicTypeIdShouldFail() {
	reqBody := map[string]interface{}{
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
	s.Equal(http.StatusBadRequest, resp.StatusCode)
}

func (s *E2ETestSuite) Test22_GetTopicsBySpaceIncludesDates() {
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

	// Handle paginated response structure
	data := response["data"].(map[string]interface{})
	topics := data["data"].([]interface{})
	s.True(len(topics) >= 1)

	found := false
	for _, t := range topics {
		topic := t.(map[string]interface{})
		if int(topic["id"].(float64)) == s.createdTopicID {
			found = true
			s.Contains(topic, "name")
			s.Contains(topic, "typeId")
			s.Contains(topic, "createdAt")
			s.Contains(topic, "modifiedAt")
			break
		}
	}
	s.True(found)
}
