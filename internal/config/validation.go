package config

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
)

type validationField struct {
	key    string
	value  reflect.Value
	schema reflect.StructField
}

type validationContext struct {
	field    validationField
	fileInfo fs.FileInfo
}

type validatorDefinition struct {
	stopOnFailure bool
	validate      func(*validationContext, string) error
}

type validationIssue struct {
	key       string
	validator string
	err       error
	hint      string
}

type validationIssues []validationIssue

func (issues validationIssues) Error() string {
	var message strings.Builder
	message.WriteString("configuration validation failed:")
	for _, issue := range issues {
		fmt.Fprintf(&message, "\n- %s (%s): %v", issue.key, issue.validator, issue.err)
		if issue.hint != "" {
			fmt.Fprintf(&message, "; set with: kstankcli config set %s %s", issue.key, issue.hint)
		}
	}
	return message.String()
}

var fieldValidators = map[string]validatorDefinition{
	"required":  {stopOnFailure: true, validate: validateRequired},
	"absolute":  {stopOnFailure: true, validate: validateAbsolute},
	"exists":    {stopOnFailure: true, validate: validateExists},
	"directory": {validate: validateDirectory},
	"file":      {validate: validateFile},
	"oneOf":     {stopOnFailure: true, validate: validateOneOf},
}

// ValidateAllFields runs every validator declared by the configuration schema.
// All value-validation failures are returned together.
func ValidateAllFields(cfg Config) error {
	fields, err := validationFields(reflect.ValueOf(cfg))
	if err != nil {
		return err
	}
	return validateSelectedFields(fields)
}

// ValidateFields runs the declared validators for the requested dot-separated
// JSON field paths. All value-validation failures are returned together.
func ValidateFields(cfg Config, keys ...string) error {
	if len(keys) == 0 {
		return fmt.Errorf("at least one config field is required for validation")
	}
	fields, err := validationFields(reflect.ValueOf(cfg))
	if err != nil {
		return err
	}
	byKey := make(map[string]validationField, len(fields))
	for _, field := range fields {
		byKey[field.key] = field
	}
	selected := make([]validationField, 0, len(keys))
	for _, key := range keys {
		field, ok := byKey[key]
		if !ok {
			return fmt.Errorf("unknown config field %q", key)
		}
		selected = append(selected, field)
	}
	return validateSelectedFields(selected)
}

func validationFields(value reflect.Value) ([]validationField, error) {
	var fields []validationField
	if err := collectValidationFields(value, nil, &fields); err != nil {
		return nil, err
	}
	sort.Slice(fields, func(i, j int) bool {
		return fields[i].key < fields[j].key
	})
	return fields, nil
}

func collectValidationFields(value reflect.Value, prefix []string, fields *[]validationField) error {
	valueType := value.Type()
	for index := 0; index < valueType.NumField(); index++ {
		schema := valueType.Field(index)
		name, ok := jsonFieldName(schema)
		if !ok {
			continue
		}
		path := appendPath(prefix, name)
		fieldValue := value.Field(index)
		if fieldValue.Kind() == reflect.Struct {
			if err := collectValidationFields(fieldValue, path, fields); err != nil {
				return err
			}
			continue
		}
		if schema.Tag.Get("validate") == "" {
			continue
		}
		*fields = append(*fields, validationField{
			key:    strings.Join(path, "."),
			value:  fieldValue,
			schema: schema,
		})
	}
	return nil
}

func validateSelectedFields(fields []validationField) error {
	var issues validationIssues
	for _, field := range fields {
		fieldIssues, err := validateField(field)
		if err != nil {
			return err
		}
		issues = append(issues, fieldIssues...)
	}
	if len(issues) > 0 {
		return issues
	}
	return nil
}

