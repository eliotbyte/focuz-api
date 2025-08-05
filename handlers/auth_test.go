package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
)

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

	var data map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&data)
	if data["success"] != nil && data["success"].(bool) {
		tokenData := data["data"].(map[string]interface{})
		s.ownerToken = tokenData["token"].(string)
		s.NotEmpty(s.ownerToken)
	} else {
		s.Fail("Login failed")
	}
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

	var data map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&data)
	if data["success"] != nil && data["success"].(bool) {
		tokenData := data["data"].(map[string]interface{})
		s.guestToken = tokenData["token"].(string)
		s.NotEmpty(s.guestToken)
	} else {
		s.Fail("Login failed")
	}
}
