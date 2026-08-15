package discovery

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/chasebank87/PourOver/internal/config"
	tmpl "github.com/chasebank87/PourOver/internal/template"
)

// DiscoverTemplateFiles renders each declared template and compares to the target.
// Sources are resolved relative to configDir when not absolute; targets expand ~.
// Source must exist and parse/render successfully.
func DiscoverTemplateFiles(templates []config.TemplateFile, configDir string, ctx tmpl.Context) ([]TemplateStatus, error) {
	configDir, err := filepath.Abs(configDir)
	if err != nil {
		return nil, fmt.Errorf("config directory: %w", err)
	}

	statuses := make([]TemplateStatus, 0, len(templates))
	for i, file := range templates {
		st, err := inspectTemplateFile(file, configDir, ctx)
		if err != nil {
			return nil, fmt.Errorf("files.templates[%d]: %w", i+1, err)
		}
		statuses = append(statuses, st)
	}
	return statuses, nil
}

func inspectTemplateFile(file config.TemplateFile, configDir string, ctx tmpl.Context) (TemplateStatus, error) {
	sourcePath, err := resolveSourcePath(file.Source, configDir)
	if err != nil {
		return TemplateStatus{}, err
	}
	targetPath, err := resolveTargetPath(file.Target)
	if err != nil {
		return TemplateStatus{}, err
	}

	srcBytes, err := os.ReadFile(sourcePath)
	if err != nil {
		if os.IsNotExist(err) {
			return TemplateStatus{}, fmt.Errorf("source %q does not exist", file.Source)
		}
		return TemplateStatus{}, fmt.Errorf("source %q: %w", file.Source, err)
	}

	rendered, err := tmpl.Render(string(srcBytes), ctx)
	if err != nil {
		return TemplateStatus{}, fmt.Errorf("render %q: %w", file.Source, err)
	}

	info, err := os.Lstat(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			return TemplateStatus{
				File:       file,
				SourcePath: sourcePath,
				TargetPath: targetPath,
				Rendered:   rendered,
				Kind:       TemplateStatusMissing,
			}, nil
		}
		return TemplateStatus{}, err
	}

	if info.IsDir() {
		return TemplateStatus{
			File:       file,
			SourcePath: sourcePath,
			TargetPath: targetPath,
			Rendered:   rendered,
			Kind:       TemplateStatusBlocked,
		}, nil
	}
	if info.Mode()&os.ModeSymlink != 0 {
		resolved, err := filepath.EvalSymlinks(targetPath)
		if err == nil {
			if st, err := os.Stat(resolved); err == nil && st.IsDir() {
				return TemplateStatus{
					File:       file,
					SourcePath: sourcePath,
					TargetPath: targetPath,
					Rendered:   rendered,
					Kind:       TemplateStatusBlocked,
				}, nil
			}
		}
	}

	currentBytes, err := os.ReadFile(targetPath)
	if err != nil {
		return TemplateStatus{}, fmt.Errorf("read target: %w", err)
	}
	current := string(currentBytes)
	kind := TemplateStatusSame
	if current != rendered {
		kind = TemplateStatusDiffer
	}
	return TemplateStatus{
		File:       file,
		SourcePath: sourcePath,
		TargetPath: targetPath,
		Rendered:   rendered,
		Current:    current,
		Kind:       kind,
	}, nil
}
