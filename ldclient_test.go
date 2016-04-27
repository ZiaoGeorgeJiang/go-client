package ldclient

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"testing"
	"time"
)

const (
	TestDataPath = "./testdata/test_data.json"
)

var (
	config = Config{
		BaseUri:       "https://localhost:3000",
		Capacity:      1000,
		FlushInterval: 5 * time.Second,
		Logger:        log.New(os.Stderr, "[LaunchDarkly]", log.LstdFlags),
		Timeout:       1500 * time.Millisecond,
		Stream:        true,
		Offline:       true,
	}
	client *LDClient
)

func TestOfflineModeAlwaysReturnsDefaultValue(t *testing.T) {
	client, _ := MakeCustomClient("api_key", config, 0)
	var key = "foo"
	res, err := client.Toggle("anything", User{Key: &key}, true)

	if err != nil {
		t.Errorf("Unexpected error in Toggle: %+v", err)
	}

	if !res {
		t.Errorf("Offline mode should return default value, but doesn't")
	}
}

type evaluateTestData struct {
	Name                 string                 `json:"name"`
	FeatureKey           string                 `json:"featureKey"`
	DefaultValue         string                 `json:"defaultValue"`
	UsersAndExpectations []usersAndExpectations `json:"usersAndExpectations"`
	FeatureFlag          FeatureFlag            `json:"featureFlag"`
}

type usersAndExpectations struct {
	ExpectedValue string `json:"expectedValue"`
	ExpectError   bool   `json:"expectError"`
	User          User   `json:"user"`
}

func TestEvaluate(t *testing.T) {
	var container []evaluateTestData
	t.Logf("Loading test data from %v", TestDataPath)

	file, err := ioutil.ReadFile(TestDataPath)
	if err != nil {
		t.Fatalf("FATAL: Error loading test_data.json file: %v", err)
	}

	err = json.Unmarshal(file, &container)
	if err != nil {
		t.Fatalf("FATAL: Error unmarshalling test_data.json file: %v", err)
	}

	count := len(container)
	if count == 0 {
		t.Fatalf("FATAL: Found zero Feature Flags to evaluate")
	}
	t.Logf("Found %d Feature Flags to evaluate:", count)

	for i, td := range container {
		pre := fmt.Sprintf("(%d/%d) ", i+1, count)
		t.Log("")
		t.Logf("%sEvaluating Feature Flag: %s", pre, td.Name)

		userCount := len(td.UsersAndExpectations)
		if userCount == 0 {
			t.Errorf("%s\tERROR: Found zero users for evaluation")
			continue
		}
		t.Logf("%s\tFound %d users to evaluate", pre, userCount)

		client, err = MakeCustomClient("api_key", config, 0)
		if err != nil {
			t.Fatalf("%s\tFATAL: Error creating client: %v", pre, err)
		}

		err = client.store.Upsert(td.FeatureFlag.Key, td.FeatureFlag)
		if err != nil {
			t.Fatalf("%s\tFATAL: Error upserting Feature Flag: %v", pre, err)
		}

		for _, ue := range td.UsersAndExpectations {
			userOk := true
			result, err := client.evaluate(td.FeatureKey, ue.User, td.DefaultValue)
			if err != nil {
				if !ue.ExpectError {
					userOk = false
					t.Errorf("%s\tERROR: Unexpected error: %+v", pre, err)
				}
			} else {
				if ue.ExpectError {
					userOk = false
					t.Errorf("%s\tERROR: Didn't get expected error", pre)
				}
			}
			if result != ue.ExpectedValue {
				userOk = false
				t.Errorf("%s\tERROR: Expected value: %+v. Instead got: %+v", pre, ue.ExpectedValue, result)
			}
			if !userOk {
				user, _ := json.Marshal(ue.User)
				t.Errorf("%s\t\tWhen evaluating user: %s", pre, string(user))
			}
		}
	}
}
