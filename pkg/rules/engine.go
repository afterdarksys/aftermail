package rules

import (
	"fmt"
	"regexp"

	"go.starlark.net/starlark"
)

// MessageContext holds the email headers and metadata accessible to Starlark
type MessageContext struct {
	Headers   map[string]string
	SenderDID string
	Actions   []string // List of actions taken by the script
}

// executeEngine runs a Starlark source script against the message context
func ExecuteEngine(scriptSource string, msg *MessageContext) error {
	// Setup the Starlark environment (The "MailScript" standard lib)
	var getHeader = starlark.NewBuiltin("get_header", func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var headerName string
		if err := starlark.UnpackArgs("get_header", args, kwargs, "name", &headerName); err != nil {
			return nil, err
		}
		
		val, ok := msg.Headers[headerName]
		if !ok {
			return starlark.String(""), nil
		}
		return starlark.String(val), nil
	})

	var discard = starlark.NewBuiltin("discard", func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		msg.Actions = append(msg.Actions, "discard")
		return starlark.None, nil
	})

	var accept = starlark.NewBuiltin("accept", func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		msg.Actions = append(msg.Actions, "accept")
		return starlark.None, nil
	})

	var fileinto = starlark.NewBuiltin("fileinto", func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var folderName string
		if err := starlark.UnpackArgs("fileinto", args, kwargs, "folder", &folderName); err != nil {
			return nil, err
		}
		msg.Actions = append(msg.Actions, fmt.Sprintf("fileinto:%s", folderName))
		return starlark.None, nil
	})

	var regexMatch = starlark.NewBuiltin("regex_match", func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var pattern, text string
		if err := starlark.UnpackArgs("regex_match", args, kwargs, "pattern", &pattern, "text", &text); err != nil {
			return nil, err
		}
		
		matched, _ := regexp.MatchString(pattern, text)
		return starlark.Bool(matched), nil
	})

	var getRecipientDID = starlark.NewBuiltin("get_recipient_did", func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		// Just a placeholder to show MailScript compatibility
		return starlark.String(msg.SenderDID), nil
	})

	var autoReply = starlark.NewBuiltin("auto_reply", func(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var replyText string
		if err := starlark.UnpackArgs("auto_reply", args, kwargs, "text", &replyText); err != nil {
			return nil, err
		}
		msg.Actions = append(msg.Actions, fmt.Sprintf("auto_reply:%s", replyText))
		return starlark.None, nil
	})

	predeclared := starlark.StringDict{
		"get_header":        getHeader,
		"discard":           discard,
		"accept":            accept,
		"fileinto":          fileinto,
		"regex_match":       regexMatch,
		"get_recipient_did": getRecipientDID,
		"auto_reply":        autoReply,
	}

	thread := &starlark.Thread{Name: "MailScriptEngine"}

	// Execute the block
	globals, err := starlark.ExecFile(thread, "script.star", scriptSource, predeclared)
	if err != nil {
		return fmt.Errorf("starlark execution failed: %w", err)
	}

	// Sieve -> Mailscript requires calling an entrypoint if defined.
	// We'll mimic the legacy by calling 'evaluate()' if it exists.
	evalFunc, ok := globals["evaluate"]
	if ok {
		_, err := starlark.Call(thread, evalFunc, nil, nil)
		if err != nil {
			return fmt.Errorf("failed calling evaluate(): %w", err)
		}
	} else {
		// If evaluate isn't defined, we just assume the root level script did the work.
		if len(msg.Actions) == 0 {
			msg.Actions = append(msg.Actions, "accept")
		}
	}

	return nil
}
