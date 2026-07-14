package provenance

import "testing"

func TestVerifyHashMissing(t *testing.T) {
	r := NewRecord(TargetCommit, "abc123")
	if status := r.VerifyHash(); status != HashMissing {
		t.Errorf("expected HashMissing, got %v", status)
	}
}

func TestSealAndVerifyHash(t *testing.T) {
	r := NewRecord(TargetCommit, "abc123")
	r.SetAttribution(AttributionHuman)
	r.SetIntent("fix bug", OriginSpec, "SPEC-1", "hash1")
	r.SetContext("JIRA-1", "", "claude-sonnet-4-6")

	r.Seal()
	if r.RecordHash == "" {
		t.Fatal("expected RecordHash to be set after Seal")
	}
	if status := r.VerifyHash(); status != HashValid {
		t.Errorf("expected HashValid, got %v", status)
	}
}

func TestVerifyHashTamperedAfterEdit(t *testing.T) {
	r := NewRecord(TargetCommit, "abc123")
	r.SetAttribution(AttributionHuman)
	r.Seal()

	// Simulate someone editing the note after it was sealed.
	r.Attribution.By = AttributionCopilot

	if status := r.VerifyHash(); status != HashTampered {
		t.Errorf("expected HashTampered, got %v", status)
	}
}

func TestComputeHashDeterministic(t *testing.T) {
	r1 := NewRecord(TargetCommit, "abc123")
	r1.SetAttribution(AttributionHuman)
	r1.Attribution.Timestamp = "2026-01-01T00:00:00Z"

	r2 := NewRecord(TargetCommit, "abc123")
	r2.SetAttribution(AttributionHuman)
	r2.Attribution.Timestamp = "2026-01-01T00:00:00Z"

	if r1.ComputeHash() != r2.ComputeHash() {
		t.Error("expected identical records to produce the same hash")
	}
}

func TestComputeHashExcludesRecordHashField(t *testing.T) {
	r := NewRecord(TargetCommit, "abc123")
	r.SetAttribution(AttributionHuman)

	h1 := r.ComputeHash()
	r.RecordHash = "garbage"
	h2 := r.ComputeHash()

	if h1 != h2 {
		t.Error("expected ComputeHash to ignore the current RecordHash value")
	}
}

func TestHashStatusString(t *testing.T) {
	cases := map[HashStatus]string{
		HashMissing:  "missing",
		HashValid:    "valid",
		HashTampered: "tampered",
	}
	for status, want := range cases {
		if got := status.String(); got != want {
			t.Errorf("HashStatus(%d).String() = %q, want %q", status, got, want)
		}
	}
}

func TestUnmarshalPreservesRecordHash(t *testing.T) {
	r := NewRecord(TargetCommit, "abc123")
	r.SetAttribution(AttributionHuman)
	r.Seal()

	data, err := r.Marshal()
	if err != nil {
		t.Fatal(err)
	}

	unmarshaled, err := Unmarshal(data)
	if err != nil {
		t.Fatal(err)
	}
	if unmarshaled.RecordHash != r.RecordHash {
		t.Errorf("expected RecordHash to round-trip, got %q vs %q", unmarshaled.RecordHash, r.RecordHash)
	}
	if status := unmarshaled.VerifyHash(); status != HashValid {
		t.Errorf("expected round-tripped record to verify as valid, got %v", status)
	}
}
