package handlers

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"strconv"
	"time"
)

func (s *E2ETestSuite) Test38_UploadFileInvalidNote() {
	// Test uploading file with invalid note ID
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "test.txt")
	part.Write([]byte("test content"))
	writer.WriteField("note_id", "99999")
	writer.Close()

	req, _ := http.NewRequest("POST", s.baseURL+"/upload", body)
	req.Header.Set("Authorization", "Bearer "+s.ownerToken)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	client := &http.Client{}
	resp, err := client.Do(req)
	s.NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusBadRequest, resp.StatusCode)
}

func (s *E2ETestSuite) Test39_UploadFile() {
	// Create a new note for this test
	reqBody := map[string]interface{}{
		"text":     "Note for file upload test",
		"tags":     []string{"upload", "test"},
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
	var noteID int
	if noteResp["success"] != nil && noteResp["success"].(bool) {
		noteData := noteResp["data"].(map[string]interface{})
		noteID = int(noteData["id"].(float64))
		s.True(noteID > 0)
	} else {
		s.Fail("Note creation failed")
		return
	}

	// Test uploading file with valid note ID
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Create the file part with explicit Content-Type
	part, _ := writer.CreatePart(map[string][]string{
		"Content-Type":        {"text/plain"},
		"Content-Disposition": {`form-data; name="file"; filename="test.txt"`},
	})
	part.Write([]byte("test content"))
	writer.WriteField("note_id", strconv.Itoa(noteID))
	writer.Close()

	req2, _ := http.NewRequest("POST", s.baseURL+"/upload", body)
	req2.Header.Set("Authorization", "Bearer "+s.ownerToken)
	req2.Header.Set("Content-Type", writer.FormDataContentType())
	client2 := &http.Client{}
	resp2, err2 := client2.Do(req2)
	s.NoError(err2)
	defer resp2.Body.Close()
	s.Equal(http.StatusCreated, resp2.StatusCode)
}

func (s *E2ETestSuite) Test40_GetFileForbidden() {
	req, _ := http.NewRequest("GET", s.baseURL+"/files/some-random-id", nil)
	req.Header.Set("Authorization", "Bearer "+s.ownerToken)
	client := &http.Client{}
	resp, err := client.Do(req)
	s.NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusNotFound, resp.StatusCode)
}

func (s *E2ETestSuite) Test41_UploadAndGetFile() {
	// Create a new note for this test
	reqBody := map[string]interface{}{
		"text":     "Note for upload and get file test",
		"tags":     []string{"upload", "get", "test"},
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
	var noteID int
	if noteResp["success"] != nil && noteResp["success"].(bool) {
		noteData := noteResp["data"].(map[string]interface{})
		noteID = int(noteData["id"].(float64))
		s.True(noteID > 0)
	} else {
		s.Fail("Note creation failed")
		return
	}

	// Test uploading and then retrieving a file
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Create the file part with explicit Content-Type
	part, _ := writer.CreatePart(map[string][]string{
		"Content-Type":        {"text/plain"},
		"Content-Disposition": {`form-data; name="file"; filename="test.txt"`},
	})
	part.Write([]byte("test content"))
	writer.WriteField("note_id", strconv.Itoa(noteID))
	writer.Close()

	req2, _ := http.NewRequest("POST", s.baseURL+"/upload", body)
	req2.Header.Set("Authorization", "Bearer "+s.ownerToken)
	req2.Header.Set("Content-Type", writer.FormDataContentType())
	client2 := &http.Client{}
	resp2, err2 := client2.Do(req2)
	s.NoError(err2)
	defer resp2.Body.Close()
	s.Equal(http.StatusCreated, resp2.StatusCode)

	// Parse response to get attachment ID
	var response map[string]interface{}
	json.NewDecoder(resp2.Body).Decode(&response)
	s.True(response["success"].(bool))

	// Check if data exists before accessing it
	if response["data"] != nil {
		data := response["data"].(map[string]interface{})
		attachmentID := data["attachment_id"].(string)
		s.NotEmpty(attachmentID)

		// Test retrieving the file
		req3, _ := http.NewRequest("GET", s.baseURL+"/files/"+attachmentID, nil)
		req3.Header.Set("Authorization", "Bearer "+s.ownerToken)
		resp3, err3 := client2.Do(req3)
		s.NoError(err3)
		defer resp3.Body.Close()
		s.Equal(http.StatusOK, resp3.StatusCode)
	} else {
		s.Fail("Upload response does not contain data")
	}
}
