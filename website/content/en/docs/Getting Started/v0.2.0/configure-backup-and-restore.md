---
title: "Configure backup and restore"
linkTitle: "Configure backup and restore"
weight: 10
date: 2019-08-05
description: >
  Prevent loss of job history
---

Backup and restore is done by container sidecar.

### PVC

#### Create PVC

Save to file pvc.yaml:
```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: <pvc_name>
  namespace: <namesapce>
spec:
  accessModes:
  - ReadWriteOnce
  resources:
    requests:
      storage: 500Gi
```

Run command:
```bash
$ kubectl -n <namesapce> create -f pvc.yaml
```

#### Configure Jenkins CR

```yaml
apiVersion: jenkins.io/v1alpha2
kind: Jenkins
metadata:
  name: <cr_name>
  namespace: <namespace>
spec:
  master:
    securityContext:
      runAsUser: 1000
      fsGroup: 1000
    containers:
    - name: jenkins-master
      image: jenkins/jenkins:lts
    - name: backup # container responsible for backup and restore
      env:
      - name: BACKUP_DIR
        value: /backup
      - name: JENKINS_HOME
        value: /jenkins-home
      - name: BACKUP_COUNT
        value: "3" # keep only the 2 most recent backups
      image: virtuslab/jenkins-operator-backup-pvc:v0.0.6 # look at backup/pvc directory
      imagePullPolicy: IfNotPresent
      volumeMounts:
      - mountPath: /jenkins-home # Jenkins home volume
        name: jenkins-home
      - mountPath: /backup # backup volume
        name: backup
    volumes:
    - name: backup # PVC volume where backups will be stored
      persistentVolumeClaim:
        claimName: <pvc_name>
  backup:
    containerName: backup # container name is responsible for backup
    action:
      exec:
        command:
        - /home/user/bin/backup.sh # this command is invoked on "backup" container to make backup, for example /home/user/bin/backup.sh <backup_number>, <backup_number> is passed by operator
    interval: 30 # how often make backup in seconds
    makeBackupBeforePodDeletion: true # make backup before pod deletion
  restore:
    containerName: backup # container name is responsible for restore backup
    action:
      exec:
        command:
        - /home/user/bin/restore.sh # this command is invoked on "backup" container to make restore backup, for example /home/user/bin/restore.sh <backup_number>, <backup_number> is passed by operator
    #recoveryOnce: <backup_number> # if want to restore specific backup configure this field and then Jenkins will be restarted and desired backup will be restored
```