func validateField(field validationField) ([]validationIssue, error) {
	context := validationContext{field: field}
	var issues []validationIssue
	for _, specification := range strings.Split(field.schema.Tag.Get("validate"), ",") {
		name, argument, _ := strings.Cut(specification, "=")
		definition, ok := fieldValidators[name]
		if !ok {
			return nil, fmt.Errorf("config field %q uses unknown validator %q", field.key, name)
		}
		if err := definition.validate(&context, argument); err != nil {
			issues = append(issues, validationIssue{key: field.key, validator: name, err: err, hint: configSetHint(field.schema)})
			if definition.stopOnFailure {
				break
			}
		}
	}
	return issues, nil
}

func configSetHint(field reflect.StructField) string {
	if hasConfigOption(field, configOptionReadOnly) {
		return ""
	}
	if hasConfigOption(field, configOptionPath) {
		return "<absolute-path>"
	}
	return "<value>"
}

func validateRequired(context *validationContext, argument string) error {
	if err := rejectValidatorArgument(context.field.key, "required", argument); err != nil {
		return err
	}
	value, err := stringFieldValue(context, "required")
	if err != nil {
		return err
	}
	if value == "" {
		return fmt.Errorf("must not be empty")
	}
	return nil
}

func validateAbsolute(context *validationContext, argument string) error {
	if err := rejectValidatorArgument(context.field.key, "absolute", argument); err != nil {
		return err
	}
	value, err := stringFieldValue(context, "absolute")
	if err != nil {
		return err
	}
	if !filepath.IsAbs(value) {
		return fmt.Errorf("%q must be an absolute path", value)
	}
	return nil
}

func validateExists(context *validationContext, argument string) error {
	if err := rejectValidatorArgument(context.field.key, "exists", argument); err != nil {
		return err
	}
	value, err := stringFieldValue(context, "exists")
	if err != nil {
		return err
	}
	info, err := os.Stat(value)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%q does not exist", value)
		}
		return fmt.Errorf("inspect %q: %w", value, err)
	}
	context.fileInfo = info
	return nil
}

func validateDirectory(context *validationContext, argument string) error {
	if err := rejectValidatorArgument(context.field.key, "directory", argument); err != nil {
		return err
	}
	info, err := statValidationPath(context, "directory")
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%q must be a directory", context.field.value.String())
	}
	return nil
}

func validateFile(context *validationContext, argument string) error {
	if err := rejectValidatorArgument(context.field.key, "file", argument); err != nil {
		return err
	}
	info, err := statValidationPath(context, "file")
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("%q must be a regular file", context.field.value.String())
	}
	return nil
}

func validateOneOf(context *validationContext, argument string) error {
	value, err := stringFieldValue(context, "oneOf")
	if err != nil {
		return err
	}
	allowed := strings.Split(argument, "|")
	if argument == "" || slicesContainEmpty(allowed) {
		return fmt.Errorf("config field %q validator %q requires one or more nonempty values", context.field.key, "oneOf")
	}
	for _, candidate := range allowed {
		if value == candidate {
			return nil
		}
	}
	return fmt.Errorf("%q must be one of: %s", value, strings.Join(allowed, ", "))
}

func statValidationPath(context *validationContext, validator string) (fs.FileInfo, error) {
	if context.fileInfo != nil {
		return context.fileInfo, nil
	}
	value, err := stringFieldValue(context, validator)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(value)
	if err != nil {
		return nil, fmt.Errorf("inspect %q: %w", value, err)
	}
	context.fileInfo = info
	return info, nil
}

func stringFieldValue(context *validationContext, validator string) (string, error) {
	if context.field.value.Kind() != reflect.String {
		return "", fmt.Errorf("config field %q uses validator %q with unsupported type %s", context.field.key, validator, context.field.value.Type())
	}
	return context.field.value.String(), nil
}

func rejectValidatorArgument(key, validator, argument string) error {
	if argument != "" {
		return fmt.Errorf("config field %q validator %q does not accept an argument", key, validator)
	}
	return nil
}
