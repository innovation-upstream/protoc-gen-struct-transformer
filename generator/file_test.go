package generator

import (
	"bytes"
	"io/ioutil"
	"path/filepath"
	"runtime"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
	plugin "github.com/gogo/protobuf/protoc-gen-gogo/plugin"
	"github.com/innovation-upstream/protoc-gen-struct-transformer/options"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("File", func() {

	Describe("File headers", func() {

		Context("when get a header", func() {

			BeforeEach(func() {
				version = "v0.0.1"
				buildTime = time.Date(2019, time.March, 1, 5, 34, 19, 0, time.UTC).Format(time.RFC3339)
			})

			It("returns version header", func() {
				v := Version()
				Expect(v).To(Equal("version: v0.0.1\nbuild-time: 2019-03-01T05:34:19Z\n"))
			})

			It("return WriteStringer with header", func() {
				o := output()
				Expect(o.String()).To(Equal("// Code generated by protoc-gen-struct-transformer, version: v0.0.1. DO NOT EDIT.\n"))
			})
		})
	})

	Describe("CollectAllMessages", func() {
		var mt = &descriptor.DescriptorProto{
			Name:    sp("message_name"),
			Options: &descriptor.MessageOptions{},
		}

		BeforeEach(func() {
			err := proto.SetExtension(mt.Options, options.E_GoStruct, sp("go_struct_name"))
			Expect(err).NotTo(HaveOccurred())
		})

		DescribeTable("check code generator request",
			func(req plugin.CodeGeneratorRequest, expectexList MessageOptionList) {
				mol, err := CollectAllMessages(req)
				Expect(err).NotTo(HaveOccurred())

				if len(expectexList) > 0 {
					m := mol["pb.message_name"]
					Expect(m).NotTo(BeNil())

					e := expectexList["message_name"]
					Expect(e).NotTo(BeNil())

					Expect(m.Target()).To(Equal(e.Target()))
					Expect(m.Full()).To(Equal(e.Full()))
					Expect(m.OneofDecl()).To(Equal(e.OneofDecl()))
				}
			},

			Entry("Empty file list", plugin.CodeGeneratorRequest{
				ProtoFile: []*descriptor.FileDescriptorProto{
					&descriptor.FileDescriptorProto{
						Name:        sp("protofile"),
						Package:     sp("pb"),
						MessageType: []*descriptor.DescriptorProto{},
					},
				},
			}, map[string]MessageOption{}),

			Entry("Messages without go_struct option", plugin.CodeGeneratorRequest{
				ProtoFile: []*descriptor.FileDescriptorProto{
					&descriptor.FileDescriptorProto{
						Name:    sp("protofile"),
						Package: sp("pb"),
						MessageType: []*descriptor.DescriptorProto{
							{Name: sp("message_name")},
						},
					},
				},
			}, map[string]MessageOption{
				"message_name": messageOption{targetName: "", fullName: "", oneofDecl: ""},
			}),

			Entry("Messages with go_struct option", plugin.CodeGeneratorRequest{
				ProtoFile: []*descriptor.FileDescriptorProto{
					&descriptor.FileDescriptorProto{
						Name:        sp("protofile"),
						Package:     sp("pb"),
						MessageType: []*descriptor.DescriptorProto{mt},
					},
				},
			}, map[string]MessageOption{
				"message_name": messageOption{targetName: "go_struct_name", fullName: "", oneofDecl: ""},
			}),

			Entry("Messages with oneOf declaration which does match to int64toString rule", plugin.CodeGeneratorRequest{
				ProtoFile: []*descriptor.FileDescriptorProto{
					&descriptor.FileDescriptorProto{
						Name:    sp("protofile"),
						Package: sp("pb"),
						MessageType: []*descriptor.DescriptorProto{
							{
								Name: sp("message_name"),
								OneofDecl: []*descriptor.OneofDescriptorProto{
									{Name: sp("oneof_decl_name")},
								},
								Field: []*descriptor.FieldDescriptorProto{
									{Name: sp("int64_value")},
									{Name: sp("string_value")},
								},
							},
						},
					},
				},
			}, map[string]MessageOption{
				"message_name": messageOption{targetName: "", fullName: "", oneofDecl: "oneof_decl_name"},
			}),

			Entry("Messages with oneOf declaration which does not match to int64toString", plugin.CodeGeneratorRequest{
				ProtoFile: []*descriptor.FileDescriptorProto{
					&descriptor.FileDescriptorProto{
						Name:    sp("protofile"),
						Package: sp("pb"),
						MessageType: []*descriptor.DescriptorProto{
							{
								Name: sp("message_name"),
								OneofDecl: []*descriptor.OneofDescriptorProto{
									{Name: sp("oneof_decl_name")},
								},
								Field: []*descriptor.FieldDescriptorProto{
									{Name: sp("some_field")},
									{Name: sp("some_other_field")},
								},
							},
						},
					},
				},
			}, map[string]MessageOption{
				"message_name": messageOption{targetName: "", fullName: "", oneofDecl: ""},
			}),
		)
	})

	Describe("ProcessFile", func() {
		Context("when get a header", func() {
			var f *descriptor.FileDescriptorProto

			BeforeEach(func() {
				f = &descriptor.FileDescriptorProto{
					Options: &descriptor.FileOptions{},
					Name:    sp("product.proto"),
					Package: sp("pb"),
					MessageType: []*descriptor.DescriptorProto{
						{
							Name: sp("Product"),
							Field: []*descriptor.FieldDescriptorProto{
								&descriptor.FieldDescriptorProto{
									Name:     sp("id"),
									Number:   nil,
									Label:    nil,
									Type:     &typInt64,
									TypeName: nil,
									Options:  &descriptor.FieldOptions{},
								},
							},
							Options: &descriptor.MessageOptions{},
						},
					},
				}

				err := proto.SetExtension(f.Options, options.E_GoModelsFilePath, sp("testdata/model.go"))
				Expect(err).NotTo(HaveOccurred())

				err = proto.SetExtension(f.MessageType[0].Options, options.E_GoStruct, sp("Product"))
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns generated code", func() {
				expectedContent, err := ioutil.ReadFile("testdata/processfile.go.golden")
				Expect(err).NotTo(HaveOccurred())

				content, err := ProcessFile(f, sp("product"), sp("helper-package"), map[string]MessageOption{}, false, "")
				Expect(err).NotTo(HaveOccurred())
				Expect(content).To(Equal(string(expectedContent)))
				//Expect(absPath).To(Equal("product_transformer.go"))
			})
		})
	})

	Describe("modelPath", func() {

		Context("when there is no option go_models_file_path in file", func() {

			It("returns files was skipped error", func() {
				p, err := modelsPath(&descriptor.FileOptions{})
				Expect(err).To(MatchError("files was skipped"))
				Expect(p).To(Equal(""))
			})
		})

		Context("when file contains go_models_file_path option", func() {
			var (
				f    *descriptor.FileDescriptorProto
				base string
				path string
			)

			BeforeEach(func() {
				f = &descriptor.FileDescriptorProto{
					Options: &descriptor.FileOptions{},
				}
				_, path, _, _ = runtime.Caller(0)
				base = filepath.Base(path)

				// set path to current test file as a value for go_models_file_path option
				err := proto.SetExtension(f.Options, options.E_GoModelsFilePath, sp(base))
				Expect(err).NotTo(HaveOccurred())
			})

		})
	})

	Describe("prefixFields", func() {

		DescribeTable("check returns",
			func(pref string, fields, expected []Field) {
				prefixFields(fields, pref)
				Expect(fields).To(Equal(expected))
			},

			Entry("One field, do not use package", "pref",
				[]Field{{ProtoToGoType: "p2g", GoToProtoType: "g2p"}},
				[]Field{{ProtoToGoType: "p2g", GoToProtoType: "g2p"}},
			),

			Entry("One field, use package, empty prefix", "",
				[]Field{{ProtoToGoType: "p2g", GoToProtoType: "g2p", UsePackage: true}},
				[]Field{{ProtoToGoType: "p2g", GoToProtoType: "g2p", UsePackage: true}},
			),

			Entry("One field, use package", "pref",
				[]Field{{ProtoToGoType: "p2g", GoToProtoType: "g2p", UsePackage: true}},
				[]Field{{ProtoToGoType: "pref.p2g", GoToProtoType: "pref.g2p", UsePackage: true}},
			),

			Entry("One field, use package", "pref",
				[]Field{{ProtoToGoType: "p2g", GoToProtoType: "g2p", UsePackage: true}},
				[]Field{{ProtoToGoType: "pref.p2g", GoToProtoType: "pref.g2p", UsePackage: true}},
			),
		)
	})

	Describe("fileHeader", func() {

		BeforeEach(func() {
			version = "v1.0.0"
		})

		DescribeTable("check results",
			func(f, p, d, expected string) {
				ws := fileHeader(f, p, d)
				Expect(ws.String()).To(Equal(expected))
			},
			Entry("case 1", "srcfile", "srcpackage", "dstpackage", `// Code generated by protoc-gen-struct-transformer, version: v1.0.0. DO NOT EDIT.
// source file: srcfile
// source package: srcpackage

package dstpackage
`),
			Entry("case 2", "abc", "cde", "fff", `// Code generated by protoc-gen-struct-transformer, version: v1.0.0. DO NOT EDIT.
// source file: abc
// source package: cde

package fff
`),
		)
	})

	Describe("execTemplate", func() {

		DescribeTable("check results",
			func(d []*Data) {
				buf := []byte{}
				w := bytes.NewBuffer(buf)

				err := execTemplate(w, d)
				Expect(err).NotTo(HaveOccurred())
			},
			Entry("", nil),
			Entry("", []*Data{
				&Data{
					SrcPref:       "src_pref",
					Src:           "src",
					SrcFn:         "src_fn",
					SrcPointer:    "src_pointer",
					DstPref:       "dst_pref",
					Dst:           "dst",
					DstFn:         "dst_fn",
					DstPointer:    "dst_pointer",
					Swapped:       false,
					HelperPackage: "hp",
					Ptr:           false,
					Fields: []Field{
						{
							Name:           "FirstField",
							ProtoName:      "proto_name",
							ProtoType:      "proto_type",
							ProtoToGoType:  "FirstProto2go",
							GoToProtoType:  "FirstGo2proto",
							GoIsPointer:    false,
							ProtoIsPointer: false,
							UsePackage:     false,
							OneofDecl:      "",
							Opts:           "",
						},
						{
							Name:           "SecondField",
							ProtoName:      "proto_name2",
							ProtoType:      "proto_type2",
							ProtoToGoType:  "SecondProto2go",
							GoToProtoType:  "SecondGo2proto",
							GoIsPointer:    false,
							ProtoIsPointer: false,
							UsePackage:     false,
							OneofDecl:      "",
							Opts:           "",
						},
					},
				},
			}),
		)

	})

})
