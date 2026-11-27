package connector

import (
	"os"
	"strings"
	"testing"
)

// docs/CONNECTOR.md is copied into apps' repositories: the contract files
// it carries must stay the same as the real ones.
func TestDocCarriesContracts(t *testing.T) {
	doc, err := os.ReadFile("../../docs/CONNECTOR.md")
	if err != nil {
		t.Fatal(err)
	}
	aidl := "../../app/android/app/src/main/aidl/dev/selfhold/shard/connector/"
	for _, f := range []string{"../../proto/connector.proto", aidl + "IShardConnector.aidl", aidl + "IConnectorWatcher.aidl"} {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(doc), strings.TrimRight(string(b), "\n")+"\n```") {
			t.Errorf("docs/CONNECTOR.md does not carry the current %s: copy it into the appendix", f)
		}
	}
}
