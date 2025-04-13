package handlers

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
)

func (s *E2ETestSuite) Test20_UploadFileInvalidNote() {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("note_id", "99999")
	part, _ := writer.CreateFormFile("file", "test.pdf")
	part.Write([]byte("fake pdf data"))
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

func (s *E2ETestSuite) Test21_UploadFile() {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("note_id", strconv.Itoa(s.createdNoteID))
	part, _ := writer.CreateFormFile("file", "image.png")
	part.Write([]byte("fake image data"))
	writer.Close()

	req, _ := http.NewRequest("POST", s.baseURL+"/upload", body)
	req.Header.Set("Authorization", "Bearer "+s.ownerToken)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	client := &http.Client{}
	resp, err := client.Do(req)
	s.NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusOK, resp.StatusCode)
}

func (s *E2ETestSuite) Test22_GetFileForbidden() {
	req, _ := http.NewRequest("GET", s.baseURL+"/files/some-random-id", nil)
	req.Header.Set("Authorization", "Bearer "+s.guestToken)
	client := &http.Client{}
	resp, err := client.Do(req)
	s.NoError(err)
	defer resp.Body.Close()
	s.True(resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusNotFound)
}

func (s *E2ETestSuite) Test23_UploadAndGetFile() {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("note_id", strconv.Itoa(s.createdNoteID))
	part, _ := writer.CreateFormFile("file", "document.pdf")
	part.Write([]byte("fake pdf data"))
	writer.Close()

	req, _ := http.NewRequest("POST", s.baseURL+"/upload", body)
	req.Header.Set("Authorization", "Bearer "+s.ownerToken)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	client := &http.Client{}
	resp, err := client.Do(req)
	s.NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusOK, resp.StatusCode)

	// read attachment_id
	buf := new(bytes.Buffer)
	io.Copy(buf, resp.Body)
	// naive parse, just check we got a 200 OK and an attachment id field
	s.Contains(buf.String(), "attachment_id")
}
