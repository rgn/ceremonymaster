package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
)

// buildGroups constructs huh.Groups from GroupConfig entries. If reviewers
// are provided and a group's name is "Wertung" it will expand that group into
// one per reviewer, suffixing keys with the reviewer key to avoid collisions.
func (m *Model) buildGroups(groupCfgs []GroupConfig) []*huh.Group {
	var res []*huh.Group

	for _, gcfg := range groupCfgs {
		groupKey := gcfg.Key
		// Default handling for non-Wertung groups (or Wertung when no reviewers)
		var fields []huh.Field
		for _, fc := range gcfg.Fields {
			fcKey := BuildFieldKey(groupKey, fc.Key)
			switch fc.Type {
			case "range":
				v := ""
				m.Values[fc.Key] = &v
				sel := huh.NewSelect[string]().
					Key(fcKey).
					Value(&v).
					Title(fc.Title).
					Description(fc.Description)
				sel = sel.Options(
					huh.NewOption[string]("", "0"),
					huh.NewOption[string]("⭐", "1"),
					huh.NewOption[string]("⭐⭐", "2"),
					huh.NewOption[string]("⭐⭐⭐", "3"),
					huh.NewOption[string]("⭐⭐⭐⭐", "4"),
					huh.NewOption[string]("⭐⭐⭐⭐⭐", "5"),
				)
				if fc.Mandatory {
					sel = sel.Validate(func(s string) error {
						if strings.TrimSpace(s) == "" {
							return fmt.Errorf("%s is required", fc.Title)
						}
						return nil
					})
				}
				fields = append(fields, sel)
			case "input":
				v := ""
				m.Values[fc.Key] = &v
				inp := huh.NewInput().
					Key(fcKey).
					Value(&v).
					Title(fc.Title).
					Description(fc.Description)
				if fc.Mandatory {
					inp = inp.Validate(func(s string) error {
						if strings.TrimSpace(s) == "" {
							return fmt.Errorf("%s is required", fc.Title)
						}
						return nil
					})
				}
				fields = append(fields, inp)
			case "select":
				v := ""
				m.Values[fc.Key] = &v
				sel := huh.NewSelect[string]().
					Key(fcKey).
					Value(&v).
					Title(fc.Title).
					Description(fc.Description)
				if len(fc.Options) > 0 {
					sel = sel.Options(huh.NewOptions[string](fc.Options...)...)
				}
				if fc.Mandatory {
					sel = sel.Validate(func(s string) error {
						if strings.TrimSpace(s) == "" {
							return fmt.Errorf("%s is required", fc.Title)
						}
						return nil
					})
				}
				fields = append(fields, sel)
			case "text":
				v := ""
				m.Values[fc.Key] = &v
				txt := huh.NewText().
					Key(fcKey).
					Value(&v).
					Title(fc.Title).
					Description(fc.Description)
				if fc.Mandatory {
					txt = txt.Validate(func(s string) error {
						if strings.TrimSpace(s) == "" {
							return fmt.Errorf("%s is required", fc.Title)
						}
						return nil
					})
				}
				fields = append(fields, txt)
			case "filepicker":
				v := ""
				m.Values[fc.Key] = &v
				fp := huh.NewFilePicker().
					Key(fcKey).
					Value(&v).
					Title(fc.Title).
					Description(fc.Description).
					AllowedTypes(fc.Options)
				if fc.Mandatory {
					fp = fp.Validate(func(s string) error {
						if strings.TrimSpace(s) == "" {
							return fmt.Errorf("%s is required", fc.Title)
						}
						return nil
					})
				}
				fields = append(fields, fp)
			case "confirm":
				b := false
				m.Values[fc.Key] = &b
				conf := huh.NewConfirm().
					Key(fc.Key).
					Value(&b).
					Title(fc.Title)
				if fc.Description != "" {
					conf = conf.Description(fc.Description)
				}
				if fc.Affirmative != "" {
					conf = conf.Affirmative(fc.Affirmative)
				}
				if fc.Negative != "" {
					conf = conf.Negative(fc.Negative)
				}
				if fc.RequireYes {
					conf = conf.Validate(func(v bool) error {
						if !v {
							return fmt.Errorf("Welp, finish up then")
						}
						return nil
					})
				}
				fields = append(fields, conf)
			case "multiselect":
				var vs []string
				m.Values[fc.Key] = &vs
				ms := huh.NewMultiSelect[string]().
					Key(fcKey).
					Value(&vs).
					Title(fc.Title).
					Description(fc.Description)
				if len(fc.Options) > 0 {
					ms = ms.Options(huh.NewOptions[string](fc.Options...)...)
				}
				if fc.Mandatory {
					ms = ms.Validate(func(s []string) error {
						if len(s) == 0 {
							return fmt.Errorf("%s is required", fc.Title)
						}
						return nil
					})
				}
				fields = append(fields, ms)
			default:
				// unknown field type: ignore
			}
		}

		group := huh.NewGroup(fields...).
			Title(gcfg.Title).
			Description(gcfg.Description + "\n")

		res = append(res, group)
	}

	return res
}
