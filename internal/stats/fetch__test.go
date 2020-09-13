package stats

import (
	"fmt"
	"testing"
	"time"
)

func TestFetch(t *testing.T) {

	id := 939
	domain := "osrs"

	res, err := Fetch(time.Now(), domain, id)
	if err != nil || res == nil {
		t.Errorf("[FAIL] TestFetch: %s\n", err)
	}

	fmt.Printf("result: %v+\n", res)
}
