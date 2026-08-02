package config

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
)

// FormatText 将报告渲染为面向操作者的稳定控制台输出。
func (r Report) FormatText() string {
	if r.ErrorCount() == 0 {
		if len(r.Findings) == 0 {
			return "Configuration valid.\n"
		}
		return formatWarnings(r)
	}

	var output strings.Builder
	output.WriteString("====================================\n Configuration Validation Failed\n====================================\n\n")
	writeFindings(&output, r.Findings, SeverityError)
	if hasSeverity(r.Findings, SeverityWarning) {
		output.WriteString("Warnings:\n\n")
		writeFindings(&output, r.Findings, SeverityWarning)
	}
	output.WriteString("====================================\n\nApplication startup aborted.\n====================================\n")
	return output.String()
}

// FormatJSON 将报告编码为不泄露配置值的 JSON。
func (r Report) FormatJSON() ([]byte, error) {
	reportCopy := r
	reportCopy.Values = nil
	return json.MarshalIndent(reportCopy, "", "  ")
}

// FormatPatch 生成只读的 .env 迁移建议，敏感值始终保留为待填写占位符。
func (r Report) FormatPatch() string {
	var output strings.Builder
	output.WriteString("# Configuration migration suggestions. Review before applying.\n")
	findings := append([]Finding(nil), r.Findings...)
	sort.Slice(findings, func(i, j int) bool { return findings[i].Key < findings[j].Key })
	for _, finding := range findings {
		switch finding.Code {
		case "required", "required_any_of":
			_, _ = fmt.Fprintf(&output, "# Required since %s: %s\n", finding.Introduced, finding.Description)
			if strings.Contains(finding.Key, " or ") {
				_, _ = fmt.Fprintf(&output, "# Set one of: %s\n", finding.Key)
			} else {
				_, _ = fmt.Fprintf(&output, "%s=\n", finding.Key)
			}
		case "deprecated":
			_, _ = fmt.Fprintf(&output, "# Deprecated: remove %s", finding.Key)
			if finding.Replacement != "" {
				_, _ = fmt.Fprintf(&output, "; migrate to %s", finding.Replacement)
			}
			output.WriteByte('\n')
		case "removed":
			_, _ = fmt.Fprintf(&output, "# Removed: delete %s\n", finding.Key)
		case "schema_version":
			_, _ = fmt.Fprintf(&output, "GRAFT_CONFIG_SCHEMA_VERSION=%d\n", r.SchemaVersion)
		}
	}
	return output.String()
}

func formatWarnings(report Report) string {
	var output strings.Builder
	output.WriteString("Configuration valid with warnings.\n\n")
	writeFindings(&output, report.Findings, SeverityWarning)
	return output.String()
}

func writeFindings(output *strings.Builder, findings []Finding, severity Severity) {
	filtered := make([]Finding, 0, len(findings))
	for _, finding := range findings {
		if finding.Severity == severity {
			filtered = append(filtered, finding)
		}
	}
	sort.Slice(filtered, func(i, j int) bool { return filtered[i].Key < filtered[j].Key })
	for index, finding := range filtered {
		_, _ = fmt.Fprintf(output, "%d.\nKey:\n  %s\n", index+1, finding.Key)
		if finding.Source != "" {
			_, _ = fmt.Fprintf(output, "Source:\n  %s\n", finding.Source)
		}
		if finding.Description != "" {
			_, _ = fmt.Fprintf(output, "Description:\n  %s\n", finding.Description)
		}
		if finding.Introduced != "" {
			_, _ = fmt.Fprintf(output, "Introduced:\n  %s\n", finding.Introduced)
		}
		if finding.Replacement != "" {
			_, _ = fmt.Fprintf(output, "Replacement:\n  %s\n", finding.Replacement)
		}
		output.WriteByte('\n')
	}
}

func hasSeverity(findings []Finding, severity Severity) bool {
	for _, finding := range findings {
		if finding.Severity == severity {
			return true
		}
	}
	return false
}

// WriteReport 将指定格式的报告写入输出流。
func WriteReport(writer io.Writer, report Report, format string) error {
	switch format {
	case "text":
		_, err := io.WriteString(writer, report.FormatText())
		return err
	case "json":
		encoded, err := report.FormatJSON()
		if err != nil {
			return err
		}
		_, err = writer.Write(append(encoded, '\n'))
		return err
	case "patch":
		_, err := io.WriteString(writer, report.FormatPatch())
		return err
	default:
		return fmt.Errorf("unsupported configuration report format %q", format)
	}
}
