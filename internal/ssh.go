package internal

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"github.com/mitchellh/go-homedir"
	"io/ioutil"
	"log"

	"golang.org/x/crypto/ssh"
)

func CreateNewCAKey() {
	dir, err := homedir.Dir()
	if err != nil {
		panic(WrapError(err))
	}

	savePrivateFileTo := fmt.Sprintf("%s/%s", dir, ".ssh/id_rsa")
	savePublicFileTo := fmt.Sprintf("%s/%s", dir, ".ssh/id_rsa.pub")

	bitSize := 3072

	privateKey, err := generatePrivateKey(bitSize)
	if err != nil {
		log.Fatal(err.Error())
	}

	_, publicKeyBytes, err := generatePublicKey(&privateKey.PublicKey)
	if err != nil {
		log.Fatal(err.Error())
	}

	privateKeyBytes := encodePrivateKeyToPem(privateKey)

	err = writeKeyToFile(privateKeyBytes, savePrivateFileTo)
	if err != nil {
		log.Fatal(err.Error())
	}

	err = writeKeyToFile(publicKeyBytes, savePublicFileTo)
	if err != nil {
		log.Fatal(err.Error())
	}
}

func generatePrivateKey(bitSize int) (*rsa.PrivateKey, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, bitSize)
	if err != nil {
		return nil, err
	}

	err = privateKey.Validate()
	if err != nil {
		return nil, err
	}

	log.Println("Private Key generated")
	return privateKey, nil
}

func encodePrivateKeyToPem(privateKey *rsa.PrivateKey) []byte {
	privDER := x509.MarshalPKCS1PrivateKey(privateKey)

	privBlock := pem.Block{
		Type:    "RSA PRIVATE KEY",
		Headers: nil,
		Bytes:   privDER,
	}

	privatePem := pem.EncodeToMemory(&privBlock)

	return privatePem
}

func generatePublicKey(privateKey *rsa.PublicKey) (ssh.PublicKey, []byte, error) {
	publicRsaKey, err := ssh.NewPublicKey(privateKey)
	if err != nil {
		return nil, nil, err
	}
	pubKeyBytes := ssh.MarshalAuthorizedKey(publicRsaKey)

	log.Println("Public Key generated")

	return publicRsaKey, pubKeyBytes, nil
}

func writeKeyToFile(keyBytes []byte, saveFileTo string) error {
	err := ioutil.WriteFile(saveFileTo, keyBytes, 0600)
	if err != nil {
		return err
	}

	log.Printf("Key saved to: %s", saveFileTo)
	return nil
}
