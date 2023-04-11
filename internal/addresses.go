package internal

import (
	"net/url"
	"path"

	"github.com/sirupsen/logrus"
)

const DefaultTrainingsServer = "academy-grpc.threedots.tech:443"
const WebsiteAddress = "https://academy.threedots.tech/"

func ExerciseURL(trainingName, exerciseID string) string {
	exerciseURL, err := url.Parse(WebsiteAddress)
	if err != nil {
		logrus.WithError(err).Warn("Can't parse website URL")
	}
	exerciseURL.Path = path.Join("trainings/" + trainingName + "/exercise/" + exerciseID)

	return exerciseURL.String()
}
