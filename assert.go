package abide

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"strings"
	"testing"

	"github.com/beme/abide/internal"
	"github.com/sergi/go-diff/diffmatchpatch"
)

// Assertable represents an object that can be asserted.
type Assertable interface {
	String() string
}

// Assert asserts the value of an object with implements Assertable.
func Assert(t *testing.T, id string, a Assertable) {
	data := a.String()
	snapshot := getSnapshot(snapshotID(id))

	var err error
	if snapshot == nil {
		fmt.Printf("Creating snapshot `%s`\n", id)
		snapshot, err = createSnapshot(snapshotID(id), data)
		if err != nil {
			t.Fatal(err)
		}
		snapshot.evaluated = true
		return
	}

	snapshot.evaluated = true
	diff := compareResults(t, snapshot.value, strings.TrimSpace(data))
	if diff != "" {
		if snapshot != nil && args.shouldUpdate {
			fmt.Printf("Updating snapshot `%s`\n", id)
			_, err = createSnapshot(snapshotID(id), data)
			if err != nil {
				t.Fatal(err)
			}
			return
		}

		msg := didNotMatchMessage(diff)

		t.Error(msg)
		return
	}
}

// AssertHTTPResponse asserts the value of an http.Response.
func AssertHTTPResponse(t *testing.T, id string, w *http.Response) {
	config, err := getConfig()
	if err != nil {
		t.Fatal(err)
	}

	snapshot := getSnapshot(snapshotID(id))

	body, err := httputil.DumpResponse(w, true)
	if err != nil {
		t.Fatal(err)
	}

	data := string(body)

	// If the response body is JSON, indent.
	if w.Header.Get("Content-Type") == "application/json" {
		lines := strings.Split(data, "\n")
		jsonStr := lines[len(lines)-1]

		var jsonIface map[string]interface{}
		err = json.Unmarshal([]byte(jsonStr), &jsonIface)
		if err != nil {
			t.Fatal(err)
		}

		// Clean/update json based on config.
		if config != nil {
			for k, v := range config.Defaults {
				jsonIface = internal.UpdateKeyValuesInMap(k, v, jsonIface)
			}
		}

		out, err := json.MarshalIndent(jsonIface, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		lines[len(lines)-1] = string(out)
		data = strings.Join(lines, "\n")
	}

	if snapshot == nil {
		fmt.Printf("Creating snapshot `%s`\n", id)
		snapshot, err = createSnapshot(snapshotID(id), data)
		if err != nil {
			t.Fatal(err)
		}
		snapshot.evaluated = true
		return
	}

	snapshot.evaluated = true
	diff := compareResults(t, snapshot.value, strings.TrimSpace(data))
	if diff != "" {
		if snapshot != nil && args.shouldUpdate {
			fmt.Printf("Updating snapshot `%s`\n", id)
			_, err = createSnapshot(snapshotID(id), data)
			if err != nil {
				t.Fatal(err)
			}
			return
		}

		msg := didNotMatchMessage(diff)

		t.Error(msg)
		return
	}
}

func compareResults(t *testing.T, existing, new string) string {
	dmp := diffmatchpatch.New()
	dmp.PatchMargin = 20
	allDiffs := dmp.DiffMain(existing, new, false)
	nonEqualDiffs := []diffmatchpatch.Diff{}
	for _, diff := range allDiffs {
		if diff.Type != diffmatchpatch.DiffEqual {
			nonEqualDiffs = append(nonEqualDiffs, diff)
		}
	}

	if len(nonEqualDiffs) == 0 {
		return ""
	}

	return dmp.DiffPrettyText(allDiffs)
}

func didNotMatchMessage(diff string) string {
	msg := "\n\nExisting snapshot does not match results...\n\n"
	msg += diff
	msg += "\n\n"
	msg += "If this change was intentional, run tests again, $ go test -v -- -u\n"
	return msg
}
