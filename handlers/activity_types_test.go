package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"
)

func (s *E2ETestSuite) Test20_CreateActivityType_Success() {
	body := map[string]interface{}{
		"name":        "money",
		"value_type":  "float",
		"unit":        "RUR",
		"min_value":   -1000,
		"max_value":   1000,
		"aggregation": "sum",
		"category_id": nil,
	}
	jsonBody, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", s.baseURL+"/spaces/"+strconv.Itoa(s.createdSpaceID)+"/activity-types", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.ownerToken)
	client := &http.Client{}
	resp, err := client.Do(req)
	s.NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusCreated, resp.StatusCode)
}

func (s *E2ETestSuite) Test21_CreateActivityType_MinValueError() {
	body := map[string]interface{}{
		"name":        "invalid_range",
		"value_type":  "integer",
		"min_value":   100,
		"max_value":   50,
		"aggregation": "sum",
	}
	jsonBody, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", s.baseURL+"/spaces/"+strconv.Itoa(s.createdSpaceID)+"/activity-types", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.ownerToken)
	client := &http.Client{}
	resp, err := client.Do(req)
	s.NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusBadRequest, resp.StatusCode)
}

func (s *E2ETestSuite) Test22_DeleteActivityType() {
	body := map[string]interface{}{
		"name":        "money",
		"value_type":  "float",
		"aggregation": "sum",
	}
	jsonBody, _ := json.Marshal(body)
	reqCreate, _ := http.NewRequest("POST", s.baseURL+"/spaces/"+strconv.Itoa(s.createdSpaceID)+"/activity-types", bytes.NewBuffer(jsonBody))
	reqCreate.Header.Set("Authorization", "Bearer "+s.ownerToken)
	reqCreate.Header.Set("Content-Type", "application/json")
	client := &http.Client{}

	respCreate, err := client.Do(reqCreate)
	s.NoError(err)
	if err != nil {
		return
	}
	defer respCreate.Body.Close()

	if respCreate.StatusCode == http.StatusCreated {
		var respData map[string]interface{}
		json.NewDecoder(respCreate.Body).Decode(&respData)
		typeID := int(respData["id"].(float64))

		reqDel, _ := http.NewRequest("PATCH", s.baseURL+"/spaces/"+strconv.Itoa(s.createdSpaceID)+"/activity-types/"+strconv.Itoa(typeID)+"/delete", nil)
		reqDel.Header.Set("Authorization", "Bearer "+s.ownerToken)
		respDel, delErr := client.Do(reqDel)
		s.NoError(delErr)
		if delErr != nil {
			return
		}
		defer respDel.Body.Close()
		s.Equal(http.StatusNoContent, respDel.StatusCode)
	}
}

func (s *E2ETestSuite) Test23_RestoreActivityType() {
	// Create a type to restore
	body := map[string]interface{}{
		"name":        "restorable",
		"value_type":  "boolean",
		"aggregation": "and",
	}
	jsonBody, _ := json.Marshal(body)
	reqCreate, _ := http.NewRequest("POST", s.baseURL+"/spaces/"+strconv.Itoa(s.createdSpaceID)+"/activity-types", bytes.NewBuffer(jsonBody))
	reqCreate.Header.Set("Authorization", "Bearer "+s.ownerToken)
	reqCreate.Header.Set("Content-Type", "application/json")
	client := &http.Client{}

	respCreate, err := client.Do(reqCreate)
	s.NoError(err)
	if err != nil {
		return
	}
	defer respCreate.Body.Close()
	s.Equal(http.StatusCreated, respCreate.StatusCode)

	var respData map[string]interface{}
	json.NewDecoder(respCreate.Body).Decode(&respData)
	typeID := int(respData["id"].(float64))

	// Delete it
	reqDel, _ := http.NewRequest("PATCH", s.baseURL+"/spaces/"+strconv.Itoa(s.createdSpaceID)+"/activity-types/"+strconv.Itoa(typeID)+"/delete", nil)
	reqDel.Header.Set("Authorization", "Bearer "+s.ownerToken)
	respDel, delErr := client.Do(reqDel)
	s.NoError(delErr)
	if delErr != nil {
		return
	}
	defer respDel.Body.Close()
	s.Equal(http.StatusNoContent, respDel.StatusCode)

	// Restore it
	reqRestore, _ := http.NewRequest("PATCH", s.baseURL+"/spaces/"+strconv.Itoa(s.createdSpaceID)+"/activity-types/"+strconv.Itoa(typeID)+"/restore", nil)
	reqRestore.Header.Set("Authorization", "Bearer "+s.ownerToken)
	respRestore, restoreErr := client.Do(reqRestore)
	s.NoError(restoreErr)
	if restoreErr != nil {
		return
	}
	defer respRestore.Body.Close()
	s.Equal(http.StatusNoContent, respRestore.StatusCode)
}

