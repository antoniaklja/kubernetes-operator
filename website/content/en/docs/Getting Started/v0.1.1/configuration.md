---
title: "Configuration"
linkTitle: "Configuration"
weight: 2
date: 2019-08-05
description: >
  How to configure Jenkins with Operator
---

Jenkins operator uses [job-dsl][job-dsl] and [kubernetes-credentials-provider][kubernetes-credentials-provider] plugins for configuring jobs
and deploy keys.

## Prepare job definitions and pipelines

First you have to prepare pipelines and job definition in your GitHub repository using the following structure:

```
cicd/
├── jobs
│   └── build.jenkins
└── pipelines
    └── build.jenkins
```

**cicd/jobs/build.jenkins** it's a job definition:

```
#!/usr/bin/env groovy

pipelineJob('build-jenkins-operator') {
    displayName('Build jenkins-operator')

    definition {
        cpsScm {
            scm {
                git {
                    remote {
                        url('https://github.com/jenkinsci/kubernetes-operator.git')
                        credentials('jenkins-operator')
                    }
                    branches('*/master')
                }
            }
            scriptPath('cicd/pipelines/build.jenkins')
        }
    }
}
```

**cicd/jobs/build.jenkins** it's an actual Jenkins pipeline:

```
#!/usr/bin/env groovy

def label = "build-jenkins-operator-${UUID.randomUUID().toString()}"
def home = "/home/jenkins"
def workspace = "${home}/workspace/build-jenkins-operator"
def workdir = "${workspace}/src/github.com/jenkinsci/kubernetes-operator/"

podTemplate(label: label,
        containers: [
                containerTemplate(name: 'jnlp', image: 'jenkins/jnlp-slave:alpine'),
                containerTemplate(name: 'go', image: 'golang:1-alpine', command: 'cat', ttyEnabled: true),
        ],
        envVars: [
                envVar(key: 'GOPATH', value: workspace),
        ],
        ) {

    node(label) {
        dir(workdir) {
            stage('Init') {
                timeout(time: 3, unit: 'MINUTES') {
                    checkout scm
                }
                container('go') {
                    sh 'apk --no-cache --update add make git gcc libc-dev'
                }
            }

            stage('Dep') {
                container('go') {
                    sh 'make dep'
                }
            }

            stage('Test') {
                container('go') {
                    sh 'make test'
                }
            }

            stage('Build') {
                container('go') {
                    sh 'make build'
                }
            }
        }
    }
}
```

## Configure Seed Jobs

Jenkins Seed Jobs are configured using `Jenkins.spec.seedJobs` section from your custom resource manifest:

```
apiVersion: jenkins.io/v1alpha2
kind: Jenkins
metadata:
  name: example
spec:
  seedJobs:
  - id: jenkins-operator
    targets: "cicd/jobs/*.jenkins"
    description: "Jenkins Operator repository"
    repositoryBranch: master
    repositoryUrl: https://github.com/jenkinsci/kubernetes-operator.git
```

**Jenkins Operator** will automatically discover and configure all seed jobs.

You can verify if deploy keys were successfully configured in Jenkins **Credentials** tab.

![jenkins](/kubernetes-operator/img/jenkins-credentials.png)

You can verify if your pipelines were successfully configured in Jenkins Seed Job console output.

![jenkins](/kubernetes-operator/img/jenkins-seed.png)

If your GitHub repository is **private** you have to configure SSH or username/password authentication.

### SSH authentication

#### Generate SSH Keys

There are two methods of SSH private key generation:

```bash
$ openssl genrsa -out <filename> 2048
```

or

```bash
$ ssh-keygen -t rsa -b 2048
$ ssh-keygen -p -f <filename> -m pem
```

Then copy content from generated file. 

#### Public key

If you want to upload your public key to your Git server you need to extract it.

If key was generated by `openssl` then you need to type this to extract public key:

```bash
$ openssl rsa -in <filename> -pubout > <filename>.pub
```

If key was generated by `ssh-keygen` the public key content is located in <filename>.pub and there is no need to extract public key

#### Configure SSH authentication

Configure seed job like:

```
apiVersion: jenkins.io/v1alpha2
kind: Jenkins
metadata:
  name: example
spec:
  seedJobs:
  - id: jenkins-operator-ssh
    credentialType: basicSSHUserPrivateKey
    credentialID: k8s-ssh
    targets: "cicd/jobs/*.jenkins"
    description: "Jenkins Operator repository"
    repositoryBranch: master
    repositoryUrl: git@github.com:jenkinsci/kubernetes-operator.git
```

and create Kubernetes Secret(name of secret should be the same from `credentialID` field):

```
apiVersion: v1
kind: Secret
metadata:
  name: k8s-ssh
stringData:
  privateKey: |
    -----BEGIN RSA PRIVATE KEY-----
    MIIJKAIBAAKCAgEAxxDpleJjMCN5nusfW/AtBAZhx8UVVlhhhIKXvQ+dFODQIdzO
    oDXybs1zVHWOj31zqbbJnsfsVZ9Uf3p9k6xpJ3WFY9b85WasqTDN1xmSd6swD4N8
    ...
  username: github_user_name
```

### Username & password authentication

Configure seed job like:

```
apiVersion: jenkins.io/v1alpha2
kind: Jenkins
metadata:
  name: example
spec:
  seedJobs:
  - id: jenkins-operator-user-pass
    credentialType: usernamePassword
    credentialID: k8s-user-pass
    targets: "cicd/jobs/*.jenkins"
    description: "Jenkins Operator repository"
    repositoryBranch: master
    repositoryUrl: https://github.com/jenkinsci/kubernetes-operator.git
```

and create Kubernetes Secret(name of secret should be the same from `credentialID` field):

```
apiVersion: v1
kind: Secret
metadata:
  name: k8s-user-pass
stringData:
  username: github_user_name
  password: password_or_token
```

## Jenkins login credentials

The operator automatically generate Jenkins user name and password and stores it in Kubernetes secret named 
`jenkins-operator-credentials-<cr_name>` in namespace where Jenkins CR has been deployed.

If you want change it you can override the secret:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: jenkins-operator-credentials-<cr-name>
  namespace: <namespace>
data:
  user: <base64-encoded-new-username>
  password: <base64-encoded-new-password>
```

If needed **Jenkins Operator** will restart Jenkins master pod and then you can login with the new user and password 
credentials.

## Override default Jenkins container command

The default command for the Jenkins master container `jenkins/jenkins:lts` looks like:

```yaml
command:
- bash
- -c
- /var/jenkins/scripts/init.sh && /sbin/tini -s -- /usr/local/bin/jenkins.sh
```

The script`/var/jenkins/scripts/init.sh` is provided be the operator and configures init.groovy.d(creates Jenkins user) 
and installs plugins.
The `/sbin/tini -s -- /usr/local/bin/jenkins.sh` command runs the Jenkins master main process.

You can overwrite it in the following pattern:

```yaml
command:
- bash
- -c
- /var/jenkins/scripts/init.sh && <custom-code-here> && /sbin/tini -s -- /usr/local/bin/jenkins.sh
```

[job-dsl]:https://github.com/jenkinsci/job-dsl-plugin
[kubernetes-credentials-provider]:https://jenkinsci.github.io/kubernetes-credentials-provider-plugin/