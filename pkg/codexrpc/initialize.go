package codexrpc

type InitializeOptions struct {
	ClientName                string
	ClientTitle               string
	ClientVersion             string
	OptOutNotificationMethods []string
	Extra                     map[string]any
}

func BuildInitializeParams(opts InitializeOptions) map[string]any {
	params := map[string]any{}
	if opts.Extra != nil {
		for k, v := range opts.Extra {
			params[k] = v
		}
	}

	clientInfo := map[string]any{}
	if opts.ClientName != "" {
		clientInfo["name"] = opts.ClientName
	}
	if opts.ClientTitle != "" {
		clientInfo["title"] = opts.ClientTitle
	}
	if opts.ClientVersion != "" {
		clientInfo["version"] = opts.ClientVersion
	}
	if len(clientInfo) > 0 {
		params["clientInfo"] = clientInfo
	}

	if len(opts.OptOutNotificationMethods) > 0 {
		capabilities := map[string]any{}
		if existing, ok := params["capabilities"].(map[string]any); ok {
			for k, v := range existing {
				capabilities[k] = v
			}
		}
		capabilities["optOutNotificationMethods"] = opts.OptOutNotificationMethods
		params["capabilities"] = capabilities
	}

	return params
}
