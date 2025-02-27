package seedjobs

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/log"

	"github.com/go-logr/logr"
	stackerr "github.com/pkg/errors"
	"github.com/robfig/cron"
	"k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

// ValidateSeedJobs verify seed jobs configuration
func (s *SeedJobs) ValidateSeedJobs(jenkins v1alpha2.Jenkins) (bool, error) {
	valid := true

	if !s.validateIfIDIsUnique(jenkins.Spec.SeedJobs) {
		valid = false
	}

	for _, seedJob := range jenkins.Spec.SeedJobs {
		logger := s.logger.WithValues("seedJob", seedJob.ID).V(log.VWarn)

		if len(seedJob.ID) == 0 {
			logger.Info("id can't be empty")
			valid = false
		}

		if len(seedJob.RepositoryBranch) == 0 {
			logger.Info("repository branch can't be empty")
			valid = false
		}

		if len(seedJob.RepositoryURL) == 0 {
			logger.Info("repository URL branch can't be empty")
			valid = false
		}

		if len(seedJob.Targets) == 0 {
			logger.Info("targets can't be empty")
			valid = false
		}

		if _, ok := v1alpha2.AllowedJenkinsCredentialMap[string(seedJob.JenkinsCredentialType)]; !ok {
			logger.Info("unknown credential type")
			return false, nil
		}

		if (seedJob.JenkinsCredentialType == v1alpha2.BasicSSHCredentialType ||
			seedJob.JenkinsCredentialType == v1alpha2.UsernamePasswordCredentialType) && len(seedJob.CredentialID) == 0 {
			logger.Info("credential ID can't be empty")
			valid = false
		}

		// validate repository url match private key
		if strings.Contains(seedJob.RepositoryURL, "git@") && seedJob.JenkinsCredentialType == v1alpha2.NoJenkinsCredentialCredentialType {
			logger.Info("Jenkins credential must be set while using ssh repository url")
			valid = false
		}

		if seedJob.JenkinsCredentialType == v1alpha2.BasicSSHCredentialType || seedJob.JenkinsCredentialType == v1alpha2.UsernamePasswordCredentialType {
			secret := &v1.Secret{}
			namespaceName := types.NamespacedName{Namespace: jenkins.Namespace, Name: seedJob.CredentialID}
			err := s.k8sClient.Get(context.TODO(), namespaceName, secret)
			if err != nil && apierrors.IsNotFound(err) {
				logger.Info(fmt.Sprintf("required secret '%s' with Jenkins credential not found", seedJob.CredentialID))
				return false, nil
			} else if err != nil {
				return false, stackerr.WithStack(err)
			}

			if seedJob.JenkinsCredentialType == v1alpha2.BasicSSHCredentialType {
				if ok := validateBasicSSHSecret(logger, *secret); !ok {
					valid = false
				}
			}
			if seedJob.JenkinsCredentialType == v1alpha2.UsernamePasswordCredentialType {
				if ok := validateUsernamePasswordSecret(logger, *secret); !ok {
					valid = false
				}
			}
		}

		if len(seedJob.BuildPeriodically) > 0 {
			if !s.validateSchedule(seedJob, seedJob.BuildPeriodically, "buildPeriodically") {
				valid = false
			}
		}

		if len(seedJob.PollSCM) > 0 {
			if !s.validateSchedule(seedJob, seedJob.PollSCM, "pollSCM") {
				valid = false
			}
		}

		if seedJob.GitHubPushTrigger {
			if !s.validateGitHubPushTrigger(jenkins) {
				valid = false
			}
		}
	}

	return valid, nil
}

func (s *SeedJobs) validateSchedule(job v1alpha2.SeedJob, str string, key string) bool {
	_, err := cron.Parse(str)
	if err != nil {
		s.logger.V(log.VWarn).Info(fmt.Sprintf("`%s` schedule '%s' is invalid cron spec in `%s`", key, str, job.ID))
		return false
	}
	return true
}

func (s *SeedJobs) validateGitHubPushTrigger(jenkins v1alpha2.Jenkins) bool {
	exists := false
	for _, plugin := range jenkins.Spec.Master.BasePlugins {
		if plugin.Name == "github" {
			exists = true
		}
	}

	userExists := false
	for _, plugin := range jenkins.Spec.Master.Plugins {
		if plugin.Name == "github" {
			userExists = true
		}
	}

	if !exists && !userExists {
		s.logger.V(log.VWarn).Info("githubPushTrigger is set. This function requires `github` plugin installed in .Spec.Master.Plugins because seed jobs Push Trigger function needs it")
		return false
	}
	return true
}

func (s *SeedJobs) validateIfIDIsUnique(seedJobs []v1alpha2.SeedJob) bool {
	ids := map[string]bool{}
	for _, seedJob := range seedJobs {
		if _, found := ids[seedJob.ID]; found {
			s.logger.V(log.VWarn).Info(fmt.Sprintf("'%s' seed job ID is not unique", seedJob.ID))
			return false
		}
		ids[seedJob.ID] = true
	}
	return true
}

func validateBasicSSHSecret(logger logr.InfoLogger, secret v1.Secret) bool {
	valid := true
	username, exists := secret.Data[UsernameSecretKey]
	if !exists {
		logger.Info(fmt.Sprintf("required data '%s' not found in secret '%s'", UsernameSecretKey, secret.ObjectMeta.Name))
		valid = false
	}
	if len(username) == 0 {
		logger.Info(fmt.Sprintf("required data '%s' is empty in secret '%s'", UsernameSecretKey, secret.ObjectMeta.Name))
		valid = false
	}

	privateKey, exists := secret.Data[PrivateKeySecretKey]
	if !exists {
		logger.Info(fmt.Sprintf("required data '%s' not found in secret '%s'", PrivateKeySecretKey, secret.ObjectMeta.Name))
		valid = false
	}
	if len(string(privateKey)) == 0 {
		logger.Info(fmt.Sprintf("required data '%s' not found in secret '%s'", PrivateKeySecretKey, secret.ObjectMeta.Name))
		return false
	}
	if err := validatePrivateKey(string(privateKey)); err != nil {
		logger.Info(fmt.Sprintf("private key '%s' invalid in secret '%s': %s", PrivateKeySecretKey, secret.ObjectMeta.Name, err))
		valid = false
	}

	return valid
}

func validateUsernamePasswordSecret(logger logr.InfoLogger, secret v1.Secret) bool {
	valid := true
	username, exists := secret.Data[UsernameSecretKey]
	if !exists {
		logger.Info(fmt.Sprintf("required data '%s' not found in secret '%s'", UsernameSecretKey, secret.ObjectMeta.Name))
		valid = false
	}
	if len(username) == 0 {
		logger.Info(fmt.Sprintf("required data '%s' is empty in secret '%s'", UsernameSecretKey, secret.ObjectMeta.Name))
		valid = false
	}
	password, exists := secret.Data[PasswordSecretKey]
	if !exists {
		logger.Info(fmt.Sprintf("required data '%s' not found in secret '%s'", PasswordSecretKey, secret.ObjectMeta.Name))
		valid = false
	}
	if len(password) == 0 {
		logger.Info(fmt.Sprintf("required data '%s' is empty in secret '%s'", PasswordSecretKey, secret.ObjectMeta.Name))
		valid = false
	}

	return valid
}

func validatePrivateKey(privateKey string) error {
	block, _ := pem.Decode([]byte(privateKey))
	if block == nil {
		return stackerr.New("failed to decode PEM block")
	}

	priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return stackerr.WithStack(err)
	}

	err = priv.Validate()
	if err != nil {
		return stackerr.WithStack(err)
	}

	return nil
}
