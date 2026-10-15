package config

import "errors"

type DiagnosticSeverity string

const (
	DiagnosticError   DiagnosticSeverity = "error"
	DiagnosticWarning DiagnosticSeverity = "warning"
)

type Diagnostic struct {
	Severity DiagnosticSeverity `json:"severity"`
	Code     string             `json:"code"`
	Field    string             `json:"field,omitempty"`
	Message  string             `json:"message"`
	Hint     string             `json:"hint,omitempty"`
}

type ValidationReport struct {
	Diagnostics []Diagnostic `json:"diagnostics"`
}

func (report *ValidationReport) AddError(code, field, message, hint string) {
	report.Diagnostics = append(report.Diagnostics, Diagnostic{
		Severity: DiagnosticError,
		Code:     code,
		Field:    field,
		Message:  message,
		Hint:     hint,
	})
}

func (report *ValidationReport) AddWarning(code, field, message, hint string) {
	report.Diagnostics = append(report.Diagnostics, Diagnostic{
		Severity: DiagnosticWarning,
		Code:     code,
		Field:    field,
		Message:  message,
		Hint:     hint,
	})
}

func (report ValidationReport) HasErrors() bool {
	for _, diagnostic := range report.Diagnostics {
		if diagnostic.Severity == DiagnosticError {
			return true
		}
	}
	return false
}

func (report ValidationReport) Count(severity DiagnosticSeverity) int {
	count := 0
	for _, diagnostic := range report.Diagnostics {
		if diagnostic.Severity == severity {
			count++
		}
	}
	return count
}

func (report ValidationReport) Error() error {
	for _, diagnostic := range report.Diagnostics {
		if diagnostic.Severity == DiagnosticError {
			return errors.New(diagnostic.Message)
		}
	}
	return nil
}
