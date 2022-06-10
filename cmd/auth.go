package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/masuldev/mcl/internal"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"os"
)

var (
	connectCommand = &cobra.Command{
		Use:   "auth",
		Short: "Exec `Issuing SSH Certificate` under AWS with interactive CLI",
		Long:  "Exec `Issuing SSH Certificate` under AWS with interactive CLI",
		Run: func(cmd *cobra.Command, args []string) {
			home, _ := homedir.Dir()
			privateKeyPath := fmt.Sprintf("%s/%s", home, ".ssh/id_rsa")
			publicKeyPath := fmt.Sprintf("%s/%s", home, ".ssh/id_rsa.pub")

			checkExistKeyFile(privateKeyPath, publicKeyPath)
			publicKeyByte, err := openPublicKeyFile(publicKeyPath)
			if err != nil {
				internal.RealPanic(internal.WrapError(err))
			}

			time, err := internal.AskTime()
			if err != nil {
				internal.RealPanic(internal.WrapError(err))
			}

			certificateByte, err := internal.GetCertificate(publicKeyByte, time.Name)
			if err != nil {
				internal.RealPanic(internal.WrapError(err))
			}

			err = writeCertificateFile(home, certificateByte)
			if err != nil {
				internal.RealPanic(internal.WrapError(err))
			}
		},
	}
)

func writeCertificateFile(home string, certificate []byte) error {
	err := os.WriteFile(fmt.Sprintf("%s/%s", home, ".ssh/id_rsa-cert.pub"), certificate, 0600)
	if err != nil {
		return err
	}
	return nil
}

func openPublicKeyFile(publicKeyPath string) ([]byte, error) {
	openFile, err := os.Open(publicKeyPath)
	if err != nil {
		return nil, err
	}

	var publicKeyByte bytes.Buffer
	_, err = publicKeyByte.ReadFrom(openFile)
	if err != nil {
		return nil, err
	}

	return publicKeyByte.Bytes(), nil
}

func checkExistKeyFile(privateKeyPath, publicKeyPath string) {
	existPrivateKey := false
	existPublicKey := false

	_, err := os.Stat(privateKeyPath)
	if err == nil && !errors.Is(err, os.ErrNotExist) {
		existPrivateKey = true
	}

	_, err = os.Stat(publicKeyPath)
	if err == nil && !errors.Is(err, os.ErrNotExist) {
		existPublicKey = true
	}

	if existPrivateKey == false && existPublicKey == false {
		internal.CreateNewCAKey()
	} else if existPrivateKey == false && existPublicKey == true {
		_ = os.Remove(publicKeyPath)
		internal.CreateNewCAKey()
	} else if existPrivateKey == true && existPublicKey == false {
		_ = os.Remove(privateKeyPath)
		internal.CreateNewCAKey()
	}
}

func init() {
	rootCmd.AddCommand(connectCommand)
}