func (s *E2ETestSuite) Test24_CreateActivityType_DuplicateName() {
	body := map[string]interface{}{
		"name":        "duplicate_test",
		"value_type":  "text",
		"aggregation": "count",
	}
	jsonBody, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", s.baseURL+"/spaces/"+strconv.Itoa(s.createdSpaceID)+"/activity-types", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.ownerToken)
	client := &http.Client{}
	resp, err := client.Do(req)
	s.NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusCreated, resp.StatusCode)

	// Try again with same name
	req2, _ := http.NewRequest("POST", s.baseURL+"/spaces/"+strconv.Itoa(s.createdSpaceID)+"/activity-types", bytes.NewBuffer(jsonBody))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+s.ownerToken)
	resp2, err2 := client.Do(req2)
	s.NoError(err2)
	defer resp2.Body.Close()
	s.Equal(http.StatusBadRequest, resp2.StatusCode)
}

func (s *E2ETestSuite) Test25_InvalidValueType() {
	body := map[string]interface{}{
		"name":        "invalid_type",
		"value_type":  "unknown",
		"aggregation": "sum",
	}
	jsonBody, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", s.baseURL+"/spaces/"+strconv.Itoa(s.createdSpaceID)+"/activity-types", bytes.NewBuffer(jsonBody))
	req.Header.Set("Authorization", "Bearer "+s.ownerToken)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	s.NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusBadRequest, resp.StatusCode)
}

func (s *E2ETestSuite) Test26_InvalidAggregation() {
	body := map[string]interface{}{
		"name":        "invalid_agg",
		"value_type":  "float",
		"aggregation": "sum_everything",
	}
	jsonBody, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", s.baseURL+"/spaces/"+strconv.Itoa(s.createdSpaceID)+"/activity-types", bytes.NewBuffer(jsonBody))
	req.Header.Set("Authorization", "Bearer "+s.ownerToken)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	s.NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusBadRequest, resp.StatusCode)
}

func (s *E2ETestSuite) Test27_TimeWithNullUnit_Valid() {
	body := map[string]interface{}{
		"name":        "sleep_duration",
		"value_type":  "time",
		"aggregation": "sum",
		"unit":        nil,
	}
	jsonBody, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", s.baseURL+"/spaces/"+strconv.Itoa(s.createdSpaceID)+"/activity-types", bytes.NewBuffer(jsonBody))
	req.Header.Set("Authorization", "Bearer "+s.ownerToken)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	s.NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusCreated, resp.StatusCode)
}

func (s *E2ETestSuite) Test28_TimeWithUnit_Invalid() {
	unit := "minutes"
	body := map[string]interface{}{
		"name":        "sleep_with_unit",
		"value_type":  "time",
		"aggregation": "sum",
		"unit":        unit,
	}
	jsonBody, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", s.baseURL+"/spaces/"+strconv.Itoa(s.createdSpaceID)+"/activity-types", bytes.NewBuffer(jsonBody))
	req.Header.Set("Authorization", "Bearer "+s.ownerToken)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	s.NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusBadRequest, resp.StatusCode)
}

func (s *E2ETestSuite) Test29_BooleanWithInvalidAggregation() {
	body := map[string]interface{}{
		"name":        "bool_avg",
		"value_type":  "boolean",
		"aggregation": "avg",
	}
	jsonBody, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", s.baseURL+"/spaces/"+strconv.Itoa(s.createdSpaceID)+"/activity-types", bytes.NewBuffer(jsonBody))
	req.Header.Set("Authorization", "Bearer "+s.ownerToken)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	s.NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusBadRequest, resp.StatusCode)
}

func (s *E2ETestSuite) Test30_TextWithInvalidAggregation() {
	body := map[string]interface{}{
		"name":        "text_invalid_agg",
		"value_type":  "text",
		"aggregation": "sum",
	}
	jsonBody, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", s.baseURL+"/spaces/"+strconv.Itoa(s.createdSpaceID)+"/activity-types", bytes.NewBuffer(jsonBody))
	req.Header.Set("Authorization", "Bearer "+s.ownerToken)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	s.NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusBadRequest, resp.StatusCode)
}
