package client

import (
	"bytes"
	"log"
	"net/http"
)

func requestGET(URI string, auth bool) (*http.Response, error) {

	req, err := http.NewRequest("GET", host+URI, nil)
	if err != nil {
		log.Println("[getResponseFromServer] new request error : ", err)
		return &http.Response{}, err
	}

	if auth {
		req.SetBasicAuth(myUser.UID, myUser.Secret)
	}

	c := http.DefaultClient
	res, err := c.Do(req)
	if err != nil {
		log.Println("[getResponseFromServer] do error : ", err)
		return &http.Response{}, err
	}
	return res, nil
}

func requestPOST(URI string, data []byte, auth bool) (*http.Response, error) {

	req, err := http.NewRequest("POST", host+URI, bytes.NewBuffer(data))
	if err != nil {
		log.Println("[requestPOST] new request error : ", err)
		return nil, err
	}
	req.Header.Set("content-type", "application/json")

	if auth {
		req.SetBasicAuth(myUser.UID, myUser.Secret)
	}

	c := http.DefaultClient
	res, err := c.Do(req)
	if err != nil {
		log.Println("[requestPOST] do error : ", err)
		return nil, err
	}

	return res, nil
}
