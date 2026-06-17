package provenance

import (
	"testing"
	"time"
)

func TestNewRecord(t *testing.T) {
	ref := "abc123"
	record := NewRecord(TargetCommit, ref)

	if record.Schema != SchemaVersion {
		t.Errorf("expected schema %q, got %q", SchemaVersion, record.Schema)
	}
	if record.Target.Type != TargetCommit {
		t.Errorf("expected target type %q, got %q", TargetCommit, record.Target.Type)
	}
	if record.Target.Ref != ref {
		t.Errorf("expected ref %q, got %q", ref, record.Target.Ref)
	}
	if record.Attribution.By != AttributionHuman {
		t.Errorf("expected default attribution %q, got %q", AttributionHuman, record.Attribution.By)
	}
	if record.Attribution.Timestamp == "" {
		t.Error("expected timestamp to be set")
	}
}

func TestNewRecordPR(t *testing.T) {
	record := NewRecord(TargetPR, "42")
	if record.Target.Type != TargetPR {
		t.Errorf("expected target type PR, got %q", record.Target.Type)
	}
	if record.Target.Ref != "42" {
		t.Errorf("expected ref 42, got %q", record.Target.Ref)
	}
}

func TestSetAttribution(t *testing.T) {
	record := NewRecord(TargetCommit, "abc")
	record.Attribution.Timestamp = ""
	record.SetAttribution(AttributionCopilot)

	if record.Attribution.By != AttributionCopilot {
		t.Errorf("expected %q, got %q", AttributionCopilot, record.Attribution.By)
	}
	if record.Attribution.Timestamp == "" {
		t.Error("expected timestamp to be set after SetAttribution")
	}

	record.SetAttribution(AgentAttribution("blue"))
	if record.Attribution.By != AttributionType("agent:blue") {
		t.Errorf("expected agent:blue, got %q", record.Attribution.By)
	}

	record.SetAttribution(AttributionPair)
	if record.Attribution.By != AttributionPair {
		t.Errorf("expected pair, got %q", record.Attribution.By)
	}
}

func TestSetIntent(t *testing.T) {
	record := NewRecord(TargetCommit, "abc")
	record.SetIntent("fix auth bug", OriginSpec, "BLUE-204", "abc123")

	if record.Intent.Summary != "fix auth bug" {
		t.Errorf("expected summary 'fix auth bug', got %q", record.Intent.Summary)
	}
	if record.Intent.Origin != OriginSpec {
		t.Errorf("expected origin %q, got %q", OriginSpec, record.Intent.Origin)
	}
	if record.Intent.Spec != "BLUE-204" {
		t.Errorf("expected spec BLUE-204, got %q", record.Intent.Spec)
	}
	if record.Intent.SpecHash != "abc123" {
		t.Errorf("expected spec hash abc123, got %q", record.Intent.SpecHash)
	}

	record.SetIntent("", OriginPrompt, "", "")
	if record.Intent.Summary != "" {
		t.Errorf("expected empty summary")
	}
	if record.Intent.Origin != OriginPrompt {
		t.Errorf("expected origin prompt, got %q", record.Intent.Origin)
	}
}

func TestSetContext(t *testing.T) {
	record := NewRecord(TargetCommit, "abc")
	record.SetContext("JIRA-42", "add retry logic with exponential backoff", "claude-4")

	if record.Context.Ticket != "JIRA-42" {
		t.Errorf("expected ticket JIRA-42, got %q", record.Context.Ticket)
	}
	if record.Context.Prompt != "add retry logic with exponential backoff" {
		t.Errorf("unexpected prompt: %q", record.Context.Prompt)
	}
	if record.Context.Model != "claude-4" {
		t.Errorf("expected model claude-4, got %q", record.Context.Model)
	}

	record.SetContext("", "", "")
	if record.Context.Ticket != "" || record.Context.Prompt != "" || record.Context.Model != "" {
		t.Error("expected all context fields to be empty")
	}
}

func TestSetGitContext(t *testing.T) {
	record := NewRecord(TargetCommit, "abc")
	record.SetGitContext("feature/BLUE-123-auth")

	if record.Context.Branch != "feature/BLUE-123-auth" {
		t.Errorf("expected branch, got %q", record.Context.Branch)
	}

	record.SetGitContext("")
	if record.Context.Branch != "" {
		t.Error("expected branch to be cleared")
	}
}

