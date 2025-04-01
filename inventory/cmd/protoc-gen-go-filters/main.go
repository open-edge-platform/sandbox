// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

// This generator generates resource transpilers.

package main

import (
	"fmt"
	"html/template"
	"io"
	"regexp"
	gosort "sort"
	"strings"

	"github.com/iancoleman/strcase"
	"github.com/rs/zerolog"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/util/collections"
)

const (
	thisGenerator = "protoc-gen-go-filters"
	// template used by generator.
	templateFile = "cmd/" + thisGenerator + "/template.gotmpl"
)

var (
	log = logging.GetLogger(thisGenerator).Level(zerolog.InfoLevel)

	// TODO: work on addressing those naming inconsistencies.
	resourceIdentifiersOutOfNamingConvention = map[string]string{
		"operatingsystemresource":   "os",
		"operatingsystem":           "os",
		"workloadmember":            "workload_member",
		"remoteaccessconfiguration": "rmt_access_conf",
		"telemetryprofile":          "telemetry_profile",
		"telemetrygroupresource":    "telemetry_group",
		"telemetrygroup":            "telemetry_group",
	}

	// Generator skips intentionally .proto definitions listed below.
	// None of them contain inventory resource requiring transpilers generation.
	excludedProtoPackages = []string{"inventory.v1", "status.v1", "ent", "errors", "infrainv"}
)

func main() {
	var rts []ResourceTranspiler

	protogen.Options{}.Run(func(gen *protogen.Plugin) error {
		log.Trace().Msgf("PATHS: %+v", maps.Keys(gen.FilesByPath))

		filteredFiles := collections.Filter[*protogen.File](gen.Files, func(f *protogen.File) bool {
			if slices.Contains(excludedProtoPackages, string(f.Desc.FullName())) {
				log.Debug().Msgf("Skipping proto file (%v)", f.Desc.Path())
				return false
			}
			return true
		})
		filteredFiles = collections.Filter[*protogen.File](filteredFiles, func(f *protogen.File) bool {
			return f.Generate
		})

		log.Trace().Msgf("All files(%d), filtered files: %d", len(gen.Files), len(filteredFiles))

		for _, f := range filteredFiles {
			log.Debug().Msgf("Processing proto file (%v)", f.Desc.Path())
			res := collections.MapSlice[*protogen.Message, ResourceTranspiler](
				f.Messages,
				func(m *protogen.Message) ResourceTranspiler {
					edges := collections.Filter[*protogen.Field](m.Fields, func(field *protogen.Field) bool {
						return field.Desc.Kind() == protoreflect.MessageKind
					})

					ehs := collections.MapSlice[*protogen.Field, EdgeHandler](edges, func(f *protogen.Field) EdgeHandler {
						log.Trace().Msgf(">> Edge detected (%v)", f.Desc.Name())
						return EdgeHandler{
							Name:               fieldDescNameAsEdgeName(f),
							TargetResourceKind: asResourceKind(string(f.Desc.Message().Name())),
						}
					})

					return ResourceTranspiler{
						ResourceName: string(m.Desc.Name()),
						ResourceKind: asResourceKind(string(m.Desc.Name())),
						EntPkg:       strings.ToLower(string(m.Desc.Name())),
						EdgeHandlers: ehs,
					}
				})
			rts = append(rts, sort(res)...)
		}

		gen.Error(serialize(gen, rts))
		return nil
	})
}

func serialize(gen *protogen.Plugin, rts []ResourceTranspiler) error {
	for _, rt := range rts {
		targetFileName := createOutputFileName(rt.ResourceName)
		log.Info().Msgf("Generating %s for (%s)", targetFileName, rt.ResourceName)
		targetFile := gen.NewGeneratedFile(targetFileName, "")
		if err := evaluateTemplate(targetFile, rt); err != nil {
			return err
		}
	}
	return nil
}

func createOutputFileName(resourceName string) string {
	normalizedResourceName := normalizeResourceID(resourceName)
	normalizedLower := strings.ToLower(normalizedResourceName)
	shortName := strings.ReplaceAll(strings.ReplaceAll(normalizedLower, "_", ""), "resource", "")
	return shortName + "_transpiler.go"
}

func fieldDescNameAsEdgeName(field *protogen.Field) string {
	return strcase.ToCamel("edge" + "-" + string(field.Desc.Name()))
}

func evaluateTemplate(out io.Writer, rt ResourceTranspiler) error {
	t, err := template.New("").
		Funcs(
			template.FuncMap{
				"trimPrefix": strings.TrimPrefix,
				"toCamel":    strcase.ToCamel,
			}).ParseFiles(templateFile)
	if err != nil {
		return err
	}
	return t.ExecuteTemplate(out, "registrar", rt)
}

func asResourceKind(s string) string {
	return "ResourceKind_RESOURCE_KIND_" + strings.ToUpper(strings.TrimSuffix(normalizeResourceID(s), "Resource"))
}

func normalizeResourceID(s string) string {
	for is, shallBe := range resourceIdentifiersOutOfNamingConvention {
		s = regexp.MustCompile(fmt.Sprintf(`(?i)\b%s\b`, is)).ReplaceAllString(s, shallBe)
	}
	return s
}

func sort(ts []ResourceTranspiler) []ResourceTranspiler {
	gosort.Slice(ts, func(i, j int) bool {
		return ts[i].EntPkg < ts[j].EntPkg
	})
	for _, t := range ts {
		gosort.SliceStable(t.EdgeHandlers, func(i, j int) bool {
			return t.EdgeHandlers[i].Name < t.EdgeHandlers[j].Name
		})
	}
	return ts
}

type ResourceTranspiler struct {
	ResourceName string
	ResourceKind string
	EntPkg       string
	EdgeHandlers []EdgeHandler
}

type EdgeHandler struct {
	Name               string
	TargetResourceKind string
}
