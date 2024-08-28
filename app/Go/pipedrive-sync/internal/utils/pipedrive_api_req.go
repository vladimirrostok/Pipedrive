package utils

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

type personsSearchResult struct {
	Success  bool            `json:"success"`
	Data     []person        `json:"data"`
	Addition additional_data `json:"additional_data"`
}

type person struct {
	Id             int    `json:"id"`
	RealCustomerId string `json:"e9b0195d8a37ff2ffa0ed1e320838907e34c3140"`
}

type dealsSearchResult struct {
	Success  bool            `json:"success"`
	Data     []deal          `json:"data"`
	Addition additional_data `json:"additional_data"`
}

type deal struct {
	ID          int    `json:"id"`
	RealOrderId string `json:"a966b2ff81e2e054362de2093428fbc2bb417beb"`
	Title       string `json:"title"`
	Value       int    `json:"value"`
}

type additional_data struct {
	Pagination pagination `json:"pagination"`
}

type pagination struct {
	Start                 int  `json:"start"`
	Limit                 int  `json:"limit"`
	MoreItemsInCollection bool `json:"more_items_in_collection"`
	NextStart             bool `json:"next_start"`
}

func getAllPersons(client HttpClient, errChan chan error) personsSearchResult {
	req, err := http.NewRequest("GET", "https://api.pipedrive.com/v1/persons", nil)
	if err != nil {
		errChan <- err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	q := req.URL.Query()
	q.Add("start", "0")
	q.Add("limit", "5000")
	q.Add("api_token", os.Getenv("PIPEDRIVE_API_KEY"))

	req.URL.RawQuery = q.Encode()

	resp, err := client.Do(req)
	if err != nil {
		errChan <- fmt.Errorf("error in the worker job requesting persons %s ", err)
	}

	defer resp.Body.Close()

	var obj personsSearchResult
	if err := json.NewDecoder(resp.Body).Decode(&obj); err != nil {
		errChan <- fmt.Errorf("error in the worker decoding persons %s ", err)
	}

	return obj
}

func getAllDeals(client HttpClient, errChan chan error) dealsSearchResult {
	req, err := http.NewRequest("GET", "https://api.pipedrive.com/v1/deals", nil)
	if err != nil {
		errChan <- err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	q := req.URL.Query()
	q.Add("start", "0")
	q.Add("limit", "5000")
	q.Add("api_token", os.Getenv("PIPEDRIVE_API_KEY"))

	req.URL.RawQuery = q.Encode()

	resp, err := client.Do(req)
	if err != nil {
		errChan <- fmt.Errorf("error in the worker job requesting deals %s ", err)
	}

	defer resp.Body.Close()

	var obj dealsSearchResult
	if err := json.NewDecoder(resp.Body).Decode(&obj); err != nil {
		errChan <- fmt.Errorf("error in the worker decoding deals %s ", err)
	}

	return obj
}
