package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

func (s *E2ETestSuite) Test40_CreateActivityValid() {
	noteReq := map[string]interface{}{
		"text":    "Test note for activity",
		"tags":    []string{"test"},
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
	s.True(noteData["success"].(bool))
	noteResponseData := noteData["data"].(map[string]interface{})
	noteID := int(noteResponseData["id"].(float64))

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
	s.True(actData["success"].(bool))
	actResponseData := actData["data"].(map[string]interface{})
	s.True(actResponseData["id"].(float64) > 0)
	s.NotEmpty(actResponseData["createdAt"])
	s.NotEmpty(actResponseData["modifiedAt"])

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
	s.True(respData["success"].(bool))
	activityResponseData := respData["data"].(map[string]interface{})
	activityID := int(activityResponseData["id"].(float64))

	reqDel, _ := http.NewRequest("PATCH", s.baseURL+"/activities/"+strconv.Itoa(activityID)+"/delete", nil)
	reqDel.Header.Set("Authorization", "Bearer "+s.ownerToken)
	respDel, errDel := client.Do(reqDel)
	s.NoError(errDel)
	defer respDel.Body.Close()
	s.Equal(http.StatusOK, respDel.StatusCode)

	reqRestore, _ := http.NewRequest("PATCH", s.baseURL+"/activities/"+strconv.Itoa(activityID)+"/restore", nil)
	reqRestore.Header.Set("Authorization", "Bearer "+s.ownerToken)
	respRestore, errRestore := client.Do(reqRestore)
	s.NoError(errRestore)
	defer respRestore.Body.Close()
	s.Equal(http.StatusOK, respRestore.StatusCode)
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
	s.True(respData["success"].(bool))
	activityResponseData := respData["data"].(map[string]interface{})
	activityID := int(activityResponseData["id"].(float64))

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
	s.Equal(http.StatusOK, respUpdate.StatusCode)
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
	s.True(spaceResp["success"].(bool))
	spaceResponseData := spaceResp["data"].(map[string]interface{})
	newSpaceID := int(spaceResponseData["id"].(float64))

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
	s.True(privateNote["success"].(bool))
	privateNoteResponseData := privateNote["data"].(map[string]interface{})
	privateNoteID := int(privateNoteResponseData["id"].(float64))

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
	var data map[string]interface{}
	json.NewDecoder(respLogin.Body).Decode(&data)
	s.True(data["success"].(bool))
	tokenData := data["data"].(map[string]interface{})
	outsiderToken := tokenData["token"].(string)

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

func (s *E2ETestSuite) Test46_GetActivitiesAnalysis() {
	client := &http.Client{}

	now := time.Now().AddDate(0, 0, -1)
	reqBody1 := map[string]interface{}{
		"text":    "Note 1 for analysis",
		"tags":    []string{"analysis"},
		"topicId": s.createdTopicID,
		"date":    now.Format("2006-01-02"),
	}
	b1, _ := json.Marshal(reqBody1)
	reqN1, _ := http.NewRequest("POST", s.baseURL+"/notes", bytes.NewBuffer(b1))
	reqN1.Header.Set("Authorization", "Bearer "+s.ownerToken)
	reqN1.Header.Set("Content-Type", "application/json")
	respN1, err := client.Do(reqN1)
	s.Require().NoError(err)
	defer respN1.Body.Close()
	var nd1 map[string]interface{}
	json.NewDecoder(respN1.Body).Decode(&nd1)
	s.True(nd1["success"].(bool))
	note1ResponseData := nd1["data"].(map[string]interface{})
	note1ID := int(note1ResponseData["id"].(float64))

	actReq1 := map[string]interface{}{
		"typeId":  1,
		"value":   "5",
		"note_id": note1ID,
	}
	a1, _ := json.Marshal(actReq1)
	reqA1, _ := http.NewRequest("POST", s.baseURL+"/activities", bytes.NewBuffer(a1))
	reqA1.Header.Set("Authorization", "Bearer "+s.ownerToken)
	reqA1.Header.Set("Content-Type", "application/json")
	client.Do(reqA1)

	reqBody2 := map[string]interface{}{
		"text":    "Note 2 for analysis",
		"tags":    []string{"analysis"},
		"topicId": s.createdTopicID,
		"date":    now.Format("2006-01-02"),
	}
	b2, _ := json.Marshal(reqBody2)
	reqN2, _ := http.NewRequest("POST", s.baseURL+"/notes", bytes.NewBuffer(b2))
	reqN2.Header.Set("Authorization", "Bearer "+s.ownerToken)
	reqN2.Header.Set("Content-Type", "application/json")
	respN2, err := client.Do(reqN2)
	s.Require().NoError(err)
	defer respN2.Body.Close()
	var nd2 map[string]interface{}
	json.NewDecoder(respN2.Body).Decode(&nd2)
	s.True(nd2["success"].(bool))
	note2ResponseData := nd2["data"].(map[string]interface{})
	note2ID := int(note2ResponseData["id"].(float64))

	actReq2 := map[string]interface{}{
		"typeId":  1,
		"value":   "6",
		"note_id": note2ID,
	}
	a2, _ := json.Marshal(actReq2)
	reqA2, _ := http.NewRequest("POST", s.baseURL+"/activities", bytes.NewBuffer(a2))
	reqA2.Header.Set("Authorization", "Bearer "+s.ownerToken)
	reqA2.Header.Set("Content-Type", "application/json")
	client.Do(reqA2)

	url := s.baseURL + "/activities?spaceId=" + strconv.Itoa(s.createdSpaceID) + "&typeId=1&aggregationPeriod=day"
	r, _ := http.NewRequest("GET", url, nil)
	r.Header.Set("Authorization", "Bearer "+s.ownerToken)
	resp, err := client.Do(r)
	s.NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusOK, resp.StatusCode)

	var response map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&response)
	s.True(response["success"].(bool))
	analysis := response["data"].([]interface{})
	s.True(len(analysis) >= 1)
}

