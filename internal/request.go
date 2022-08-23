package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

const (
	devUrl  = "http://localhost:11111/api/v1"
	prodUrl = "http://EQNS-Dev-mcls-1402262411.ap-northeast-2.elb.amazonaws.com/api/v1"
)

type requestCertificateForm struct {
	PublicKey []byte `json:"publicKey"`
	Time      string `json:"time"`
}

type responseCertificateForm struct {
	Certificate []byte `json:"certificate"`
}

func GetCertificate(publicKey []byte, time string) ([]byte, error) {
	requestCertificate := &requestCertificateForm{
		PublicKey: publicKey,
		Time:      time,
	}

	requestCertificateByte, err := json.Marshal(requestCertificate)
	if err != nil {
		return nil, err
	}

	path := "auth"
	url := fmt.Sprintf("%s/%s", prodUrl, path)
	response, err := http.Post(url, "application/json", bytes.NewBuffer(requestCertificateByte))
	if err != nil {
		return nil, err
	}

	var res responseCertificateForm
	json.NewDecoder(response.Body).Decode(&res)

	return res.Certificate, nil
}
