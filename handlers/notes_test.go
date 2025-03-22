package handlers

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
)

func (s *E2ETestSuite) Test12_CreateNote() {
	reqBody := map[string]interface{}{
		"text":     "My first note",
		"tags":     []string{"personal", "important"},
		"parentId": nil,
		"date":     nil,
		"topicId":  s.createdTopicID,
	}
	jsonBody, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", s.baseURL+"/notes", bytes.NewBuffer(jsonBody))
	req.Header.Set("Authorization", "Bearer "+s.ownerToken)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	s.NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusCreated, resp.StatusCode)

	var noteResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&noteResp)
	s.createdNoteID = int(noteResp["id"].(float64))
	s.True(s.createdNoteID > 0)
}

func (s *E2ETestSuite) Test13_CreateNoteReply() {
	reqBody := map[string]interface{}{
		"text":     "Reply to note",
		"tags":     []string{"reply"},
		"parentId": s.createdNoteID,
		"topicId":  s.createdTopicID,
	}
	jsonBody, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", s.baseURL+"/notes", bytes.NewBuffer(jsonBody))
	req.Header.Set("Authorization", "Bearer "+s.ownerToken)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	s.NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusCreated, resp.StatusCode)
}

func (s *E2ETestSuite) Test14_CreateNoteAsGuestInDiary() {
	reqBody := map[string]interface{}{
		"text":    "Guest diary note",
		"tags":    []string{"guest", "diary"},
		"topicId": s.createdTopicID,
	}
	jsonBody, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", s.baseURL+"/notes", bytes.NewBuffer(jsonBody))
	req.Header.Set("Authorization", "Bearer "+s.guestToken)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	s.NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusCreated, resp.StatusCode)
}

func (s *E2ETestSuite) Test15_CreateNoteAsGuestInDashboard() {
	dashboardTopic := s.createTopic("My Dashboard", 2)
	reqBody := map[string]interface{}{
		"text":    "Guest dashboard note",
		"tags":    []string{"guest", "dashboard"},
		"topicId": dashboardTopic,
	}
	jsonBody, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", s.baseURL+"/notes", bytes.NewBuffer(jsonBody))
	req.Header.Set("Authorization", "Bearer "+s.guestToken)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	s.NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusForbidden, resp.StatusCode)
}

func (s *E2ETestSuite) Test16_EditNote() {
	reqBody := map[string]interface{}{
		"text":    "Edited note text",
		"tags":    []string{"important"},
		"topicId": s.createdTopicID,
	}
	jsonBody, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("PATCH", s.baseURL+"/notes/"+strconv.Itoa(s.createdNoteID), bytes.NewBuffer(jsonBody))
	req.Header.Set("Authorization", "Bearer "+s.ownerToken)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	s.NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusNotFound, resp.StatusCode)
}

func (s *E2ETestSuite) Test17_GetSingleNote() {
	req, _ := http.NewRequest("GET", s.baseURL+"/notes/"+strconv.Itoa(s.createdNoteID), nil)
	req.Header.Set("Authorization", "Bearer "+s.ownerToken)
	client := &http.Client{}
	resp, err := client.Do(req)
	s.NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusOK, resp.StatusCode)
}

func (s *E2ETestSuite) Test18_GetAllNotes() {
	url := s.baseURL + "/notes?spaceId=" + strconv.Itoa(s.createdSpaceID) + "&page=1&pageSize=10"
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+s.ownerToken)
	client := &http.Client{}
	resp, err := client.Do(req)
	s.NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusOK, resp.StatusCode)
	bytesBody, _ := ioutil.ReadAll(resp.Body)
	s.Contains(string(bytesBody), "My first note")
}

func (s *E2ETestSuite) Test19_GetAllNotesWithFilters() {
	url := s.baseURL + "/notes?spaceId=" + strconv.Itoa(s.createdSpaceID) + "&includeTags=important&excludeTags=reply&notReply=true"
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+s.ownerToken)
	client := &http.Client{}
	resp, err := client.Do(req)
	s.NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusOK, resp.StatusCode)
}