func (s *E2ETestSuite) Test47_CannotCreateDuplicateActivityOnSameNote() {
	actReq := map[string]interface{}{
		"typeId":  1,
		"value":   "2",
		"note_id": s.createdNoteID,
	}
	actJSON, _ := json.Marshal(actReq)
	client := &http.Client{}

	req1, _ := http.NewRequest("POST", s.baseURL+"/activities", bytes.NewBuffer(actJSON))
	req1.Header.Set("Authorization", "Bearer "+s.ownerToken)
	req1.Header.Set("Content-Type", "application/json")
	resp1, err1 := client.Do(req1)
	s.NoError(err1)
	defer resp1.Body.Close()
	s.Equal(http.StatusCreated, resp1.StatusCode)

	req2, _ := http.NewRequest("POST", s.baseURL+"/activities", bytes.NewBuffer(actJSON))
	req2.Header.Set("Authorization", "Bearer "+s.ownerToken)
	req2.Header.Set("Content-Type", "application/json")
	resp2, err2 := client.Do(req2)
	s.NoError(err2)
	defer resp2.Body.Close()
	s.Equal(http.StatusBadRequest, resp2.StatusCode)
}

func (s *E2ETestSuite) Test48_CanCreateActivityWhenExistingIsDeleted() {
	client := &http.Client{}

	actReq := map[string]interface{}{
		"typeId":  1,
		"value":   "4",
		"note_id": s.createdNoteID,
	}
	actJSON, _ := json.Marshal(actReq)

	reqCreate, _ := http.NewRequest("POST", s.baseURL+"/activities", bytes.NewBuffer(actJSON))
	reqCreate.Header.Set("Authorization", "Bearer "+s.ownerToken)
	reqCreate.Header.Set("Content-Type", "application/json")
	respCreate, errCreate := client.Do(reqCreate)
	s.NoError(errCreate)
	defer respCreate.Body.Close()
	s.Equal(http.StatusCreated, respCreate.StatusCode)

	var data map[string]interface{}
	json.NewDecoder(respCreate.Body).Decode(&data)
	s.True(data["success"].(bool))
	activityResponseData := data["data"].(map[string]interface{})
	activityID := int(activityResponseData["id"].(float64))

	reqDel, _ := http.NewRequest("PATCH", s.baseURL+"/activities/"+strconv.Itoa(activityID)+"/delete", nil)
	reqDel.Header.Set("Authorization", "Bearer "+s.ownerToken)
	respDel, errDel := client.Do(reqDel)
	s.NoError(errDel)
	defer respDel.Body.Close()
	s.Equal(http.StatusOK, respDel.StatusCode)

	reqCreate2, _ := http.NewRequest("POST", s.baseURL+"/activities", bytes.NewBuffer(actJSON))
	reqCreate2.Header.Set("Authorization", "Bearer "+s.ownerToken)
	reqCreate2.Header.Set("Content-Type", "application/json")
	respCreate2, errCreate2 := client.Do(reqCreate2)
	s.NoError(errCreate2)
	defer respCreate2.Body.Close()
	s.Equal(http.StatusCreated, respCreate2.StatusCode)
}