func TestMarshalUnmarshalBranch(t *testing.T) {
	original := NewRecord(TargetCommit, "abc123")
	original.SetContext("JIRA-42", "prompt", "gpt-4")
	original.SetGitContext("feature/TICKET-321-login")

	data, err := original.Marshal()
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	restored, err := Unmarshal(data)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if restored.Context.Branch != "feature/TICKET-321-login" {
		t.Errorf("branch mismatch: %q", restored.Context.Branch)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		record  *Record
		wantErr bool
	}{
		{
			name:    "valid commit record",
			record:  NewRecord(TargetCommit, "abc"),
			wantErr: false,
		},
		{
			name:    "valid PR record",
			record:  NewRecord(TargetPR, "42"),
			wantErr: false,
		},
		{
			name:    "missing ref",
			record:  &Record{Schema: SchemaVersion, Target: Target{Type: TargetCommit}, Attribution: Attribution{By: AttributionHuman}},
			wantErr: true,
		},
		{
			name:    "missing attribution",
			record:  &Record{Schema: SchemaVersion, Target: Target{Type: TargetCommit, Ref: "abc"}},
			wantErr: true,
		},
		{
			name:    "bad schema",
			record:  &Record{Schema: "bad/v2", Target: Target{Type: TargetCommit, Ref: "abc"}, Attribution: Attribution{By: AttributionHuman}},
			wantErr: true,
		},
		{
			name:    "invalid target type",
			record:  &Record{Schema: SchemaVersion, Target: Target{Type: "invalid", Ref: "abc"}, Attribution: Attribution{By: AttributionHuman}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.record.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMarshalUnmarshal(t *testing.T) {
	original := NewRecord(TargetCommit, "abc123")
	original.SetAttribution(AgentAttribution("blue"))
	original.SetIntent("add retry logic", OriginSpec, "BLUE-317", "a1b2c3d")
	original.SetContext("TICKET-1", "implement exponential backoff", "gpt-4")

	data, err := original.Marshal()
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	restored, err := Unmarshal(data)
	if err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if restored.Schema != original.Schema {
		t.Errorf("schema mismatch: %q vs %q", restored.Schema, original.Schema)
	}
	if restored.Target.Ref != original.Target.Ref {
		t.Errorf("ref mismatch: %q vs %q", restored.Target.Ref, original.Target.Ref)
	}
	if restored.Attribution.By != original.Attribution.By {
		t.Errorf("attribution mismatch: %q vs %q", restored.Attribution.By, original.Attribution.By)
	}
	if restored.Intent.Summary != original.Intent.Summary {
		t.Errorf("intent mismatch: %q vs %q", restored.Intent.Summary, original.Intent.Summary)
	}
	if restored.Context.Ticket != original.Context.Ticket {
		t.Errorf("ticket mismatch: %q vs %q", restored.Context.Ticket, original.Context.Ticket)
	}
}

func TestIsAgentAttribution(t *testing.T) {
	tests := []struct {
		by     AttributionType
		isAgen bool
	}{
		{AttributionHuman, false},
		{AttributionCopilot, false},
		{AttributionPair, false},
		{AttributionType(""), false},
		{AttributionType("x"), false},
		{AttributionType("agent"), false},
		{AttributionType("agent:"), false},
		{AttributionType("agent:a"), true},
		{AgentAttribution("blue"), true},
		{AttributionType("agent:copilot"), true},
	}

	for _, tt := range tests {
		t.Run(string(tt.by), func(t *testing.T) {
			if got := IsAgentAttribution(tt.by); got != tt.isAgen {
				t.Errorf("IsAgentAttribution(%q) = %v, want %v", tt.by, got, tt.isAgen)
			}
		})
	}
}

func TestTimestampFormat(t *testing.T) {
	record := NewRecord(TargetCommit, "abc")
	_, err := time.Parse(time.RFC3339, record.Attribution.Timestamp)
	if err != nil {
		t.Errorf("timestamp is not RFC3339: %v", err)
	}
}

func TestMarshalRoundTripJSON(t *testing.T) {
	r := NewRecord(TargetPR, "123")
	r.SetAttribution(AgentAttribution("blue"))
	r.SetIntent("implement multi-tenant auth", OriginSpec, "BLUE-204", "def456")
	r.SetContext("", "generate auth middleware", "claude-opus-4")

	data, err := r.Marshal()
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	got, err := Unmarshal(data)
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if got.Intent.Spec != "BLUE-204" {
		t.Errorf("spec mismatch: %q", got.Intent.Spec)
	}
	if got.Attribution.By != "agent:blue" {
		t.Errorf("attribution mismatch: %q", got.Attribution.By)
	}
	if got.Target.Ref != "123" {
		t.Errorf("ref mismatch: %q", got.Target.Ref)
	}
}

func TestEmptyRecordFailsValidate(t *testing.T) {
	r := &Record{}
	if err := r.Validate(); err == nil {
		t.Error("expected validation error for empty record")
	}
}

func TestUnmarshalInvalidJSON(t *testing.T) {
	_, err := Unmarshal([]byte("{invalid json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestAgentAttributionFormat(t *testing.T) {
	agent := AgentAttribution("my-agent")
	if string(agent) != "agent:my-agent" {
		t.Errorf("expected 'agent:my-agent', got %q", agent)
	}
}

func TestMarshalIndent(t *testing.T) {
	r := NewRecord(TargetCommit, "abc123")
	data, err := r.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty marshaled data")
	}
}

func TestOriginTypes(t *testing.T) {
	if OriginHuman != "human" {
		t.Errorf("unexpected OriginHuman: %q", OriginHuman)
	}
	if OriginSpec != "spec" {
		t.Errorf("unexpected OriginSpec: %q", OriginSpec)
	}
	if OriginPrompt != "prompt" {
		t.Errorf("unexpected OriginPrompt: %q", OriginPrompt)
	}
	if OriginTemplate != "template" {
		t.Errorf("unexpected OriginTemplate: %q", OriginTemplate)
	}
	if OriginUpstream != "upstream" {
		t.Errorf("unexpected OriginUpstream: %q", OriginUpstream)
	}
}

func TestTargetTypes(t *testing.T) {
	if TargetCommit != "commit" {
		t.Errorf("unexpected TargetCommit: %q", TargetCommit)
	}
	if TargetPR != "pr" {
		t.Errorf("unexpected TargetPR: %q", TargetPR)
	}
}

func TestAttributionTypes(t *testing.T) {
	if AttributionHuman != "human" {
		t.Errorf("unexpected AttributionHuman: %q", AttributionHuman)
	}
	if AttributionCopilot != "copilot" {
		t.Errorf("unexpected AttributionCopilot: %q", AttributionCopilot)
	}
	if AttributionPair != "pair" {
		t.Errorf("unexpected AttributionPair: %q", AttributionPair)
	}
}
