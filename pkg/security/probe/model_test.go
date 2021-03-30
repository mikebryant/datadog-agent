// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

// +build linux

package probe

import (
	"reflect"
	"testing"

	"github.com/DataDog/datadog-agent/pkg/security/model"
	"github.com/DataDog/datadog-agent/pkg/security/secl/eval"
)

func TestPathValidation(t *testing.T) {
	mod := &Model{}
	if err := mod.ValidateField("open.file.path", eval.FieldValue{Value: "/var/log/*"}); err != nil {
		t.Fatalf("shouldn't return an error: %s", err)
	}
	if err := mod.ValidateField("open.file.path", eval.FieldValue{Value: "~/apache/httpd.conf"}); err == nil {
		t.Fatal("should return an error")
	}
	if err := mod.ValidateField("open.file.path", eval.FieldValue{Value: "../../../etc/apache/httpd.conf"}); err == nil {
		t.Fatal("should return an error")
	}
	if err := mod.ValidateField("open.file.path", eval.FieldValue{Value: "/etc/apache/./httpd.conf"}); err == nil {
		t.Fatal("should return an error")
	}
	if err := mod.ValidateField("open.file.path", eval.FieldValue{Value: "*/"}); err == nil {
		t.Fatal("should return an error")
	}

	var val string
	for i := 0; i <= model.MaxPathDepth; i++ {
		val += "a/"
	}
	if err := mod.ValidateField("open.file.path", eval.FieldValue{Value: val}); err == nil {
		t.Fatal("should return an error")
	}

	val = ""
	for i := 0; i <= model.MaxSegmentLength; i++ {
		val += "a"
	}
	if err := mod.ValidateField("open.file.path", eval.FieldValue{Value: val}); err == nil {
		t.Fatal("should return an error")
	}

	if err := mod.ValidateField("open.file.path", eval.FieldValue{Value: ".*", Type: eval.RegexpValueType}); err == nil {
		t.Fatal("should return an error")
	}

	if err := mod.ValidateField("open.file.path", eval.FieldValue{Value: "/etc/*", Type: eval.PatternValueType}); err != nil {
		t.Fatal("shouldn't return an error")
	}
}

func TestSetFieldValue(t *testing.T) {
	event := &Event{}

	for _, field := range event.GetFields() {
		kind, err := event.GetFieldType(field)
		if err != nil {
			t.Fatal(err)
		}

		switch kind {
		case reflect.String:
			if err = event.SetFieldValue(field, "aaa"); err != nil {
				t.Fatal(err)
			}
		case reflect.Int:
			if err = event.SetFieldValue(field, 123); err != nil {
				t.Fatal(err)
			}
		case reflect.Bool:
			if err = event.SetFieldValue(field, true); err != nil {
				t.Fatal(err)
			}
		default:
			t.Fatalf("type unknown: %v", kind)
		}
	}
}
