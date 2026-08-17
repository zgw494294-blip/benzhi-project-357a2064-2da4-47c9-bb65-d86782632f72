package batch

import (
	"fmt"
	"sort"
	"strings"
)

type State string

const (
	Active State = "active"
	Closed State = "closed"
)

type Disposition string

const (
	Clean  Disposition = "clean"
	Rework Disposition = "rework"
	Retire Disposition = "retire"
)

type Result string

const (
	AllReusable Result = "all-reusable"
	Attention   Result = "attention"
)

type Screen struct {
	ID      string       `json:"id"`
	Outcome *Disposition `json:"outcome,omitempty"`
}

type Report struct {
	Clean  int    `json:"clean"`
	Rework int    `json:"rework"`
	Retire int    `json:"retire"`
	Result Result `json:"result"`
}

type Batch struct {
	ID        string   `json:"id"`
	InkFamily string   `json:"ink_family"`
	Note      *string  `json:"note,omitempty"`
	State     State    `json:"state"`
	Screens   []Screen `json:"screens"`
	Report    *Report  `json:"report,omitempty"`
}

type Inspection struct {
	Active                   []Batch `json:"active"`
	ClosedRequiringAttention []Batch `json:"closed_requiring_attention"`
}

func New(id, inkFamily string, screenIDs []string, note *string) (Batch, error) {
	b := Batch{
		ID:        id,
		InkFamily: inkFamily,
		Note:      note,
		State:     Active,
		Screens:   make([]Screen, len(screenIDs)),
	}
	for i, id := range screenIDs {
		b.Screens[i] = Screen{ID: id}
	}
	if err := b.Validate(); err != nil {
		return Batch{}, err
	}
	return b, nil
}

func (b Batch) Validate() error {
	if strings.TrimSpace(b.ID) == "" {
		return fmt.Errorf("batch id is required")
	}
	if strings.TrimSpace(b.InkFamily) == "" {
		return fmt.Errorf("ink family is required")
	}
	if b.State != Active && b.State != Closed {
		return fmt.Errorf("batch %q has invalid state %q", b.ID, b.State)
	}
	if len(b.Screens) == 0 {
		return fmt.Errorf("batch %q needs at least one screen", b.ID)
	}

	seen := make(map[string]struct{}, len(b.Screens))
	counts := Report{}
	complete := true
	for i, screen := range b.Screens {
		if strings.TrimSpace(screen.ID) == "" {
			return fmt.Errorf("batch %q screen %d has an empty id", b.ID, i)
		}
		if _, exists := seen[screen.ID]; exists {
			return fmt.Errorf("batch %q repeats screen %q", b.ID, screen.ID)
		}
		seen[screen.ID] = struct{}{}
		if screen.Outcome == nil {
			complete = false
			continue
		}
		switch *screen.Outcome {
		case Clean:
			counts.Clean++
		case Rework:
			counts.Rework++
		case Retire:
			counts.Retire++
		default:
			return fmt.Errorf("batch %q screen %q has invalid disposition %q", b.ID, screen.ID, *screen.Outcome)
		}
	}

	if b.State == Active {
		if b.Report != nil {
			return fmt.Errorf("active batch %q has a closure report", b.ID)
		}
		return nil
	}
	if !complete {
		return fmt.Errorf("closed batch %q is incomplete", b.ID)
	}
	if b.Report == nil {
		return fmt.Errorf("closed batch %q has no closure report", b.ID)
	}
	if b.Report.Clean != counts.Clean || b.Report.Rework != counts.Rework || b.Report.Retire != counts.Retire {
		return fmt.Errorf("closed batch %q has incorrect disposition counts", b.ID)
	}
	expectedResult := AllReusable
	if counts.Rework > 0 || counts.Retire > 0 {
		expectedResult = Attention
	}
	if b.Report.Result != expectedResult {
		return fmt.Errorf("closed batch %q has incorrect result", b.ID)
	}
	return nil
}

func (b *Batch) Record(screenID string, disposition Disposition) error {
	if err := validDisposition(disposition); err != nil {
		return err
	}
	if err := b.Validate(); err != nil {
		return err
	}
	if b.State != Active {
		return fmt.Errorf("batch %q is already closed", b.ID)
	}
	for i := range b.Screens {
		if b.Screens[i].ID != screenID {
			continue
		}
		if b.Screens[i].Outcome != nil {
			return fmt.Errorf("screen %q already has a disposition", screenID)
		}
		value := disposition
		b.Screens[i].Outcome = &value
		return nil
	}
	return fmt.Errorf("screen %q is not in batch %q", screenID, b.ID)
}

func (b *Batch) Close() error {
	if err := b.Validate(); err != nil {
		return err
	}
	if b.State != Active {
		return fmt.Errorf("batch %q is already closed", b.ID)
	}

	report := Report{}
	for _, screen := range b.Screens {
		if screen.Outcome == nil {
			return fmt.Errorf("batch %q still has screens without dispositions", b.ID)
		}
		switch *screen.Outcome {
		case Clean:
			report.Clean++
		case Rework:
			report.Rework++
		case Retire:
			report.Retire++
		}
	}
	report.Result = AllReusable
	if report.Rework > 0 || report.Retire > 0 {
		report.Result = Attention
	}
	b.State = Closed
	b.Report = &report
	return nil
}

func Inspect(batches []Batch) Inspection {
	ordered := append([]Batch(nil), batches...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].ID < ordered[j].ID })
	result := Inspection{
		Active:                   make([]Batch, 0),
		ClosedRequiringAttention: make([]Batch, 0),
	}
	for _, b := range ordered {
		switch {
		case b.State == Active:
			result.Active = append(result.Active, b)
		case b.State == Closed && b.Report != nil && b.Report.Result == Attention:
			result.ClosedRequiringAttention = append(result.ClosedRequiringAttention, b)
		}
	}
	return result
}

func ParseDisposition(value string) (Disposition, error) {
	disposition := Disposition(strings.ToLower(strings.TrimSpace(value)))
	if err := validDisposition(disposition); err != nil {
		return "", err
	}
	return disposition, nil
}

func validDisposition(disposition Disposition) error {
	switch disposition {
	case Clean, Rework, Retire:
		return nil
	default:
		return fmt.Errorf("invalid disposition %q; use clean, rework, or retire", disposition)
	}
}
