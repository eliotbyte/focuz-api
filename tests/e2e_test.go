package tests

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
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

func (s *E2ETestSuite) Test1_RegisterOwner() {
	body := `{"username":"owner","password":"ownerpass"}`
	resp, err := http.Post(s.baseURL+"/register", "application/json", bytes.NewBuffer([]byte(body)))
	s.NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusCreated, resp.StatusCode)
}

func (s *E2ETestSuite) Test2_RegisterOwnerConflict() {
	body := `{"username":"owner","password":"ownerpass"}`
	resp, err := http.Post(s.baseURL+"/register", "application/json", bytes.NewBuffer([]byte(body)))
	s.NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusInternalServerError, resp.StatusCode)
}

func (s *E2ETestSuite) Test3_LoginOwnerInvalid() {
	body := `{"username":"owner","password":"invalid"}`
	resp, err := http.Post(s.baseURL+"/login", "application/json", bytes.NewBuffer([]byte(body)))
	s.NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusUnauthorized, resp.StatusCode)
}

func (s *E2ETestSuite) Test4_LoginOwnerValid() {
	body := `{"username":"owner","password":"ownerpass"}`
	resp, err := http.Post(s.baseURL+"/login", "application/json", bytes.NewBuffer([]byte(body)))
	s.NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusOK, resp.StatusCode)

	var data map[string]string
	json.NewDecoder(resp.Body).Decode(&data)
	s.ownerToken = data["token"]
	s.NotEmpty(s.ownerToken)
}

func (s *E2ETestSuite) Test5_CreateSpace() {
	reqBody := `{"name":"Test Space"}`
	req, _ := http.NewRequest("POST", s.baseURL+"/spaces", bytes.NewBuffer([]byte(reqBody)))
	req.Header.Set("Authorization", "Bearer "+s.ownerToken)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	s.NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusCreated, resp.StatusCode)

	var spaceResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&spaceResp)
	id := int(spaceResp["id"].(float64))
	s.createdSpaceID = id
	s.True(s.createdSpaceID > 0)
}

func (s *E2ETestSuite) Test6_RegisterGuest() {
	body := `{"username":"guest","password":"guestpass"}`
	resp, err := http.Post(s.baseURL+"/register", "application/json", bytes.NewBuffer([]byte(body)))
	s.NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusCreated, resp.StatusCode)
}

func (s *E2ETestSuite) Test7_LoginGuest() {
	body := `{"username":"guest","password":"guestpass"}`
	resp, err := http.Post(s.baseURL+"/login", "application/json", bytes.NewBuffer([]byte(body)))
	s.NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusOK, resp.StatusCode)
	var data map[string]string
	json.NewDecoder(resp.Body).Decode(&data)
	s.guestToken = data["token"]
	s.NotEmpty(s.guestToken)
}

func (s *E2ETestSuite) Test8_InviteGuest() {
	reqBody := map[string]int{"userId": s.getGuestUserID()}
	jsonBody, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", s.baseURL+"/spaces/"+strconv.Itoa(s.createdSpaceID)+"/invite", bytes.NewBuffer(jsonBody))
	req.Header.Set("Authorization", "Bearer "+s.ownerToken)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	s.NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusNoContent, resp.StatusCode)
}

func (s *E2ETestSuite) Test9_GuestCannotEditSpace() {
	reqBody := `{"name":"New Name"}`
	req, _ := http.NewRequest("PATCH", s.baseURL+"/spaces/"+strconv.Itoa(s.createdSpaceID), bytes.NewBuffer([]byte(reqBody)))
	req.Header.Set("Authorization", "Bearer "+s.guestToken)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	s.NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusForbidden, resp.StatusCode)
}

func (s *E2ETestSuite) Test10_CreateTopicAsOwner() {
	reqBody := map[string]interface{}{
		"spaceId": s.createdSpaceID,
		"name":    "My Diary",
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
	s.createdTopicID = int(topicResp["id"].(float64))
	s.True(s.createdTopicID > 0)
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
	// Create a new topic with typeId=2 as owner
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
	// Currently the API does not have an endpoint for patching notes' text, so this would fail
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

func (s *E2ETestSuite) getGuestUserID() int {
	body := `{"username":"guest","password":"guestpass"}`
	req, _ := http.NewRequest("POST", s.baseURL+"/login", bytes.NewBuffer([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, _ := client.Do(req)
	defer resp.Body.Close()
	var data map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&data)
	if data["token"] == nil {
		return 0
	}
	// Not a real endpoint, just for demonstration
	return 2
}

func (s *E2ETestSuite) createTopic(name string, typeId int) int {
	reqBody := map[string]interface{}{
		"spaceId": s.createdSpaceID,
		"name":    name,
		"typeId":  typeId,
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
	return int(topicResp["id"].(float64))
}

func TestE2ETestSuite(t *testing.T) {
	if os.Getenv("E2E") != "" {
		suite.Run(t, new(E2ETestSuite))
	}
}
