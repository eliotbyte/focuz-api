package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"
)

func (s *E2ETestSuite) Test40_CreateActivityValid() {
	noteReq := map[string]interface{}{
		"text":    "Activity note test",
		"topicId": s.createdTopicID,
	}
	noteJSON, _ := json.Marshal(noteReq)
	createNoteReq, _ := http.NewRequest("POST", s.baseURL+"/notes", bytes.NewBuffer(noteJSON))
	createNoteReq.Header.Set("Authorization", "Bearer "+s.ownerToken)
	createNoteReq.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	noteResp, err := client.Do(createNoteReq)
	s.NoError(err)
	defer noteResp.Body.Close()
	s.Equal(http.StatusCreated, noteResp.StatusCode)
	var noteData map[string]interface{}
	json.NewDecoder(noteResp.Body).Decode(&noteData)
	noteID := int(noteData["id"].(float64))

	actReq := map[string]interface{}{
		"typeId":  1,
		"value":   "7",
		"note_id": noteID,
	}
	actJSON, _ := json.Marshal(actReq)
	createActReq, _ := http.NewRequest("POST", s.baseURL+"/activities", bytes.NewBuffer(actJSON))
	createActReq.Header.Set("Authorization", "Bearer "+s.ownerToken)
	createActReq.Header.Set("Content-Type", "application/json")
	actResp, err := client.Do(createActReq)
	s.NoError(err)
	defer actResp.Body.Close()
	s.Equal(http.StatusCreated, actResp.StatusCode)
	var actData map[string]interface{}
	json.NewDecoder(actResp.Body).Decode(&actData)
	s.True(actData["id"].(float64) > 0)
	s.NotEmpty(actData["createdAt"])
	s.NotEmpty(actData["modifiedAt"])

	s.createdNoteID = noteID
}

func (s *E2ETestSuite) Test41_CreateActivityOutOfRange() {
	actReq := map[string]interface{}{
		"typeId":  1,
		"value":   "9999",
		"note_id": s.createdNoteID,
	}
	actJSON, _ := json.Marshal(actReq)
	req, _ := http.NewRequest("POST", s.baseURL+"/activities", bytes.NewBuffer(actJSON))
	req.Header.Set("Authorization", "Bearer "+s.ownerToken)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	s.NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusBadRequest, resp.StatusCode)
}

func (s *E2ETestSuite) Test42_CreateActivityInvalidType() {
	actReq := map[string]interface{}{
		"typeId":  999999,
		"value":   "whatever",
		"note_id": s.createdNoteID,
	}
	actJSON, _ := json.Marshal(actReq)
	req, _ := http.NewRequest("POST", s.baseURL+"/activities", bytes.NewBuffer(actJSON))
	req.Header.Set("Authorization", "Bearer "+s.ownerToken)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	s.NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusBadRequest, resp.StatusCode)
}

func (s *E2ETestSuite) Test43_DeleteRestoreActivity() {
	actReq := map[string]interface{}{
		"typeId":  1,
		"value":   "3",
		"note_id": s.createdNoteID,
	}
	actJSON, _ := json.Marshal(actReq)
	client := &http.Client{}
	reqCreate, _ := http.NewRequest("POST", s.baseURL+"/activities", bytes.NewBuffer(actJSON))
	reqCreate.Header.Set("Authorization", "Bearer "+s.ownerToken)
	reqCreate.Header.Set("Content-Type", "application/json")
	respCreate, errCreate := client.Do(reqCreate)
	s.NoError(errCreate)
	defer respCreate.Body.Close()
	s.Equal(http.StatusCreated, respCreate.StatusCode)

	var respData map[string]interface{}
	json.NewDecoder(respCreate.Body).Decode(&respData)
	activityID := int(respData["id"].(float64))

	reqDel, _ := http.NewRequest("PATCH", s.baseURL+"/activities/"+strconv.Itoa(activityID)+"/delete", nil)
	reqDel.Header.Set("Authorization", "Bearer "+s.ownerToken)
	respDel, errDel := client.Do(reqDel)
	s.NoError(errDel)
	defer respDel.Body.Close()
	s.Equal(http.StatusNoContent, respDel.StatusCode)

	reqRestore, _ := http.NewRequest("PATCH", s.baseURL+"/activities/"+strconv.Itoa(activityID)+"/restore", nil)
	reqRestore.Header.Set("Authorization", "Bearer "+s.ownerToken)
	respRestore, errRestore := client.Do(reqRestore)
	s.NoError(errRestore)
	defer respRestore.Body.Close()
	s.Equal(http.StatusNoContent, respRestore.StatusCode)
}

