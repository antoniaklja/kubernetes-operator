package notifications

import (
	"net/http"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"
)

const (
	titleText                  = "Operator reconciled."
	messageFieldName           = "Message"
	loggingLevelFieldName      = "Logging Level"
	crNameFieldName            = "CR Name"
	configurationTypeFieldName = "Configuration Type"
	namespaceFieldName         = "Namespace"
	footerContent              = "Powered by Jenkins Operator"
)

var (
	testConfigurationType = "test-configuration"
	testCrName            = "test-cr"
	testNamespace         = "default"
	testMessage           = "test-message"
	testMessageVerbose    = "detail-test-message"
	testLoggingLevel      = v1alpha2.NotificationLogLevelWarning

	client = http.Client{}
)

// StatusColor is useful for better UX
type StatusColor string

// LoggingLevel is type for selecting different logging levels
type LoggingLevel string

// Event contains event details which will be sent as a notification
type Event struct {
	Jenkins           v1alpha2.Jenkins
	ConfigurationType string
	LogLevel          v1alpha2.NotificationLogLevel
	Message           string
	MessageVerbose    string
}

/*type service interface {
	Send(event Event, notificationConfig v1alpha2.Notification) error
}

// Listen listens for incoming events and send it as notifications
func Listen(events chan Event, k8sClient k8sclient.Client) {
	for event := range events {
		logger := log.Log.WithValues("cr", event.Jenkins.Name)
		for _, notificationConfig := range event.Jenkins.Spec.Notifications {
			var err error
			var svc service

			if notificationConfig.Slack != nil {
				svc = Slack{k8sClient: k8sClient}
			} else if notificationConfig.Teams != nil {
				svc = Teams{k8sClient: k8sClient}
			} else if notificationConfig.Mailgun != nil {
				svc = MailGun{k8sClient: k8sClient}
			} else {
				logger.V(log.VWarn).Info(fmt.Sprintf("Unexpected notification `%+v`", notificationConfig))
				continue
			}

			go func(notificationConfig v1alpha2.Notification) {
				err = notify(svc, event, notificationConfig)

				if err != nil {
					if log.Debug {
						logger.Error(nil, fmt.Sprintf("%+v", errors.WithMessage(err, "failed to send notification")))
					} else {
						logger.Error(nil, fmt.Sprintf("%s", errors.WithMessage(err, "failed to send notification")))
					}
				}
			}(notificationConfig)
		}

	}
}

func notify(svc service, event Event, manifest v1alpha2.Notification) error {
	if event.LogLevel == v1alpha2.NotificationLogLevelInfo && manifest.LoggingLevel == v1alpha2.NotificationLogLevelWarning {
		return nil
	}

	return svc.Send(event, manifest)
}*/
