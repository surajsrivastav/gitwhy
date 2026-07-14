package provenance

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"
)

const SchemaVersion = "gitwhy/v1"

type TargetType string

const (
	TargetCommit TargetType = "commit"
	TargetPR     TargetType = "pr"
)

type AttributionType string

const (
	AttributionHuman    AttributionType = "human"
	AttributionCopilot  AttributionType = "copilot"
	AttributionPair     AttributionType = "pair"
	AttributionUnknown  AttributionType = "unknown"
)

func AgentAttribution(name string) AttributionType {
	return AttributionType(fmt.Sprintf("agent:%s", name))
}

type OriginType string

const (
	OriginHuman    OriginType = "human"
	OriginSpec     OriginType = "spec"
	OriginPrompt   OriginType = "prompt"
	OriginTemplate OriginType = "template"
	OriginUpstream OriginType = "upstream"
	OriginAI       OriginType = "ai"
	OriginUnknown  OriginType = "unknown"
)

type Target struct {
	Type TargetType `json:"type" yaml:"type"`
	Ref  string     `json:"ref" yaml:"ref"`
}

type Attribution struct {
	By        AttributionType `json:"by" yaml:"by"`
	Timestamp string          `json:"timestamp" yaml:"timestamp"`
}

type Intent struct {
	Summary  string     `json:"summary" yaml:"summary"`
	Spec     string     `json:"spec,omitempty" yaml:"spec,omitempty"`
	SpecHash string     `json:"spec_hash,omitempty" yaml:"spec_hash,omitempty"`
	Origin   OriginType `json:"origin,omitempty" yaml:"origin,omitempty"`
}

type Context struct {
	Ticket string `json:"ticket,omitempty" yaml:"ticket,omitempty"`
	Prompt string `json:"prompt,omitempty" yaml:"prompt,omitempty"`
	Model  string `json:"model,omitempty" yaml:"model,omitempty"`
	Branch string `json:"branch,omitempty" yaml:"branch,omitempty"`
}

type Record struct {
	Schema      string      `json:"schema" yaml:"schema"`
	Target      Target      `json:"target" yaml:"target"`
	Attribution Attribution `json:"attribution" yaml:"attribution"`
	Intent      Intent      `json:"intent" yaml:"intent"`
	Context     Context     `json:"context,omitempty" yaml:"context,omitempty"`
	// RecordHash is a SHA-256 digest of the record's content (excluding this
	// field), computed at Seal() time. It doesn't prevent tampering — anyone
	// who can rewrite the note can also recompute the hash — but it detects
	// tampering by anything that edits the stored JSON without going through
	// gitwhy, e.g. a manual `git notes edit`.
	RecordHash string `json:"record_hash,omitempty" yaml:"record_hash,omitempty"`
}

func NewRecord(targetType TargetType, ref string) *Record {
	return &Record{
		Schema: SchemaVersion,
		Target: Target{
			Type: targetType,
			Ref:  ref,
		},
		Attribution: Attribution{
			By:        AttributionUnknown,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		},
	}
}

func (r *Record) SetAttribution(by AttributionType) {
	r.Attribution.By = by
	r.Attribution.Timestamp = time.Now().UTC().Format(time.RFC3339)
}

func (r *Record) SetIntent(summary string, origin OriginType, spec, specHash string) {
	r.Intent = Intent{
		Summary:  summary,
		Spec:     spec,
		SpecHash: specHash,
		Origin:   origin,
	}
}

func (r *Record) SetContext(ticket, prompt, model string) {
	r.Context = Context{
		Ticket: ticket,
		Prompt: prompt,
		Model:  model,
	}
}

func (r *Record) SetGitContext(branch string) {
	r.Context.Branch = branch
}

func (r *Record) Validate() error {
	if r.Schema != SchemaVersion {
		return fmt.Errorf("unsupported schema version: %s", r.Schema)
	}
	if r.Target.Type != TargetCommit && r.Target.Type != TargetPR {
		return fmt.Errorf("invalid target type: %s", r.Target.Type)
	}
	if r.Attribution.By == "" {
		return fmt.Errorf("attribution is required")
	}
	if r.Target.Ref == "" {
		return fmt.Errorf("target ref is required")
	}
	return nil
}

// HashStatus describes the outcome of verifying a record's RecordHash.
type HashStatus int

const (
	HashMissing HashStatus = iota
	HashValid
	HashTampered
)

func (s HashStatus) String() string {
	switch s {
	case HashValid:
		return "valid"
	case HashTampered:
		return "tampered"
	default:
		return "missing"
	}
}

// ComputeHash returns the SHA-256 digest (hex-encoded) of the record's
// content, excluding RecordHash itself.
func (r *Record) ComputeHash() string {
	clone := *r
	clone.RecordHash = ""
	data, _ := json.Marshal(clone)
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h)
}

// Seal stamps RecordHash with the record's current content hash. Call this
// right before storing the record.
func (r *Record) Seal() {
	r.RecordHash = r.ComputeHash()
}

// VerifyHash reports whether the stored RecordHash matches the record's
// current content.
func (r *Record) VerifyHash() HashStatus {
	if r.RecordHash == "" {
		return HashMissing
	}
	if r.ComputeHash() == r.RecordHash {
		return HashValid
	}
	return HashTampered
}

func (r *Record) Marshal() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}

func Unmarshal(data []byte) (*Record, error) {
	var r Record
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("unmarshal provenance record: %w", err)
	}
	return &r, nil
}

func IsAgentAttribution(by AttributionType) bool {
	if len(by) > 6 && string(by[:6]) == "agent:" {
		return true
	}
	return false
}
