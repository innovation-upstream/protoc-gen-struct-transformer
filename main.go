// Transformation function generator for gRPC.
//
// Overview
//
// Protocol buffers complier (protoc) https://github.com/protocolbuffers/protobuf
// generates structures based on message definition in *.proto file. It's
// possible to use these generated structures directly, but it's better to have
// clear separation between transport level (gRPC) and business logic with its
// own structures. In this case you have to convert generated structures into
// structures use in business logic and vice versa.
//
// See documentation and usage examples on https://github.com/innovation-upstream/protoc-gen-struct-transformer/blob/master/README.md
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
	plugin "github.com/gogo/protobuf/protoc-gen-gogo/plugin"
	"github.com/golang/protobuf/proto"
	"github.com/innovation-upstream/protoc-gen-struct-transformer/generator"
	"github.com/pkg/errors"
	"golang.org/x/tools/imports"
)

var (
	packageName       = flag.String("package", "fallback", "Package name for generated functions.")
	helperPackageName = flag.String("helper-package", "", "Package name for helper functions.")
	versionFlag       = flag.Bool("version", false, "Print current version.")
	goimports         = flag.Bool("goimports", false, "Perform goimports on generated file.")
	debug             = flag.Bool("debug", false, "Add debug information to generated file.")
	usePackageInPath  = flag.Bool("use-package-in-path", true, "If true, package parameter will be used in path for output file.")
	paths             = flag.String("paths", "", "How to generate output filenames.")
)

type PathType int

const (
	pathTypeImport         PathType = 0
	pathTypeSourceRelative PathType = 1
)

func main() {
	flag.Parse()
	if *versionFlag {
		fmt.Println(generator.Version())
		os.Exit(0)
	}

	var gogoreq plugin.CodeGeneratorRequest

	data, err := ioutil.ReadAll(os.Stdin)
	must(err)
	must(proto.Unmarshal(data, &gogoreq))

	// Convert incoming parameters into CLI flags.
	must(generator.SetParameters(flag.CommandLine, gogoreq.Parameter))

	resp := &plugin.CodeGeneratorResponse{}
	optPath := ""

	messages, err := generator.CollectAllMessages(gogoreq)
	must(err)

	for _, f := range gogoreq.ProtoFile {
		var pathType PathType
		switch *paths {
		case "import":
			pathType = pathTypeImport
		case "source_relative":
			pathType = pathTypeSourceRelative
		default:
			log.Fatalf(`Unknown path type %q: want "import" or "source_relative".`, pathType)
		}

		content, err := generator.ProcessFile(f, packageName, helperPackageName, messages, *debug, *paths)
		if err != nil {
			if err != generator.ErrFileSkipped {
				must(err)
			}
			continue
		}

		filename := GoFileName(f, pathType, *packageName)

		content, err = runGoimports(filename, content)
		if err != nil {
			if err != generator.ErrFileSkipped {
				must(err)
			}
			continue
		}

		resp.File = append(resp.File, &plugin.CodeGeneratorResponse_File{
			Name:    proto.String(filename),
			Content: proto.String(content),
		})

		// Generate transformers for dependency
		currentFilename := GoFileName(f, pathType, *packageName)
		depFiles, err := ProcessDependency(gogoreq.ProtoFile, f, messages, pathType, currentFilename)
		if err != nil {
			must(err)
		}

		resp.File = append(resp.File, depFiles...)

		// Generate options.go
		optPath = filename

		if optPath != "" {
			optPath = filepath.Dir(optPath) + "/options.go"

			content, err := runGoimports(optPath, generator.OptHelpers(*packageName))
			if err != nil {
				if err != generator.ErrFileSkipped {
					must(err)
				}
			}

			resp.File = append(resp.File, &plugin.CodeGeneratorResponse_File{
				Name:    proto.String(optPath),
				Content: proto.String(content),
			})
		}
	}

	// Send back the results.
	data, err = proto.Marshal(resp)
	must(err)

	_, err = os.Stdout.Write(data)
	must(err)
}

func must(err error) {
	if err != nil {
		if *debug {
			log.Fatalf("%+v", err)
		} else {
			log.Fatalf("%v", err)
		}
	}
}

func runGoimports(filename, content string) (string, error) {
	if !*goimports {
		return content, nil
	}

	formatted, err := imports.Process(filename, []byte(content), nil)
	return string(formatted), err
}

func GoFileName(d *descriptor.FileDescriptorProto, pathType PathType, pn string) string {
	name := d.GetName()
	dir, name := filepath.Split(name)
	name = strings.Replace(filepath.Join(dir, pn, name), ".proto", "_transformer.go", -1)

	if pathType == pathTypeSourceRelative {
		return name
	}

	// Does the file have a "go_package" option?
	// If it does, it may override the filename.
	if impPath := d.GetOptions().GetGoPackage(); impPath != "" {
		// Replace the existing dirname with the declared import path.
		_, name = path.Split(name)
		name = path.Join(string(impPath), pn, name)
		return name
	}

	return name
}

func ProcessDependency(allProtos []*descriptor.FileDescriptorProto, currentProto *descriptor.FileDescriptorProto, messages generator.MessageOptionList, pathType PathType, currentFilename string) ([]*plugin.CodeGeneratorResponse_File, error) {
	var files []*plugin.CodeGeneratorResponse_File
	for _, d := range currentProto.GetDependency() {
	ap:
		for _, p := range allProtos {
			if p.GetName() == d {
				content, err := generator.ProcessFile(p, packageName, helperPackageName, messages, *debug, *paths)
				if err != nil {
					if err != generator.ErrFileSkipped {
						return files, errors.WithStack(err)
					}
					break ap
				}

				filename := GoFileName(p, pathType, *packageName)
				filename = strings.Replace(filename, filepath.Dir(filename), filepath.Dir(currentFilename), -1)

				content, err = runGoimports(filename, content)
				if err != nil {
					if err != generator.ErrFileSkipped {
						return files, errors.WithStack(err)
					}
					break ap
				}

				files = append(files, &plugin.CodeGeneratorResponse_File{
					Name:    proto.String(filename),
					Content: proto.String(content),
				})

				allTransitiveDepFiles, err := ProcessDependency(allProtos, p, messages, pathType, currentFilename)
				if err != nil {
					return files, errors.WithStack(err)
				}

				var transitiveDepFiles []*plugin.CodeGeneratorResponse_File
				// De-dupe allTransitiveDepFiles
			tdl:
				for _, t := range allTransitiveDepFiles {
					for _, f := range files {
						if f.GetName() == t.GetName() {
							continue tdl
						}
					}
					transitiveDepFiles = append(transitiveDepFiles, t)
				}

				files = append(files, transitiveDepFiles...)

				break ap
			}
		}
	}

	return files, nil
}
