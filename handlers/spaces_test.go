package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"
)

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
	if spaceResp["success"] != nil && spaceResp["success"].(bool) {
		spaceData := spaceResp["data"].(map[string]interface{})
		id := int(spaceData["id"].(float64))
		s.createdSpaceID = id
		s.True(s.createdSpaceID > 0)
	} else {
		s.Fail("Space creation failed")
	}
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

func (s *E2ETestSuite) Test10_GetAccessibleSpaces() {
	req, _ := http.NewRequest("GET", s.baseURL+"/spaces", nil)
	req.Header.Set("Authorization", "Bearer "+s.ownerToken)
	client := &http.Client{}
	resp, err := client.Do(req)
	s.NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusOK, resp.StatusCode)

	var response map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&response)
	s.True(response["success"].(bool))
	spaces := response["data"].([]interface{})
	s.True(len(spaces) >= 1)

	found := false
	for _, sp := range spaces {
		space := sp.(map[string]interface{})
		if int(space["id"].(float64)) == s.createdSpaceID {
			found = true
			s.Contains(space, "name")
			s.Contains(space, "ownerId")
			s.Contains(space, "createdAt")
			s.Contains(space, "modifiedAt")
			break
		}
	}
	s.True(found)
}

func (s *E2ETestSuite) Test11_CreateParticipantAndInvite() {
	body := `{"username":"participant","password":"partpass"}`
	resp, err := http.Post(s.baseURL+"/register", "application/json", bytes.NewBuffer([]byte(body)))
	s.NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusCreated, resp.StatusCode)

	reqBody := map[string]int{"userId": 3}
	jsonBody, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", s.baseURL+"/spaces/"+strconv.Itoa(s.createdSpaceID)+"/invite", bytes.NewBuffer(jsonBody))
	req.Header.Set("Authorization", "Bearer "+s.ownerToken)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp2, err2 := client.Do(req)
	s.NoError(err2)
	defer resp2.Body.Close()
	s.Equal(http.StatusNoContent, resp2.StatusCode)
}

func (s *E2ETestSuite) Test12_RemoveUser_GuestCantRemove() {
	req, _ := http.NewRequest(
		"DELETE",
		s.baseURL+"/spaces/"+strconv.Itoa(s.createdSpaceID)+"/users/3",
		nil,
	)
	req.Header.Set("Authorization", "Bearer "+s.guestToken)
	client := &http.Client{}
	resp, err := client.Do(req)
	s.NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusForbidden, resp.StatusCode)
}

func (s *E2ETestSuite) Test13_RemoveUser_ParticipantNotFound() {
	req, _ := http.NewRequest(
		"DELETE",
		s.baseURL+"/spaces/"+strconv.Itoa(s.createdSpaceID)+"/users/9999",
		nil,
	)
	req.Header.Set("Authorization", "Bearer "+s.ownerToken)
	client := &http.Client{}
	resp, err := client.Do(req)
	s.NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusNotFound, resp.StatusCode)
}

func (s *E2ETestSuite) Test14_RemoveUser_CannotRemoveOwner() {
	req, _ := http.NewRequest(
		"DELETE",
		s.baseURL+"/spaces/"+strconv.Itoa(s.createdSpaceID)+"/users/1",
		nil,
	)
	req.Header.Set("Authorization", "Bearer "+s.ownerToken)
	client := &http.Client{}
	resp, err := client.Do(req)
	s.NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusForbidden, resp.StatusCode)
}

func (s *E2ETestSuite) Test15_RemoveUser_Success() {
	req, _ := http.NewRequest(
		"DELETE",
		s.baseURL+"/spaces/"+strconv.Itoa(s.createdSpaceID)+"/users/3",
		nil,
	)
	req.Header.Set("Authorization", "Bearer "+s.ownerToken)
	client := &http.Client{}
	resp, err := client.Do(req)
	s.NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusNoContent, resp.StatusCode)
}

func (s *E2ETestSuite) Test16_GetUsersInSpace_Success() {
	req, _ := http.NewRequest("GET", s.baseURL+"/spaces/"+strconv.Itoa(s.createdSpaceID)+"/users", nil)
	req.Header.Set("Authorization", "Bearer "+s.ownerToken)
	client := &http.Client{}
	resp, err := client.Do(req)
	s.NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusOK, resp.StatusCode)

	var response map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&response)
	s.True(response["success"].(bool))
	participants := response["data"].([]interface{})
	s.True(len(participants) >= 1)

	foundOwner := false
	for _, p := range participants {
		participant := p.(map[string]interface{})
		if int(participant["id"].(float64)) == 1 {
			foundOwner = true
			break
		}
	}
	s.True(foundOwner)
}

func (s *E2ETestSuite) Test17_GetUsersInSpace_ForbiddenForNonMember() {
	body := `{"username":"outsider","password":"outsiderpass"}`
	resp, err := http.Post(s.baseURL+"/register", "application/json", bytes.NewBuffer([]byte(body)))
	s.NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusCreated, resp.StatusCode)

	loginBody := `{"username":"outsider","password":"outsiderpass"}`
	loginReq, _ := http.NewRequest("POST", s.baseURL+"/login", bytes.NewBuffer([]byte(loginBody)))
	loginReq.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	loginResp, loginErr := client.Do(loginReq)
	s.NoError(loginErr)
	defer loginResp.Body.Close()
	s.Equal(http.StatusOK, loginResp.StatusCode)

	var data map[string]interface{}
	json.NewDecoder(loginResp.Body).Decode(&data)
	s.True(data["success"].(bool))
	tokenData := data["data"].(map[string]interface{})
	outsiderToken := tokenData["token"].(string)
	s.NotEmpty(outsiderToken)

	req, _ := http.NewRequest("GET", s.baseURL+"/spaces/"+strconv.Itoa(s.createdSpaceID)+"/users", nil)
	req.Header.Set("Authorization", "Bearer "+outsiderToken)
	resp2, err2 := client.Do(req)
	s.NoError(err2)
	defer resp2.Body.Close()
	s.Equal(http.StatusForbidden, resp2.StatusCode)
}
