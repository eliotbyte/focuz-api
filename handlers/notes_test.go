package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"time"
)

func (s *E2ETestSuite) Test23_CreateNote() {
	reqBody := map[string]interface{}{
		"text":     "My first note",
		"tags":     []string{"personal", "important"},
		"parentId": nil,
		"date":     time.Now().Format(time.RFC3339),
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
	if noteResp["success"] != nil && noteResp["success"].(bool) {
		noteData := noteResp["data"].(map[string]interface{})
		s.createdNoteID = int(noteData["id"].(float64))
		s.True(s.createdNoteID > 0)
	} else {
		s.Fail("Note creation failed")
	}
}

func (s *E2ETestSuite) Test24_CreateNoteReply() {
	reqBody := map[string]interface{}{
		"text":     "Reply to note",
		"tags":     []string{"reply"},
		"parentId": s.createdNoteID,
		"date":     time.Now().Format(time.RFC3339),
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

func (s *E2ETestSuite) Test25_CreateNoteAsGuestInNotebook() {
	// Re-invite guest to the space since they might have been removed in previous tests
	reqBody := map[string]string{"username": "guest"}
	jsonBody, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", s.baseURL+"/spaces/"+strconv.Itoa(s.createdSpaceID)+"/invite", bytes.NewBuffer(jsonBody))
	req.Header.Set("Authorization", "Bearer "+s.ownerToken)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	s.NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusOK, resp.StatusCode)

	// Now create note as guest
	reqBody2 := map[string]interface{}{
		"text":    "Guest notebook note",
		"tags":    []string{"guest", "notebook"},
		"date":    time.Now().Format(time.RFC3339),
		"topicId": s.createdTopicID,
	}
	jsonBody2, _ := json.Marshal(reqBody2)
	req2, _ := http.NewRequest("POST", s.baseURL+"/notes", bytes.NewBuffer(jsonBody2))
	req2.Header.Set("Authorization", "Bearer "+s.guestToken)
	req2.Header.Set("Content-Type", "application/json")
	client2 := &http.Client{}
	resp2, err2 := client2.Do(req2)
	s.NoError(err2)
	defer resp2.Body.Close()
	s.Equal(http.StatusCreated, resp2.StatusCode)
}

func (s *E2ETestSuite) Test26_CreateNoteAsGuestInDashboard() {
	// Re-invite guest to the space since they might have been removed in previous tests
	reqBody := map[string]string{"username": "guest"}
	jsonBody, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", s.baseURL+"/spaces/"+strconv.Itoa(s.createdSpaceID)+"/invite", bytes.NewBuffer(jsonBody))
	req.Header.Set("Authorization", "Bearer "+s.ownerToken)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	s.NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusOK, resp.StatusCode)

	// Create dashboard topic
	dashboardTopic := s.createTopic("My Dashboard", 2)
	reqBody2 := map[string]interface{}{
		"text":    "Guest dashboard note",
		"tags":    []string{"guest", "dashboard"},
		"date":    time.Now().Format(time.RFC3339),
		"topicId": dashboardTopic,
	}
	jsonBody2, _ := json.Marshal(reqBody2)
	req2, _ := http.NewRequest("POST", s.baseURL+"/notes", bytes.NewBuffer(jsonBody2))
	req2.Header.Set("Authorization", "Bearer "+s.guestToken)
	req2.Header.Set("Content-Type", "application/json")
	client2 := &http.Client{}
	resp2, err2 := client2.Do(req2)
	s.NoError(err2)
	defer resp2.Body.Close()
	s.Equal(http.StatusForbidden, resp2.StatusCode)
}

func (s *E2ETestSuite) Test27_EditNote() {
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

func (s *E2ETestSuite) Test28_GetSingleNote() {
	req, _ := http.NewRequest("GET", s.baseURL+"/notes/"+strconv.Itoa(s.createdNoteID), nil)
	req.Header.Set("Authorization", "Bearer "+s.ownerToken)
	client := &http.Client{}
	resp, err := client.Do(req)
	s.NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusOK, resp.StatusCode)
}

func (s *E2ETestSuite) Test29_GetAllNotes() {
	req, _ := http.NewRequest("GET", s.baseURL+"/notes?spaceId="+strconv.Itoa(s.createdSpaceID), nil)
	req.Header.Set("Authorization", "Bearer "+s.ownerToken)
	client := &http.Client{}
	resp, err := client.Do(req)
	s.NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusOK, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	s.Contains(string(body), "My first note")
}

func (s *E2ETestSuite) Test30_GetAllNotesWithFilters() {
	url := s.baseURL + "/notes?spaceId=" + strconv.Itoa(s.createdSpaceID) + "&tags=important&tags=!reply&notReply=true"
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+s.ownerToken)
	client := &http.Client{}
	resp, err := client.Do(req)
	s.NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusOK, resp.StatusCode)
}

func (s *E2ETestSuite) Test31_GetNotesWithDateFilters() {
	// Test filtering by date range
	url := s.baseURL + "/notes?spaceId=" + strconv.Itoa(s.createdSpaceID) + "&dateFrom=2024-01-01&dateTo=2024-12-31"
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+s.ownerToken)
	client := &http.Client{}
	resp, err := client.Do(req)
	s.NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusOK, resp.StatusCode)
}

func (s *E2ETestSuite) Test32_GetNotesWithTagExclusions() {
	url := s.baseURL + "/notes?spaceId=" + strconv.Itoa(s.createdSpaceID) + "&tags=!archived&tags=!draft"
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+s.ownerToken)
	client := &http.Client{}
	resp, err := client.Do(req)
	s.NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusOK, resp.StatusCode)
}

func (s *E2ETestSuite) Test33_GetNotesWithMixedTagFilters() {
	url := s.baseURL + "/notes?spaceId=" + strconv.Itoa(s.createdSpaceID) + "&tags=important&tags=!archived"
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+s.ownerToken)
	client := &http.Client{}
	resp, err := client.Do(req)
	s.NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusOK, resp.StatusCode)
}

func (s *E2ETestSuite) Test34_GetNotesWithSorting() {
	url := s.baseURL + "/notes?spaceId=" + strconv.Itoa(s.createdSpaceID) + "&sort=modifiedat,ASC"
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+s.ownerToken)
	client := &http.Client{}
	resp, err := client.Do(req)
	s.NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusOK, resp.StatusCode)
}

func (s *E2ETestSuite) Test35_GetNotesWithSearchAndFilters() {
	url := s.baseURL + "/notes?spaceId=" + strconv.Itoa(s.createdSpaceID) + "&search=meeting&tags=!archived&notReply=true"
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+s.ownerToken)
	client := &http.Client{}
	resp, err := client.Do(req)
	s.NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusOK, resp.StatusCode)
}

func (s *E2ETestSuite) Test36_GetNotesWithParentFilter() {
	url := s.baseURL + "/notes?spaceId=" + strconv.Itoa(s.createdSpaceID) + "&parentId=0"
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+s.ownerToken)
	client := &http.Client{}
	resp, err := client.Do(req)
	s.NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusOK, resp.StatusCode)
}

func (s *E2ETestSuite) Test37_GetNotesWithComplexFilters() {
	url := s.baseURL + "/notes?spaceId=" + strconv.Itoa(s.createdSpaceID) + "&tags=important&tags=!archived&dateFrom=2024-01-01&dateTo=2024-12-31&search=note&sort=modifiedat,ASC&notReply=true&page=1&pageSize=20"
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+s.ownerToken)
	client := &http.Client{}
	resp, err := client.Do(req)
	s.NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusOK, resp.StatusCode)
}
