// NOTE: This file is NOT autogenerated and contains custom transformers,
//       which are used in the message_transformer.go

package transform

import "github.com/ZacxDev/protoc-gen-struct-transformer/example"

// PbCustomTypeToStringPtrVal is an example of the custom transformer from Pb to go
func PbCustomTypeToStringPtrVal(src *example.CustomType, opts ...TransformParam) string {
	applyOptions(opts...)

	if version == "v2" {
		return src.Value
	}

	return ""
}

// StringToPbCustomTypeValPtr is an example of the custom transformer from go to Pb
func StringToPbCustomTypeValPtr(src string, opts ...TransformParam) *example.CustomType {
	applyOptions(opts...)

	if version == "v2" {
		return &example.CustomType{
			Value: src,
		}
	}

	return nil
}

// PbCustomOneofToStringPtrVal is an example of the custom transformer from Pb to go for the object with oneof in it
func PbCustomOneofToStringPtrVal(src *example.CustomOneof, opts ...TransformParam) string {
	applyOptions(opts...)

	if version == "v2" {
		return src.GetStringValue()
	}

	return ""
}

// StringToPbCustomOneofValPtr is an example of the custom transformer from go to Pb for the object with oneof in it
func StringToPbCustomOneofValPtr(src string, opts ...TransformParam) *example.CustomOneof {
	applyOptions(opts...)

	if version == "v2" {
		return &example.CustomOneof{
			Value: &example.CustomOneof_StringValue{
				StringValue: src,
			},
		}
	}

	return nil
}

// ToPbValPtr is a transformer for a non-supported type
// TODO: We need to rename the method to include a field name or type to it.
//       So we can have more than 1 unsupported type in the file.
//       Alternatively you can use `custom` attribute to make a custom transformer
//       Current implementation is a bug, but it is used as a feature,
//       so changing the method name will break backward compatibilty
func ToPbValPtr(src string, opts ...TransformParam) *example.NotSupportedOneOf {
	return &example.NotSupportedOneOf{TheDecl: &example.NotSupportedOneOf_StringValue{StringValue: src}}
}

// PbToPtrVal is a transformer for a non-supported type
// See TODO for ToPbValPtr
func PbToPtrVal(src *example.NotSupportedOneOf, opts ...TransformParam) string {
	return src.GetStringValue()
}