func (s *E2ETestSuite) Test44_UpdateActivity() {
	actReq := map[string]interface{}{
		"typeId":  1,
		"value":   "5",
		"note_id": s.createdNoteID,
	}
	actJSON, _ := json.Marshal(actReq)
	client := &http.Client{}
	reqCreate, _ := http.NewRequest("POST", s.baseURL+"/activities", bytes.NewBuffer(actJSON))
	reqCreate.Header.Set("Authorization", "Bearer "+s.ownerToken)
	reqCreate.Header.Set("Content-Type", "application/json")
	respCreate, errCreate := client.Do(reqCreate)
	s.NoError(errCreate)
	defer respCreate.Body.Close()
	s.Equal(http.StatusCreated, respCreate.StatusCode)

	var respData map[string]interface{}
	json.NewDecoder(respCreate.Body).Decode(&respData)
	activityID := int(respData["id"].(float64))

	updateReq := map[string]interface{}{
		"value":   "8",
		"note_id": s.createdNoteID,
	}
	updateJSON, _ := json.Marshal(updateReq)
	reqUpdate, _ := http.NewRequest("PATCH", s.baseURL+"/activities/"+strconv.Itoa(activityID), bytes.NewBuffer(updateJSON))
	reqUpdate.Header.Set("Authorization", "Bearer "+s.ownerToken)
	reqUpdate.Header.Set("Content-Type", "application/json")
	respUpdate, errUpdate := client.Do(reqUpdate)
	s.NoError(errUpdate)
	defer respUpdate.Body.Close()
	s.Equal(http.StatusNoContent, respUpdate.StatusCode)
}

func (s *E2ETestSuite) Test45_CreateActivityInaccessibleNote() {
	body := map[string]interface{}{
		"name": "Outside Space",
	}
	b, _ := json.Marshal(body)
	client := &http.Client{}
	reqSpace, _ := http.NewRequest("POST", s.baseURL+"/spaces", bytes.NewBuffer(b))
	reqSpace.Header.Set("Authorization", "Bearer "+s.ownerToken)
	reqSpace.Header.Set("Content-Type", "application/json")
	respSpace, errSpace := client.Do(reqSpace)
	s.NoError(errSpace)
	defer respSpace.Body.Close()

	var spaceResp map[string]interface{}
	json.NewDecoder(respSpace.Body).Decode(&spaceResp)
	newSpaceID := int(spaceResp["id"].(float64))

	reqNote := map[string]interface{}{
		"text":    "Private note",
		"topicId": s.createTopic("Private Topic", 1),
	}
	noteJSON, _ := json.Marshal(reqNote)
	reqNoteCreate, _ := http.NewRequest("POST", s.baseURL+"/notes", bytes.NewBuffer(noteJSON))
	reqNoteCreate.Header.Set("Authorization", "Bearer "+s.ownerToken)
	reqNoteCreate.Header.Set("Content-Type", "application/json")
	respNoteCreate, errNoteCreate := client.Do(reqNoteCreate)
	if s.NoError(errNoteCreate) {
		defer respNoteCreate.Body.Close()
	}
	var privateNote map[string]interface{}
	json.NewDecoder(respNoteCreate.Body).Decode(&privateNote)
	privateNoteID := int(privateNote["id"].(float64))

	bodyUser := `{"username":"outsider","password":"outsiderpass"}`
	respUser, errUser := http.Post(s.baseURL+"/register", "application/json", bytes.NewBuffer([]byte(bodyUser)))
	s.NoError(errUser)
	defer respUser.Body.Close()

	loginBody := `{"username":"outsider","password":"outsiderpass"}`
	loginReq, _ := http.NewRequest("POST", s.baseURL+"/login", bytes.NewBuffer([]byte(loginBody)))
	loginReq.Header.Set("Content-Type", "application/json")
	respLogin, errLogin := client.Do(loginReq)
	s.NoError(errLogin)
	defer respLogin.Body.Close()
	var data map[string]string
	json.NewDecoder(respLogin.Body).Decode(&data)
	outsiderToken := data["token"]

	actReq := map[string]interface{}{
		"typeId":  1,
		"value":   "5",
		"note_id": privateNoteID,
	}
	actJSON, _ := json.Marshal(actReq)
	reqCreateAct, _ := http.NewRequest("POST", s.baseURL+"/activities", bytes.NewBuffer(actJSON))
	reqCreateAct.Header.Set("Authorization", "Bearer "+outsiderToken)
	reqCreateAct.Header.Set("Content-Type", "application/json")
	respCreateAct, errAct := client.Do(reqCreateAct)
	s.NoError(errAct)
	defer respCreateAct.Body.Close()
	s.Equal(http.StatusForbidden, respCreateAct.StatusCode)

	_ = newSpaceID
}
