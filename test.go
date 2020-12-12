package interview_accountapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"testing"
)

const TestApiBase = "http://localhost:8080/"

type TestContext struct {
	Client   *ApiClient
	ApiBase  string
	PageSize uint
	T        *testing.T
}

func NewTestContext(t *testing.T) *TestContext {
	var err error

	t.Log("NewTestContext()")
	test := TestContext{ApiBase: TestApiBase, PageSize: 5, T: t}

	test.Client, err = NewClient(ApiClient{})
	if err == nil {
		err = test.Client.SetBaseURL(TestApiBase)
	}
	if err != nil {
		test.T.Log("Failed to create API client,", err)
		test.T.Fail()
		return nil
	}

	return &test
}

// returns string of two capital latin letters //
func alpha2() string {
	return fmt.Sprintf("%c%c",
		'A'+rand.Intn(26),
		'A'+rand.Intn(26))
}

// returns random uuid4 (don't forget to seed the random generator) //
func uuid4s() string {
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		rand.Uint32(),
		rand.Int()&0xffff,
		rand.Int()&0x0fff|0x4000,
		rand.Int()&0x3fff|0x8000,
		rand.Uint64()&0xffffffffffff,
	)
}

func printJson(account *Account) error {
	jsonData, err := json.Marshal(*account)
	if err != nil {
		return errors.New(fmt.Sprintf("json.Marshal() failed: %s", err))
	}
	fmt.Println(string(jsonData))
	return nil
}
