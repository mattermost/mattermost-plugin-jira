package telemetry

import rudder "github.com/rudderlabs/analytics-go"

// rudderDataPlaneURL is set to the common Data Plane URL for all Mattermost Projects.
// It can be set during build time. More info in the package documentation.
const rudderDataPlaneURL = "https://pdat.matterlytics.com"

// rudderWriteKey is set during build time. More info in the package documentation.
var rudderWriteKey string

func NewRudderClient() (Client, error) {
	return NewRudderClientWithCredentials(rudderWriteKey, rudderDataPlaneURL)
}

func NewRudderClientWithCredentials(writeKey, dataPlaneURL string) (Client, error) {
	client, err := rudder.NewWithConfig(writeKey, dataPlaneURL, rudder.Config{})
	if err != nil {
		return nil, err
	}

	return &rudderWrapper{client: client}, nil
}

type rudderWrapper struct {
	client rudder.Client
}

func (r *rudderWrapper) Enqueue(t Track) error {
	err := r.client.Enqueue(rudder.Track{
		UserId:     t.UserID,
		Event:      t.Event,
		Properties: t.Properties,
	})

	if err != nil {
		return err
	}

	return nil
}

func (r *rudderWrapper) Close() error {
	err := r.client.Close()
	if err != nil {
		return err
	}
	return nil
}
