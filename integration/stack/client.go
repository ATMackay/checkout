package stack

import (
	"testing"

	"github.com/ATMackay/checkout/client"
)

func MakeAuthClient(t *testing.T, baseURL, password string) *client.Client {
	cl, err := client.New(baseURL)
	if err != nil {
		t.Fatal(err)
	}
	cl.AddAuthorizationHeader(password)
	return cl
}
