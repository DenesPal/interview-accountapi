// Copyleft 2020

package interview_accountapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"testing"
	"time"
)

const TestApiBase = "http://localhost:8080/"

type TestContext struct {
	Client   *ApiClient
	ApiBase  string
	PageSize uint
	T        *testing.T
}

// NewTestContext returns a TestContext with initialised ApiClient and testing.T
//
// The purpose of TestContext is to collect repetitive test actions as utility methods.
func NewTestContext(t *testing.T) *TestContext {
	t.Log("NewTestContext()")
	test := TestContext{ApiBase: TestApiBase, PageSize: 5, T: t}

	test.Client = NewApiClient()

	if err := test.Client.SetBaseURL(TestApiBase); err != nil {
		test.T.Fatalf("Failed to create API client: %s", err)
	}

	rand.Seed(time.Now().UnixNano())
	test.T.Log("Random seed initialised")

	return &test
}

// alpha2 returns a random string of two capital latin letters.
//
// The random number generator shall be initialised beforehand by rand.Seed() to obtain a pseudo-random result.
func alpha2() string {
	return fmt.Sprintf("%c%c",
		'A'+rand.Intn(26),
		'A'+rand.Intn(26))
}

// uuid4s returns a random uuid4 string.
//
// The random number generator shall be initialised beforehand by rand.Seed() to obtain a pseudo-random result.
func uuid4s() string {
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		rand.Uint32(),
		rand.Int()&0xffff,
		rand.Int()&0x0fff|0x4000,
		rand.Int()&0x3fff|0x8000,
		rand.Uint64()&0xffffffffffff,
	)
}

// printJson prints a thing JSON formatted
func printJson(thing interface{}) error {
	jsonData, err := json.Marshal(thing)
	if err != nil {
		return errors.New(fmt.Sprintf("json.Marshal() failed: %s", err))
	}
	fmt.Println(string(jsonData))
	return nil
}
