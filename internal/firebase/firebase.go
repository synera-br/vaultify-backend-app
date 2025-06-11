package firebase

import (
	"context"
	"encoding/base64"
	"errors"
	"os"

	firebase "firebase.google.com/go/v4"
	"google.golang.org/api/option"
)

func InitFirebase() (*firebase.App, error) {
	ctx := context.Background()
	projectID := os.Getenv("FIREBASE_PROJECT_ID")
	if projectID == "" {
		return nil, errors.New("FIREBASE_PROJECT_ID must be set")
	}

	var opt option.ClientOption
	credsPath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	credsJSONBase64 := os.Getenv("FIREBASE_SERVICE_ACCOUNT_JSON_BASE64")

	if credsPath != "" {
		opt = option.WithCredentialsFile(credsPath)
	} else if credsJSONBase64 != "" {
		jsonKey, err := base64.StdEncoding.DecodeString(credsJSONBase64)
		if err != nil {
			return nil, errors.New("FIREBASE_SERVICE_ACCOUNT_JSON_BASE64 is not a valid base64 string")
		}
		opt = option.WithCredentialsJSON(jsonKey)
	} else {
		// For local development or environments where ADC is set up,
		// the SDK can automatically find credentials if no option is provided.
		// However, the issue explicitly asks for one of the two env vars.
		return nil, errors.New("Either GOOGLE_APPLICATION_CREDENTIALS or FIREBASE_SERVICE_ACCOUNT_JSON_BASE64 must be set")
	}

	conf := &firebase.Config{
		ProjectID: projectID,
	}

	app, err := firebase.NewApp(ctx, conf, opt)
	if err != nil {
		return nil, errors.New("Error initializing Firebase app: " + err.Error())
	}

	return app, nil
}
