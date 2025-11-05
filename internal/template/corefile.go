package template

import (
	"fmt"
	"os"
	"strings"
	"text/template"

	"netbird-coredns/internal/config"
)

const corefileTemplate = `.{{ if ne .DNSPort 53 }}:{{ .DNSPort }}{{ end }} {
    netbird {{ .DomainsString }}
{{- if .ForwardTo }}
    forward . {{ .ForwardTo }}
{{- end }}
    log
    errors
}
`

// CorefileData represents the data used to generate the Corefile
type CorefileData struct {
	DomainsString string
	ForwardTo     string
	DNSPort       int
}

// Generator handles Corefile generation
type Generator struct {
	template *template.Template
}

// NewGenerator creates a new Corefile generator
func NewGenerator() (*Generator, error) {
	tmpl, err := template.New("corefile").Parse(corefileTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse corefile template: %w", err)
	}

	return &Generator{
		template: tmpl,
	}, nil
}

// GenerateCorefile generates a Corefile based on the provided configuration
func (g *Generator) GenerateCorefile(cfg *config.Config) (string, error) {
	// Join domains with spaces for the netbird plugin line
	domainsString := strings.Join(cfg.Domains, " ")

	data := CorefileData{
		DomainsString: domainsString,
		ForwardTo:     cfg.ForwardTo,
		DNSPort:       cfg.DNSPort,
	}

	var buf strings.Builder
	if err := g.template.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute corefile template: %w", err)
	}

	return buf.String(), nil
}

// WriteCorefile generates and writes a Corefile to the specified path
func (g *Generator) WriteCorefile(cfg *config.Config, outputPath string) error {
	content, err := g.GenerateCorefile(cfg)
	if err != nil {
		return fmt.Errorf("failed to generate corefile: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write corefile to %s: %w", outputPath, err)
	}

	return nil
}

