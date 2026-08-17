package cli

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"meshturn/internal/batch"
	"meshturn/internal/ledger"
)

const defaultLedgerPath = "meshturn.json"

func Run(args []string, stdout, stderr io.Writer) error {
	ledgerPath, commandArgs, err := parseGlobal(args)
	if err != nil {
		return err
	}
	if len(commandArgs) == 0 {
		return writeJSON(stdout, map[string]any{
			"name":     "meshturn",
			"ledger":   ledgerPath,
			"commands": []string{"open", "record", "close", "show", "attention", "smoke"},
		})
	}
	command := commandArgs[0]
	rest := commandArgs[1:]
	switch command {
	case "open":
		return openBatch(ledgerPath, rest, stdout)
	case "record":
		return recordDisposition(ledgerPath, rest, stdout)
	case "close":
		return closeBatch(ledgerPath, rest, stdout)
	case "show":
		return showBatch(ledgerPath, rest, stdout)
	case "attention":
		return showAttention(ledgerPath, rest, stdout)
	case "smoke":
		return smoke(rest, stdout)
	default:
		return fmt.Errorf("unknown command %q", command)
	}
}

func parseGlobal(args []string) (string, []string, error) {
	path := defaultLedgerPath
	for len(args) > 0 {
		switch {
		case args[0] == "--ledger":
			if len(args) < 2 || strings.TrimSpace(args[1]) == "" {
				return "", nil, errors.New("--ledger requires a path")
			}
			path = args[1]
			args = args[2:]
		case strings.HasPrefix(args[0], "--ledger="):
			path = strings.TrimPrefix(args[0], "--ledger=")
			if strings.TrimSpace(path) == "" {
				return "", nil, errors.New("--ledger requires a path")
			}
			args = args[1:]
		default:
			return path, args, nil
		}
	}
	return path, args, nil
}

func openBatch(path string, args []string, stdout io.Writer) error {
	flags := commandFlags("open")
	id := flags.String("id", "", "batch identifier")
	inkFamily := flags.String("ink-family", "", "ink family")
	screensValue := flags.String("screens", "", "comma-separated ordered screen identifiers")
	noteValue := flags.String("note", "", "optional batch note")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("open does not accept positional arguments")
	}
	screenIDs, err := parseScreens(*screensValue)
	if err != nil {
		return err
	}
	var note *string
	if wasSet(flags, "note") {
		note = noteValue
	}
	b, err := batch.New(*id, *inkFamily, screenIDs, note)
	if err != nil {
		return err
	}
	result, err := ledger.Load(path)
	if err != nil {
		return err
	}
	if err := result.Add(b); err != nil {
		return err
	}
	if err := result.Save(path); err != nil {
		return err
	}
	return writeJSON(stdout, b)
}

func recordDisposition(path string, args []string, stdout io.Writer) error {
	flags := commandFlags("record")
	id := flags.String("id", "", "batch identifier")
	screenID := flags.String("screen", "", "screen identifier")
	outcomeValue := flags.String("outcome", "", "clean, rework, or retire")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("record does not accept positional arguments")
	}
	outcome, err := batch.ParseDisposition(*outcomeValue)
	if err != nil {
		return err
	}
	result, err := ledger.Load(path)
	if err != nil {
		return err
	}
	b, found := result.Find(*id)
	if !found {
		return fmt.Errorf("batch %q not found", *id)
	}
	if err := b.Record(*screenID, outcome); err != nil {
		return err
	}
	if err := result.Save(path); err != nil {
		return err
	}
	return writeJSON(stdout, *b)
}

func closeBatch(path string, args []string, stdout io.Writer) error {
	flags := commandFlags("close")
	id := flags.String("id", "", "batch identifier")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("close does not accept positional arguments")
	}
	result, err := ledger.Load(path)
	if err != nil {
		return err
	}
	b, found := result.Find(*id)
	if !found {
		return fmt.Errorf("batch %q not found", *id)
	}
	if err := b.Close(); err != nil {
		return err
	}
	if err := result.Save(path); err != nil {
		return err
	}
	return writeJSON(stdout, *b)
}

func showBatch(path string, args []string, stdout io.Writer) error {
	flags := commandFlags("show")
	id := flags.String("id", "", "batch identifier")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("show does not accept positional arguments")
	}
	result, err := ledger.Load(path)
	if err != nil {
		return err
	}
	b, found := result.Find(*id)
	if !found {
		return fmt.Errorf("batch %q not found", *id)
	}
	return writeJSON(stdout, *b)
}

func showAttention(path string, args []string, stdout io.Writer) error {
	flags := commandFlags("attention")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("attention does not accept positional arguments")
	}
	result, err := ledger.Load(path)
	if err != nil {
		return err
	}
	return writeJSON(stdout, batch.Inspect(result.Batches))
}

func smoke(args []string, stdout io.Writer) error {
	flags := commandFlags("smoke")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("smoke does not accept positional arguments")
	}
	directory, err := os.MkdirTemp("", "meshturn-smoke-")
	if err != nil {
		return fmt.Errorf("create smoke directory: %w", err)
	}
	defer os.RemoveAll(directory)
	path := filepath.Join(directory, "ledger.json")

	note := ""
	result := ledger.New()
	b, err := batch.New("smoke-batch", "water-based", []string{"screen-01", "screen-02", "screen-03"}, &note)
	if err != nil {
		return err
	}
	if err := result.Add(b); err != nil {
		return err
	}
	if err := result.Save(path); err != nil {
		return err
	}
	for _, update := range []struct {
		screen      string
		disposition batch.Disposition
	}{{"screen-01", batch.Clean}, {"screen-02", batch.Rework}, {"screen-03", batch.Clean}} {
		result, err = ledger.Load(path)
		if err != nil {
			return err
		}
		stored, found := result.Find("smoke-batch")
		if !found {
			return errors.New("smoke batch disappeared")
		}
		if err := stored.Record(update.screen, update.disposition); err != nil {
			return err
		}
		if err := result.Save(path); err != nil {
			return err
		}
	}
	result, err = ledger.Load(path)
	if err != nil {
		return err
	}
	stored, found := result.Find("smoke-batch")
	if !found {
		return errors.New("smoke batch disappeared before close")
	}
	if err := stored.Close(); err != nil {
		return err
	}
	if err := result.Save(path); err != nil {
		return err
	}
	result, err = ledger.Load(path)
	if err != nil {
		return err
	}
	stored, found = result.Find("smoke-batch")
	if !found || stored.State != batch.Closed || stored.Report == nil || stored.Report.Result != batch.Attention {
		return errors.New("smoke workflow did not produce the expected closed report")
	}
	return writeJSON(stdout, map[string]any{
		"ok":    true,
		"batch": *stored,
	})
}

func commandFlags(name string) *flag.FlagSet {
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	return flags
}

func parseScreens(value string) ([]string, error) {
	if strings.TrimSpace(value) == "" {
		return nil, errors.New("--screens requires a comma-separated list")
	}
	parts := strings.Split(value, ",")
	result := make([]string, len(parts))
	for i, part := range parts {
		result[i] = strings.TrimSpace(part)
		if result[i] == "" {
			return nil, errors.New("--screens cannot contain an empty screen identifier")
		}
	}
	return result, nil
}

func wasSet(flags *flag.FlagSet, name string) bool {
	set := false
	flags.Visit(func(flag *flag.Flag) {
		if flag.Name == name {
			set = true
		}
	})
	return set
}

func writeJSON(writer io.Writer, value any) error {
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(value)
}
