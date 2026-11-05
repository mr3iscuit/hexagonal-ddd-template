package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"
)

// decodeRequest reads a JSON body into RequestDTO. It checks only the request's
// shape, because business rules stay in the domain. Its messages are written
// here and never copied from the decoder, which can quote client input.
func decodeRequest[RequestDTO any](responseWriter http.ResponseWriter, request *http.Request, fields []string) (RequestDTO, error) {
	var requestDTO RequestDTO
	if err := checkContentType(request); err != nil {
		return requestDTO, err
	}
	content, err := io.ReadAll(http.MaxBytesReader(responseWriter, request.Body, maximumRequestBodyBytes))
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			return requestDTO, NewAPIError(http.StatusRequestEntityTooLarge, codeRequestTooLarge, requestTooLargeMessage)
		}
		if cancelled := request.Context().Err(); cancelled != nil {
			return requestDTO, cancelled
		}
		return requestDTO, malformedRequest()
	}
	// The decoder would replace broken text without saying so, which would
	// store something the client never sent.
	if !utf8.Valid(content) {
		return requestDTO, malformedRequest()
	}
	var properties map[string]json.RawMessage
	if err := json.Unmarshal(content, &properties); err != nil || hasLoneSurrogate(content) {
		return requestDTO, malformedRequest()
	}
	fieldErrors := checkProperties(properties, fields)
	if hasRepeatedProperty(content) {
		fieldErrors = append(fieldErrors, FieldError{Message: repeatedPropertyMessage, Location: bodyLocation})
	}
	if len(fieldErrors) > 0 {
		return requestDTO, invalidRequest(fieldErrors...)
	}
	if err := json.Unmarshal(content, &requestDTO); err != nil {
		var wrongType *json.UnmarshalTypeError
		if errors.As(err, &wrongType) {
			return requestDTO, invalidRequest(FieldError{Message: fmt.Sprintf(wrongTypeFormat, wrongType.Field), Location: bodyLocation + locationSeparator + wrongType.Field})
		}
		return requestDTO, malformedRequest()
	}
	return requestDTO, nil
}

// withoutBody guards an endpoint that takes no body, so a client's mistake is
// reported instead of ignored.
func withoutBody(endpoint func(http.ResponseWriter, *http.Request) error) func(http.ResponseWriter, *http.Request) error {
	return func(responseWriter http.ResponseWriter, request *http.Request) error {
		if _, err := io.ReadAll(http.MaxBytesReader(responseWriter, request.Body, 0)); err != nil {
			return invalidRequest(FieldError{Message: unexpectedBodyMessage, Location: bodyLocation})
		}
		return endpoint(responseWriter, request)
	}
}

// hasRepeatedProperty finds a key that appears twice, because a map keeps only
// its last value. It needs valid JSON, so none of its reads can fail.
func hasRepeatedProperty(content []byte) bool {
	decoder := json.NewDecoder(bytes.NewReader(content))
	if opening, _ := decoder.Token(); opening != json.Delim('{') {
		return false
	}
	seen := map[json.Token]bool{}
	for decoder.More() {
		name, _ := decoder.Token()
		if seen[name] {
			return true
		}
		seen[name] = true
		var value json.RawMessage
		_ = decoder.Decode(&value)
	}
	return false
}

// hasLoneSurrogate finds an escape such as \ud800 that has no partner. It needs
// valid JSON, where a backslash can only appear inside a string.
func hasLoneSurrogate(content []byte) bool {
	for position := 0; position < len(content); position++ {
		if content[position] != '\\' {
			continue
		}
		first, isUnicodeEscape := escapedCodeUnit(content[position:])
		if !isUnicodeEscape || !utf16.IsSurrogate(first) {
			// This skips the escaped character, so \\ud800 stays plain text.
			position++
			continue
		}
		position += unicodeEscapeLength
		second, _ := escapedCodeUnit(content[position:])
		if utf16.DecodeRune(first, second) == unicode.ReplacementChar {
			return true
		}
		position += unicodeEscapeLength - 1
	}
	return false
}

func escapedCodeUnit(text []byte) (rune, bool) {
	if len(text) < unicodeEscapeLength || text[0] != '\\' || text[1] != 'u' {
		return 0, false
	}
	codeUnit, err := strconv.ParseUint(string(text[unicodeEscapePrefixLength:unicodeEscapeLength]), hexadecimalBase, codeUnitBits)
	return rune(codeUnit), err == nil
}

// archivedFilter reads the list's only query parameter. It parses the query
// itself, because URL.Query drops any pair it can't read. Unknown names stay out
// of the location, because they are client input.
func archivedFilter(request *http.Request) (bool, error) {
	values, err := url.ParseQuery(request.URL.RawQuery)
	if err != nil {
		return false, invalidRequest(FieldError{Message: malformedQueryMessage, Location: queryLocation})
	}
	for name := range values {
		if name != archivedParameter {
			return false, invalidRequest(FieldError{Message: unexpectedQueryMessage, Location: queryLocation})
		}
	}
	if !values.Has(archivedParameter) {
		return false, nil
	}
	filter := values[archivedParameter]
	if len(filter) == 1 && (filter[0] == archivedTrue || filter[0] == archivedFalse) {
		return filter[0] == archivedTrue, nil
	}
	return false, invalidRequest(FieldError{Message: invalidArchivedMessage, Location: queryLocation + locationSeparator + archivedParameter})
}

// checkContentType accepts JSON, and treats a missing content type as JSON.
func checkContentType(request *http.Request) error {
	contentType := request.Header.Get("Content-Type")
	if contentType == "" {
		return nil
	}
	mediaType, parameters, err := mime.ParseMediaType(contentType)
	charset, hasCharset := parameters[charsetParameter]
	if err == nil && mediaType == "application/json" && (!hasCharset || strings.EqualFold(charset, utf8Charset)) {
		return nil
	}
	return NewAPIError(http.StatusUnsupportedMediaType, codeUnsupportedMediaType, validationFailedMessage).
		withFieldErrors(FieldError{Message: wrongContentTypeMessage, Location: bodyLocation})
}

func checkProperties(properties map[string]json.RawMessage, fields []string) []FieldError {
	var fieldErrors []FieldError
	for _, field := range fields {
		if value, present := properties[field]; !present || string(value) == jsonNull {
			fieldErrors = append(fieldErrors, FieldError{Message: fmt.Sprintf(missingPropertyFormat, field), Location: bodyLocation})
		}
	}
	// The location stays generic, because an unknown name is client input.
	for field := range properties {
		if !slices.Contains(fields, field) {
			fieldErrors = append(fieldErrors, FieldError{Message: unexpectedFieldMessage, Location: bodyLocation})
		}
	}
	return fieldErrors
}

func malformedRequest() APIError {
	return NewAPIError(http.StatusBadRequest, codeMalformedRequest, validationFailedMessage).
		withFieldErrors(FieldError{Message: malformedBodyMessage, Location: bodyLocation})
}

func invalidRequest(fieldErrors ...FieldError) APIError {
	return NewAPIError(http.StatusUnprocessableEntity, codeValidationFailed, validationFailedMessage).
		withFieldErrors(fieldErrors...)
}
